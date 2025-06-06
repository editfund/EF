// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"testing"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
)

func TestRepoEdit(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx, _ := contexttest.MockAPIContext(t, "user2/repo1")
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadUser(t, ctx, 2)
	ctx.Repo.Owner = ctx.Doer
	description := "new description"
	website := "http://wwww.newwebsite.com"
	private := true
	hasIssues := false
	hasWiki := false
	defaultBranch := "master"
	hasPullRequests := true
	ignoreWhitespaceConflicts := true
	allowMerge := false
	allowRebase := false
	allowRebaseMerge := false
	allowSquashMerge := false
	allowFastForwardOnlyMerge := false
	archived := true
	opts := api.EditRepoOption{
		Name:                      &ctx.Repo.Repository.Name,
		Description:               &description,
		Website:                   &website,
		Private:                   &private,
		HasIssues:                 &hasIssues,
		HasWiki:                   &hasWiki,
		DefaultBranch:             &defaultBranch,
		HasPullRequests:           &hasPullRequests,
		IgnoreWhitespaceConflicts: &ignoreWhitespaceConflicts,
		AllowMerge:                &allowMerge,
		AllowRebase:               &allowRebase,
		AllowRebaseMerge:          &allowRebaseMerge,
		AllowSquash:               &allowSquashMerge,
		AllowFastForwardOnly:      &allowFastForwardOnlyMerge,
		Archived:                  &archived,
	}

	web.SetForm(ctx, &opts)
	Edit(ctx)

	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
		ID: 1,
	}, unittest.Cond("name = ? AND is_archived = 1", *opts.Name))
}

func TestRepoEditNameChange(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx, _ := contexttest.MockAPIContext(t, "user2/repo1")
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadUser(t, ctx, 2)
	ctx.Repo.Owner = ctx.Doer
	name := "newname"
	opts := api.EditRepoOption{
		Name: &name,
	}

	web.SetForm(ctx, &opts)
	Edit(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())

	unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
		ID: 1,
	}, unittest.Cond("name = ?", opts.Name))
}
