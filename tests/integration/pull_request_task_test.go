// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: MIT

package integration

import (
	"testing"
	"time"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	repo_module "forgejo.org/modules/repository"
	"forgejo.org/modules/test"
	"forgejo.org/modules/timeutil"
	pull_service "forgejo.org/services/pull"
	repo_service "forgejo.org/services/repository"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequestSynchronized(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// unmerged pull request of user2/repo1 from branch2 to master
	pull := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	// tip of tests/gitea-repositories-meta/user2/repo1 branch2
	pull.HeadCommitID = "985f0301dba5e7b34be866819cd15ad3d8f508ee"
	pull.LoadIssue(db.DefaultContext)
	pull.Issue.Created = timeutil.TimeStampNanoNow()
	issues_model.UpdateIssueCols(db.DefaultContext, pull.Issue, "created")

	require.Equal(t, pull.HeadRepoID, pull.BaseRepoID)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: pull.HeadRepoID})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	for _, testCase := range []struct {
		name     string
		timeNano int64
		expected bool
	}{
		{
			name:     "AddTestPullRequestTask process PR",
			timeNano: int64(pull.Issue.Created),
			expected: true,
		},
		{
			name:     "AddTestPullRequestTask skip PR",
			timeNano: 0,
			expected: false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			logChecker, cleanup := test.NewLogChecker(log.DEFAULT, log.TRACE)
			logChecker.Filter("Updating PR").StopMark("TestPullRequest ")
			defer cleanup()

			opt := &repo_module.PushUpdateOptions{
				PusherID:     owner.ID,
				PusherName:   owner.Name,
				RepoUserName: owner.Name,
				RepoName:     repo.Name,
				RefFullName:  git.RefName("refs/heads/branch2"),
				OldCommitID:  pull.HeadCommitID,
				NewCommitID:  pull.HeadCommitID,
				TimeNano:     testCase.timeNano,
			}
			require.NoError(t, repo_service.PushUpdate(opt))
			logFiltered, logStopped := logChecker.Check(5 * time.Second)
			assert.True(t, logStopped)
			assert.Equal(t, testCase.expected, logFiltered[0])
		})
	}

	for _, testCase := range []struct {
		name      string
		olderThan int64
		expected  bool
	}{
		{
			name:      "TestPullRequest process PR",
			olderThan: int64(pull.Issue.Created),
			expected:  true,
		},
		{
			name:      "TestPullRequest skip PR",
			olderThan: int64(pull.Issue.Created) - 1,
			expected:  false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			logChecker, cleanup := test.NewLogChecker(log.DEFAULT, log.TRACE)
			logChecker.Filter("Updating PR").StopMark("TestPullRequest ")
			defer cleanup()

			pull_service.TestPullRequest(t.Context(), owner, repo.ID, testCase.olderThan, "branch2", true, pull.HeadCommitID, pull.HeadCommitID)
			logFiltered, logStopped := logChecker.Check(5 * time.Second)
			assert.True(t, logStopped)
			assert.Equal(t, testCase.expected, logFiltered[0])
		})
	}
}
