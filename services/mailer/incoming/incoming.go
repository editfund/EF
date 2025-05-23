// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package incoming

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	net_mail "net/mail"
	"regexp"
	"slices"
	"strings"
	"time"

	"forgejo.org/modules/log"
	"forgejo.org/modules/process"
	"forgejo.org/modules/setting"
	"forgejo.org/services/mailer/token"

	"code.forgejo.org/forgejo/reply"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/jhillyerd/enmime/v2"
)

var (
	addressTokenRegex   *regexp.Regexp
	referenceTokenRegex *regexp.Regexp
)

func Init(ctx context.Context) error {
	if !setting.IncomingEmail.Enabled {
		return nil
	}

	var err error
	addressTokenRegex, err = regexp.Compile(
		fmt.Sprintf(
			`\A%s\z`,
			strings.Replace(regexp.QuoteMeta(setting.IncomingEmail.ReplyToAddress), regexp.QuoteMeta(setting.IncomingEmail.TokenPlaceholder), "(.+)", 1),
		),
	)
	if err != nil {
		return err
	}
	referenceTokenRegex, err = regexp.Compile(fmt.Sprintf(`\Areply-(.+)@%s\z`, regexp.QuoteMeta(setting.Domain)))
	if err != nil {
		return err
	}

	go func() {
		ctx, _, finished := process.GetManager().AddTypedContext(ctx, "Incoming Email", process.SystemProcessType, true)
		defer finished()

		// This background job processes incoming emails. It uses the IMAP IDLE command to get notified about incoming emails.
		// The following loop restarts the processing logic after errors until ctx indicates to stop.

		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := processIncomingEmails(ctx); err != nil {
					log.Error("Error while processing incoming emails: %v", err)
				}
				select {
				case <-ctx.Done():
					return
				case <-time.NewTimer(10 * time.Second).C:
				}
			}
		}
	}()

	return nil
}

// processIncomingEmails is the "main" method with the wait/process loop
func processIncomingEmails(ctx context.Context) error {
	server := fmt.Sprintf("%s:%d", setting.IncomingEmail.Host, setting.IncomingEmail.Port)

	var c *client.Client
	var err error
	if setting.IncomingEmail.UseTLS {
		c, err = client.DialTLS(server, &tls.Config{InsecureSkipVerify: setting.IncomingEmail.SkipTLSVerify})
	} else {
		c, err = client.Dial(server)
	}
	if err != nil {
		return fmt.Errorf("could not connect to server '%s': %w", server, err)
	}

	if err := c.Login(setting.IncomingEmail.Username, setting.IncomingEmail.Password); err != nil {
		return fmt.Errorf("could not login: %w", err)
	}
	defer func() {
		if err := c.Logout(); err != nil {
			log.Error("Logout from incoming email server failed: %v", err)
		}
	}()

	if _, err := c.Select(setting.IncomingEmail.Mailbox, false); err != nil {
		return fmt.Errorf("selecting box '%s' failed: %w", setting.IncomingEmail.Mailbox, err)
	}

	// The following loop processes messages. If there are no messages available, IMAP IDLE is used to wait for new messages.
	// This process is repeated until an IMAP error occurs or ctx indicates to stop.

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := processMessages(ctx, c); err != nil {
				return fmt.Errorf("could not process messages: %w", err)
			}
			if err := waitForUpdates(ctx, c); err != nil {
				return fmt.Errorf("wait for updates failed: %w", err)
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.NewTimer(time.Second).C:
			}
		}
	}
}

// waitForUpdates uses IMAP IDLE to wait for new emails
func waitForUpdates(ctx context.Context, c *client.Client) error {
	updates := make(chan client.Update, 1)

	c.Updates = updates
	defer func() {
		c.Updates = nil
	}()

	errs := make(chan error, 1)
	stop := make(chan struct{})
	go func() {
		errs <- c.Idle(stop, nil)
	}()

	stopped := false
	for {
		select {
		case update := <-updates:
			switch update.(type) {
			case *client.MailboxUpdate:
				if !stopped {
					close(stop)
					stopped = true
				}
			default:
			}
		case err := <-errs:
			if err != nil {
				return fmt.Errorf("imap idle failed: %w", err)
			}
			return nil
		case <-ctx.Done():
			return nil
		}
	}
}

