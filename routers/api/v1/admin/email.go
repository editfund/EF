// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"

	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
)

// GetAllEmails
func GetAllEmails(ctx *context.APIContext) {
	// swagger:operation GET /admin/emails admin adminGetAllEmails
	// ---
	// summary: List all emails
	// produces:
	// - application/json
	// parameters:
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/EmailList"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	listOptions := utils.GetListOptions(ctx)

	emails, maxResults, err := user_model.SearchEmails(ctx, &user_model.SearchEmailOptions{
		Keyword:     ctx.Params(":email"),
		ListOptions: listOptions,
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetAllEmails", err)
		return
	}

	results := make([]*api.Email, len(emails))
	for i := range emails {
		results[i] = convert.ToEmailSearch(emails[i])
	}

	ctx.SetLinkHeader(int(maxResults), listOptions.PageSize)
	ctx.SetTotalCountHeader(maxResults)
	ctx.JSON(http.StatusOK, &results)
}

// SearchEmail
func SearchEmail(ctx *context.APIContext) {
	// swagger:operation GET /admin/emails/search admin adminSearchEmails
	// ---
	// summary: Search all emails
	// produces:
	// - application/json
	// parameters:
	// - name: q
	//   in: query
	//   description: keyword
	//   type: string
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/EmailList"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	ctx.SetParams(":email", ctx.FormTrim("q"))
	GetAllEmails(ctx)
}
