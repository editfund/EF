// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package doctor

import (
	"context"

	actions_model "forgejo.org/models/actions"
	activities_model "forgejo.org/models/activities"
	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/migrations"
	org_model "forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
)

type consistencyCheck struct {
	Name         string
	Counter      func(context.Context) (int64, error)
	Fixer        func(context.Context) (int64, error)
	FixedMessage string
}

func (c *consistencyCheck) Run(ctx context.Context, logger log.Logger, autofix bool) error {
	count, err := c.Counter(ctx)
	if err != nil {
		logger.Critical("Error: %v whilst counting %s", err, c.Name)
		return err
	}
	if count > 0 {
		if autofix {
			var fixed int64
			if fixed, err = c.Fixer(ctx); err != nil {
				logger.Critical("Error: %v whilst fixing %s", err, c.Name)
				return err
			}

			prompt := "Deleted"
			if c.FixedMessage != "" {
				prompt = c.FixedMessage
			}

			if fixed < 0 {
				logger.Info(prompt+" %d %s", count, c.Name)
			} else {
				logger.Info(prompt+" %d/%d %s", fixed, count, c.Name)
			}
		} else {
			logger.Warn("Found %d %s", count, c.Name)
		}
	}
	return nil
}

func asFixer(fn func(ctx context.Context) error) func(ctx context.Context) (int64, error) {
	return func(ctx context.Context) (int64, error) {
		err := fn(ctx)
		return -1, err
	}
}

func genericOrphanCheck(name, subject, refobject, joincond string) consistencyCheck {
	return consistencyCheck{
		Name: name,
		Counter: func(ctx context.Context) (int64, error) {
			return db.CountOrphanedObjects(ctx, subject, refobject, joincond)
		},
		Fixer: func(ctx context.Context) (int64, error) {
			err := db.DeleteOrphanedObjects(ctx, subject, refobject, joincond)
			return -1, err
		},
	}
}

