// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forms

import (
	"net/http"

	"forgejo.org/modules/web/middleware"
	"forgejo.org/services/context"

	"code.forgejo.org/go-chi/binding"
)

// NewBranchForm form for creating a new branch
type NewBranchForm struct {
	NewBranchName string `binding:"Required;MaxSize(100);GitRefName"`
	CurrentPath   string
	CreateTag     bool
}

// Validate validates the fields
func (f *NewBranchForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// RenameBranchForm form for rename a branch
type RenameBranchForm struct {
	From string `binding:"Required;MaxSize(100);GitRefName"`
	To   string `binding:"Required;MaxSize(100);GitRefName"`
}

// Validate validates the fields
func (f *RenameBranchForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
