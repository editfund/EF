// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func syncForkTest(t *testing.T, forkName, urlPart string, webSync bool) {
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 20})

	baseRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	baseUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: baseRepo.OwnerID})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	// Create a new fork
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/forks", baseUser.Name, baseRepo.LowerName), &api.CreateForkOption{Name: &forkName}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusAccepted)

	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/%s", user.Name, forkName, urlPart).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var syncForkInfo *api.SyncForkInfo
	DecodeJSON(t, resp, &syncForkInfo)

	// This is a new fork, so the commits in both branches should be the same
	assert.False(t, syncForkInfo.Allowed)
	assert.Equal(t, syncForkInfo.BaseCommit, syncForkInfo.ForkCommit)

	// Make a commit on the base branch
	err := createOrReplaceFileInBranch(baseUser, baseRepo, "sync_fork.txt", "master", "Hello")
	require.NoError(t, err)

	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/%s", user.Name, forkName, urlPart).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &syncForkInfo)

	// The commits should no longer be the same and we can sync
	assert.True(t, syncForkInfo.Allowed)
	assert.NotEqual(t, syncForkInfo.BaseCommit, syncForkInfo.ForkCommit)

	// Sync the fork
	if webSync {
		session.MakeRequest(t, NewRequestf(t, "GET", "/%s/%s/sync_fork/master", user.Name, forkName), http.StatusSeeOther)
	} else {
		req = NewRequestf(t, "POST", "/api/v1/repos/%s/%s/%s", user.Name, forkName, urlPart).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)
	}

	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/%s", user.Name, forkName, urlPart).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &syncForkInfo)

	// After the sync both commits should be the same again
	assert.False(t, syncForkInfo.Allowed)
	assert.Equal(t, syncForkInfo.BaseCommit, syncForkInfo.ForkCommit)
}

func TestAPIRepoSyncForkDefault(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		syncForkTest(t, "SyncForkDefault", "sync_fork", false)
	})
}

func TestAPIRepoSyncForkBranch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		syncForkTest(t, "SyncForkBranch", "sync_fork/master", false)
	})
}

func TestWebRepoSyncForkBranch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		syncForkTest(t, "SyncForkBranch", "sync_fork/master", true)
	})
}

func TestWebRepoSyncForkHomepage(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		baseRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		baseOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: baseRepo.OwnerID})
		baseOwnerSession := loginUser(t, baseOwner.Name)

		forkOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 20})
		forkOwnersession := loginUser(t, forkOwner.Name)
		token := getTokenForLoggedInUser(t, forkOwnersession, auth_model.AccessTokenScopeWriteRepository)

		// Rename branch "master" to "&amp;"
		baseOwnerSession.MakeRequest(t, NewRequestWithValues(t, "POST",
			"/user2/repo1/settings/rename_branch", map[string]string{
				"_csrf": GetCSRF(t, baseOwnerSession, "/user2/repo1/settings/branches"),
				"from":  "master",
				"to":    "&amp;",
			}), http.StatusSeeOther)

		forkName := "SyncForkHomepage"
		forkFullName := fmt.Sprintf("/%s/%s", forkOwner.Name, forkName)

		// Create a new fork
		req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/forks", baseOwner.Name, baseRepo.LowerName), &api.CreateForkOption{Name: &forkName}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusAccepted)

		// Make a commit on the base branch
		err := createOrReplaceFileInBranch(baseOwner, baseRepo, "sync_fork.txt", "&amp;", "Hello")
		require.NoError(t, err)

		resp := forkOwnersession.MakeRequest(t, NewRequest(t, "GET", forkFullName), http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		message := doc.Find("*")
		raw, _ := message.Html()
		//raw := resp.Body.String()
		assert.Contains(t, raw, fmt.Sprintf("This branch is 1 commit behind <a href='http://localhost:%s/user2/repo1/src/branch/&amp;'>user2/repo1:master</a>", u.Port()))
	})
}