func checkDBConsistency(ctx context.Context, logger log.Logger, autofix bool) error {
	// make sure DB version is up-to-date
	ensureUpToDateWrapper := func(e db.Engine) error {
		engine, err := db.GetMasterEngine(e)
		if err != nil {
			return err
		}
		return migrations.EnsureUpToDate(engine)
	}
	if err := db.InitEngineWithMigration(ctx, ensureUpToDateWrapper); err != nil {
		logger.Critical("Model version on the database does not match the current Gitea version. Model consistency will not be checked until the database is upgraded")
		return err
	}

	consistencyChecks := []consistencyCheck{
		{
			// find labels without existing repo or org
			Name:    "Orphaned Labels without existing repository or organisation",
			Counter: issues_model.CountOrphanedLabels,
			Fixer:   asFixer(issues_model.DeleteOrphanedLabels),
		},
		{
			// find IssueLabels without existing label
			Name:    "Orphaned Issue Labels without existing label",
			Counter: issues_model.CountOrphanedIssueLabels,
			Fixer:   asFixer(issues_model.DeleteOrphanedIssueLabels),
		},
		{
			// find issues without existing repository
			Name:    "Orphaned Issues without existing repository",
			Counter: issues_model.CountOrphanedIssues,
			Fixer:   asFixer(issues_model.DeleteOrphanedIssues),
		},
		// find releases without existing repository
		genericOrphanCheck("Orphaned Releases without existing repository",
			"release", "repository", "`release`.repo_id=repository.id"),
		// find pulls without existing issues
		genericOrphanCheck("Orphaned PullRequests without existing issue",
			"pull_request", "issue", "pull_request.issue_id=issue.id"),
		// find pull requests without base repository
		genericOrphanCheck("Pull request entries without existing base repository",
			"pull_request", "repository", "pull_request.base_repo_id=repository.id"),
		// find tracked times without existing issues/pulls
		genericOrphanCheck("Orphaned TrackedTimes without existing issue",
			"tracked_time", "issue", "tracked_time.issue_id=issue.id"),
		// find attachments without existing issues or releases
		{
			Name:    "Orphaned Attachments without existing issues or releases",
			Counter: repo_model.CountOrphanedAttachments,
			Fixer:   asFixer(repo_model.DeleteOrphanedAttachments),
		},
		// find null archived repositories
		{
			Name:         "Repositories with is_archived IS NULL",
			Counter:      repo_model.CountNullArchivedRepository,
			Fixer:        repo_model.FixNullArchivedRepository,
			FixedMessage: "Fixed",
		},
		// find label comments with empty labels
		{
			Name:         "Label comments with empty labels",
			Counter:      issues_model.CountCommentTypeLabelWithEmptyLabel,
			Fixer:        issues_model.FixCommentTypeLabelWithEmptyLabel,
			FixedMessage: "Fixed",
		},
		// find label comments with labels from outside the repository
		{
			Name:         "Label comments with labels from outside the repository",
			Counter:      issues_model.CountCommentTypeLabelWithOutsideLabels,
			Fixer:        issues_model.FixCommentTypeLabelWithOutsideLabels,
			FixedMessage: "Removed",
		},
		// find issue_label with labels from outside the repository
		{
			Name:         "IssueLabels with Labels from outside the repository",
			Counter:      issues_model.CountIssueLabelWithOutsideLabels,
			Fixer:        issues_model.FixIssueLabelWithOutsideLabels,
			FixedMessage: "Removed",
		},
		{
			Name:         "Action with created_unix set as an empty string",
			Counter:      activities_model.CountActionCreatedUnixString,
			Fixer:        activities_model.FixActionCreatedUnixString,
			FixedMessage: "Set to zero",
		},
		{
			Name:         "Action Runners without existing owner",
			Counter:      actions_model.CountRunnersWithoutBelongingOwner,
			Fixer:        actions_model.FixRunnersWithoutBelongingOwner,
			FixedMessage: "Removed",
		},
		{
			Name:         "Action Runners without existing repository",
			Counter:      actions_model.CountRunnersWithoutBelongingRepo,
			Fixer:        actions_model.FixRunnersWithoutBelongingRepo,
			FixedMessage: "Removed",
		},
		{
			Name:         "Topics with empty repository count",
			Counter:      repo_model.CountOrphanedTopics,
			Fixer:        repo_model.DeleteOrphanedTopics,
			FixedMessage: "Removed",
		},
		{
			Name:         "Orphaned OAuth2Application without existing User",
			Counter:      auth_model.CountOrphanedOAuth2Applications,
			Fixer:        auth_model.DeleteOrphanedOAuth2Applications,
			FixedMessage: "Removed",
		},
		{
			Name:         "Owner teams with no admin access",
			Counter:      org_model.CountInconsistentOwnerTeams,
			Fixer:        org_model.FixInconsistentOwnerTeams,
			FixedMessage: "Fixed",
		},
	}

	// TODO: function to recalc all counters

	if setting.Database.Type.IsPostgreSQL() {
		consistencyChecks = append(consistencyChecks, consistencyCheck{
			Name:         "Sequence values",
			Counter:      db.CountBadSequences,
			Fixer:        asFixer(db.FixBadSequences),
			FixedMessage: "Updated",
		})
	}

	consistencyChecks = append(consistencyChecks,
		// find protected branches without existing repository
		genericOrphanCheck("Protected Branches without existing repository",
			"protected_branch", "repository", "protected_branch.repo_id=repository.id"),
		// find branches without existing repository
		genericOrphanCheck("Branches without existing repository",
			"branch", "repository", "branch.repo_id=repository.id"),
		// find LFS locks without existing repository
		genericOrphanCheck("LFS locks without existing repository",
			"lfs_lock", "repository", "lfs_lock.repo_id=repository.id"),
		// find collaborations without users
		genericOrphanCheck("Collaborations without existing user",
			"collaboration", "user", "collaboration.user_id=`user`.id"),
		// find collaborations without repository
		genericOrphanCheck("Collaborations without existing repository",
			"collaboration", "repository", "collaboration.repo_id=repository.id"),
		// find access without users
		genericOrphanCheck("Access entries without existing user",
			"access", "user", "access.user_id=`user`.id"),
		// find access without repository
		genericOrphanCheck("Access entries without existing repository",
			"access", "repository", "access.repo_id=repository.id"),
		// find action without repository
		genericOrphanCheck("Action entries without existing repository",
			"action", "repository", "action.repo_id=repository.id"),
		// find action without user
		genericOrphanCheck("Action entries without existing user",
			"action", "user", "action.act_user_id=`user`.id"),
		// find OAuth2Grant without existing user
		genericOrphanCheck("Orphaned OAuth2Grant without existing User",
			"oauth2_grant", "user", "oauth2_grant.user_id=`user`.id"),
		// find OAuth2AuthorizationCode without existing OAuth2Grant
		genericOrphanCheck("Orphaned OAuth2AuthorizationCode without existing OAuth2Grant",
			"oauth2_authorization_code", "oauth2_grant", "oauth2_authorization_code.grant_id=oauth2_grant.id"),
		// find stopwatches without existing user
		genericOrphanCheck("Orphaned Stopwatches without existing User",
			"stopwatch", "user", "stopwatch.user_id=`user`.id"),
		// find stopwatches without existing issue
		genericOrphanCheck("Orphaned Stopwatches without existing Issue",
			"stopwatch", "issue", "stopwatch.issue_id=`issue`.id"),
		// find redirects without existing user.
		genericOrphanCheck("Orphaned Redirects without existing redirect user",
			"user_redirect", "user", "user_redirect.redirect_user_id=`user`.id"),
		// find archive download count without existing release
		genericOrphanCheck("Archive download count without existing Release",
			"repo_archive_download_count", "release", "repo_archive_download_count.release_id=release.id"),
		// find authorization tokens without existing user
		genericOrphanCheck("Authorization token without existing User",
			"forgejo_auth_token", "user", "forgejo_auth_token.uid=`user`.id"),
		// find two_factor without existing user
		genericOrphanCheck("Orphaned TwoFactor without existing User",
			"two_factor", "user", "`two_factor`.uid=`user`.id"),
	)

	for _, c := range consistencyChecks {
		if err := c.Run(ctx, logger, autofix); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	Register(&Check{
		Title:     "Check consistency of database",
		Name:      "check-db-consistency",
		IsDefault: false,
		Run:       checkDBConsistency,
		Priority:  3,
	})
}