// processMessages searches unread mails and processes them.
func processMessages(ctx context.Context, c *client.Client) error {
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	criteria.Smaller = setting.IncomingEmail.MaximumMessageSize
	ids, err := c.Search(criteria)
	if err != nil {
		return fmt.Errorf("imap search failed: %w", err)
	}

	if len(ids) == 0 {
		return nil
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)
	messages := make(chan *imap.Message, 10)

	section := &imap.BodySectionName{}

	errs := make(chan error, 1)
	go func() {
		errs <- c.Fetch(
			seqset,
			[]imap.FetchItem{section.FetchItem()},
			messages,
		)
	}()

	handledSet := new(imap.SeqSet)
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case msg, ok := <-messages:
			if !ok {
				if setting.IncomingEmail.DeleteHandledMessage && !handledSet.Empty() {
					if err := c.Store(
						handledSet,
						imap.FormatFlagsOp(imap.AddFlags, true),
						[]any{imap.DeletedFlag},
						nil,
					); err != nil {
						return fmt.Errorf("imap store failed: %w", err)
					}

					if err := c.Expunge(nil); err != nil {
						return fmt.Errorf("imap expunge failed: %w", err)
					}
				}
				return nil
			}

			err := func() error {
				if isAlreadyHandled(handledSet, msg) {
					log.Debug("Skipping already handled message")
					return nil
				}

				r := msg.GetBody(section)
				if r == nil {
					return fmt.Errorf("could not get body from message: %w", err)
				}

				env, err := enmime.ReadEnvelope(r)
				if err != nil {
					return fmt.Errorf("could not read envelope: %w", err)
				}

				if isAutomaticReply(env) {
					log.Debug("Skipping automatic email reply")
					return nil
				}

				t := searchTokenInHeaders(env)
				if t == "" {
					log.Debug("Incoming email token not found in headers")
					return nil
				}

				handlerType, user, payload, err := token.ExtractToken(ctx, t)
				if err != nil {
					if _, ok := err.(*token.ErrToken); ok {
						log.Info("Invalid incoming email token: %v", err)
						return nil
					}
					return err
				}

				handler, ok := handlers[handlerType]
				if !ok {
					return fmt.Errorf("unexpected handler type: %v", handlerType)
				}

				content := getContentFromMailReader(env)

				if err := handler.Handle(ctx, content, user, payload); err != nil {
					return fmt.Errorf("could not handle message: %w", err)
				}

				handledSet.AddNum(msg.SeqNum)

				return nil
			}()
			if err != nil {
				log.Error("Error while processing incoming email[%v]: %v", msg.Uid, err)
			}
		}
	}

	if err := <-errs; err != nil {
		return fmt.Errorf("imap fetch failed: %w", err)
	}

	return nil
}

// isAlreadyHandled tests if the message was already handled
func isAlreadyHandled(handledSet *imap.SeqSet, msg *imap.Message) bool {
	return handledSet.Contains(msg.SeqNum)
}

// isAutomaticReply tests if the headers indicate an automatic reply
func isAutomaticReply(env *enmime.Envelope) bool {
	autoSubmitted := env.GetHeader("Auto-Submitted")
	if autoSubmitted != "" && autoSubmitted != "no" {
		return true
	}
	autoReply := env.GetHeader("X-Autoreply")
	if autoReply == "yes" {
		return true
	}
	precedence := env.GetHeader("Precedence")
	if precedence == "auto_reply" {
		return true
	}
	autoRespond := env.GetHeader("X-Autorespond")
	return autoRespond != ""
}

// searchTokenInHeaders looks for the token in To, Delivered-To and References
func searchTokenInHeaders(env *enmime.Envelope) string {
	if addressTokenRegex != nil {
		to, _ := env.AddressList("To")

		token := searchTokenInAddresses(to)
		if token != "" {
			return token
		}

		deliveredTo, _ := env.AddressList("Delivered-To")

		token = searchTokenInAddresses(deliveredTo)
		if token != "" {
			return token
		}
	}

	references := env.GetHeader("References")
	for {
		begin := strings.IndexByte(references, '<')
		if begin == -1 {
			break
		}
		begin++

		end := strings.IndexByte(references, '>')
		if end == -1 || begin > end {
			break
		}

		match := referenceTokenRegex.FindStringSubmatch(references[begin:end])
		if len(match) == 2 {
			return match[1]
		}

		references = references[end+1:]
	}

	return ""
}

// searchTokenInAddresses looks for the token in an address
func searchTokenInAddresses(addresses []*net_mail.Address) string {
	for _, address := range addresses {
		match := addressTokenRegex.FindStringSubmatch(address.Address)
		if len(match) != 2 {
			continue
		}

		return match[1]
	}

	return ""
}

type MailContent struct {
	Content     string
	Attachments []*Attachment
}

type Attachment struct {
	Name    string
	Content []byte
}

// getContentFromMailReader grabs the plain content and the attachments from the mail.
// A potential reply/signature gets stripped from the content.
func getContentFromMailReader(env *enmime.Envelope) *MailContent {
	// get attachments
	attachments := make([]*Attachment, 0, len(env.Attachments))
	for _, attachment := range env.Attachments {
		attachments = append(attachments, &Attachment{
			Name:    constructFilename(attachment),
			Content: attachment.Content,
		})
	}
	// get inlines
	inlineAttachments := make([]*Attachment, 0, len(env.Inlines))
	for _, inline := range env.Inlines {
		if inline.FileName != "" && inline.ContentType != "text/plain" {
			inlineAttachments = append(inlineAttachments, &Attachment{
				Name:    constructFilename(inline),
				Content: inline.Content,
			})
		}
	}
	// get other parts (mostly multipart/related files, these are for example embedded images in an html mail)
	otherParts := make([]*Attachment, 0, len(env.Inlines))
	for _, otherPart := range env.OtherParts {
		otherParts = append(otherParts, &Attachment{
			Name:    constructFilename(otherPart),
			Content: otherPart.Content,
		})
	}

	return &MailContent{
		Content:     reply.FromText(env.Text),
		Attachments: slices.Concat(attachments, inlineAttachments, otherParts),
	}
}

// constructFilename interprets the mime part as an (inline) attachment and returns its filename
// If no filename is given it guesses a sensible filename for it based on the filetype.
func constructFilename(part *enmime.Part) string {
	if strings.TrimSpace(part.FileName) != "" {
		return part.FileName
	}

	filenameWOExtension := "unnamed_file"
	if strings.TrimSpace(part.ContentID) != "" {
		filenameWOExtension = part.ContentID
	}

	fileExtension := ".unknown"
	mimeExtensions, err := mime.ExtensionsByType(part.ContentType)
	if err == nil && len(mimeExtensions) != 0 {
		// just use the first one we find
		fileExtension = mimeExtensions[0]
	}
	return filenameWOExtension + fileExtension
}
