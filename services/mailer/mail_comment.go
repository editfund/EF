// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mailer

import (
	"context"

	activities_model "forgejo.org/models/activities"
	issues_model "forgejo.org/models/issues"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/container"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
)

// MailParticipantsComment sends new comment emails to repository watchers and mentioned people.
func MailParticipantsComment(ctx context.Context, c *issues_model.Comment, opType activities_model.ActionType, issue *issues_model.Issue, mentions []*user_model.User) error {
	if setting.MailService == nil {
		// No mail service configured
		return nil
	}

	content := c.Content
	if c.Type == issues_model.CommentTypePullRequestPush {
		content = ""
	}
	if err := mailIssueCommentToParticipants(
		&mailCommentContext{
			Context:    ctx,
			Issue:      issue,
			Doer:       c.Poster,
			ActionType: opType,
			Content:    content,
			Comment:    c,
		}, mentions); err != nil {
		log.Error("mailIssueCommentToParticipants: %v", err)
	}
	return nil
}

// MailMentionsComment sends email to users mentioned in a code comment
func MailMentionsComment(ctx context.Context, pr *issues_model.PullRequest, c *issues_model.Comment, mentions []*user_model.User) (err error) {
	if setting.MailService == nil {
		// No mail service configured
		return nil
	}

	visited := make(container.Set[int64], len(mentions)+1)
	visited.Add(c.Poster.ID)
	if err = mailIssueCommentBatch(
		&mailCommentContext{
			Context:    ctx,
			Issue:      pr.Issue,
			Doer:       c.Poster,
			ActionType: activities_model.ActionCommentPull,
			Content:    c.Content,
			Comment:    c,
		}, mentions, visited, true); err != nil {
		log.Error("mailIssueCommentBatch: %v", err)
	}
	return nil
}
