// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func testRepoMigrate(t testing.TB, session *TestSession, cloneAddr, repoName string, service structs.GitServiceType) *httptest.ResponseRecorder {
	req := NewRequest(t, "GET", fmt.Sprintf("/repo/migrate?service_type=%d", service)) // render plain git migration page
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	link, exists := htmlDoc.doc.Find("form.ui.form").Attr("action")
	assert.True(t, exists, "The template has changed")

	uid, exists := htmlDoc.doc.Find("#uid").Attr("value")
	assert.True(t, exists, "The template has changed")

	req = NewRequestWithValues(t, "POST", link, map[string]string{
		"_csrf":      htmlDoc.GetCSRF(),
		"clone_addr": cloneAddr,
		"uid":        uid,
		"repo_name":  repoName,
		"service":    fmt.Sprintf("%d", service),
	})
	resp = session.MakeRequest(t, req, http.StatusSeeOther)

	return resp
}

func TestRepoMigrate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")
	for _, s := range []struct {
		testName  string
		cloneAddr string
		repoName  string
		service   structs.GitServiceType
	}{
		{"TestMigrateGithub", "https://github.com/go-gitea/test_repo.git", "git", structs.PlainGitService},
		{"TestMigrateGithub", "https://github.com/go-gitea/test_repo.git", "github", structs.GithubService},
	} {
		t.Run(s.testName, func(t *testing.T) {
			testRepoMigrate(t, session, s.cloneAddr, s.repoName, s.service)
		})
	}
}
