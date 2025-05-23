// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"testing"

	"forgejo.org/modules/git"
	api "forgejo.org/modules/structs"
	webhook_module "forgejo.org/modules/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectMatched(t *testing.T) {
	testCases := []struct {
		desc           string
		commit         *git.Commit
		triggeredEvent webhook_module.HookEventType
		payload        api.Payloader
		yamlOn         string
		expected       bool
	}{
		{
			desc:           "HookEventCreate(create) matches GithubEventCreate(create)",
			triggeredEvent: webhook_module.HookEventCreate,
			payload:        nil,
			yamlOn:         "on: create",
			expected:       true,
		},
		{
			desc:           "HookEventIssues(issues) `opened` action matches GithubEventIssues(issues)",
			triggeredEvent: webhook_module.HookEventIssues,
			payload:        &api.IssuePayload{Action: api.HookIssueOpened},
			yamlOn:         "on: issues",
			expected:       true,
		},
		{
			desc:           "HookEventIssueComment(issue_comment) `created` action matches GithubEventIssueComment(issue_comment)",
			triggeredEvent: webhook_module.HookEventIssueComment,
			payload:        &api.IssueCommentPayload{Action: api.HookIssueCommentCreated},
			yamlOn:         "on:\n  issue_comment:\n    types: [created]",
			expected:       true,
		},

		{
			desc:           "HookEventIssues(issues) `milestoned` action matches GithubEventIssues(issues)",
			triggeredEvent: webhook_module.HookEventIssues,
			payload:        &api.IssuePayload{Action: api.HookIssueMilestoned},
			yamlOn:         "on: issues",
			expected:       true,
		},

		{
			desc:           "HookEventPullRequestSync(pull_request_sync) matches GithubEventPullRequest(pull_request)",
			triggeredEvent: webhook_module.HookEventPullRequestSync,
			payload:        &api.PullRequestPayload{Action: api.HookIssueSynchronized},
			yamlOn:         "on: pull_request",
			expected:       true,
		},
		{
			desc:           "HookEventPullRequest(pull_request) `label_updated` action doesn't match GithubEventPullRequest(pull_request) with no activity type",
			triggeredEvent: webhook_module.HookEventPullRequest,
			payload:        &api.PullRequestPayload{Action: api.HookIssueLabelUpdated},
			yamlOn:         "on: pull_request",
			expected:       false,
		},
		{
			desc:           "HookEventPullRequest(pull_request) `closed` action doesn't match GithubEventPullRequest(pull_request) with no activity type",
			triggeredEvent: webhook_module.HookEventPullRequest,
			payload:        &api.PullRequestPayload{Action: api.HookIssueClosed},
			yamlOn:         "on: pull_request",
			expected:       false,
		},
		{
			desc:           "HookEventPullRequest(pull_request) `closed` action doesn't match GithubEventPullRequest(pull_request) with branches",
			triggeredEvent: webhook_module.HookEventPullRequest,
			payload: &api.PullRequestPayload{
				Action: api.HookIssueClosed,
				PullRequest: &api.PullRequest{
					Base: &api.PRBranchInfo{},
				},
			},
			yamlOn:   "on:\n  pull_request:\n    branches: [main]",
			expected: false,
		},
		{
			desc:           "HookEventPullRequest(pull_request) `label_updated` action matches GithubEventPullRequest(pull_request) with `label` activity type",
			triggeredEvent: webhook_module.HookEventPullRequest,
			payload:        &api.PullRequestPayload{Action: api.HookIssueLabelUpdated},
			yamlOn:         "on:\n  pull_request:\n    types: [labeled]",
			expected:       true,
		},
		{
			desc:           "HookEventPullRequestReviewComment(pull_request_review_comment) matches GithubEventPullRequestReviewComment(pull_request_review_comment)",
			triggeredEvent: webhook_module.HookEventPullRequestReviewComment,
			payload:        &api.PullRequestPayload{Action: api.HookIssueReviewed},
			yamlOn:         "on:\n  pull_request_review_comment:\n    types: [created]",
			expected:       true,
		},
		{
			desc:           "HookEventPullRequestReviewRejected(pull_request_review_rejected) doesn't match GithubEventPullRequestReview(pull_request_review) with `dismissed` activity type (we don't support `dismissed` at present)",
			triggeredEvent: webhook_module.HookEventPullRequestReviewRejected,
			payload:        &api.PullRequestPayload{Action: api.HookIssueReviewed},
			yamlOn:         "on:\n  pull_request_review:\n    types: [dismissed]",
			expected:       false,
		},
		{
			desc:           "HookEventRelease(release) `published` action matches GithubEventRelease(release) with `published` activity type",
			triggeredEvent: webhook_module.HookEventRelease,
			payload:        &api.ReleasePayload{Action: api.HookReleasePublished},
			yamlOn:         "on:\n  release:\n    types: [published]",
			expected:       true,
		},
		{
			desc:           "HookEventRelease(updated) `updated` action matches GithubEventRelease(edited) with `edited` activity type",
			triggeredEvent: webhook_module.HookEventRelease,
			payload:        &api.ReleasePayload{Action: api.HookReleaseUpdated},
			yamlOn:         "on:\n  release:\n    types: [edited]",
			expected:       true,
		},

		{
			desc:           "HookEventPackage(package) `created` action doesn't match GithubEventRegistryPackage(registry_package) with `updated` activity type",
			triggeredEvent: webhook_module.HookEventPackage,
			payload:        &api.PackagePayload{Action: api.HookPackageCreated},
			yamlOn:         "on:\n  registry_package:\n    types: [updated]",
			expected:       false,
		},
		{
			desc:           "HookEventWiki(wiki) matches GithubEventGollum(gollum)",
			triggeredEvent: webhook_module.HookEventWiki,
			payload:        nil,
			yamlOn:         "on: gollum",
			expected:       true,
		},
		{
			desc:           "HookEventSchedule(schedule) matches GithubEventSchedule(schedule)",
			triggeredEvent: webhook_module.HookEventSchedule,
			payload:        nil,
			yamlOn:         "on: schedule",
			expected:       true,
		},
		{
			desc:           "HookEventWorkflowDispatch(workflow_dispatch) matches GithubEventWorkflowDispatch(workflow_dispatch)",
			triggeredEvent: webhook_module.HookEventWorkflowDispatch,
			payload:        nil,
			yamlOn:         "on: workflow_dispatch",
			expected:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			evts, err := GetEventsFromContent([]byte(tc.yamlOn))
			require.NoError(t, err)
			assert.Len(t, evts, 1)
			assert.Equal(t, tc.expected, detectMatched(nil, tc.commit, tc.triggeredEvent, tc.payload, evts[0]))
		})
	}
}
