// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"net/http"

	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
	webhook_service "forgejo.org/services/webhook"
)

// ListHooks list the authenticated user's webhooks
func ListHooks(ctx *context.APIContext) {
	// swagger:operation GET /user/hooks user userListHooks
	// ---
	// summary: List the authenticated user's webhooks
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
	//     "$ref": "#/responses/HookList"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	utils.ListOwnerHooks(
		ctx,
		ctx.Doer,
	)
}

// GetHook get the authenticated user's hook by id
func GetHook(ctx *context.APIContext) {
	// swagger:operation GET /user/hooks/{id} user userGetHook
	// ---
	// summary: Get a hook
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: id of the hook to get
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/Hook"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	hook, err := utils.GetOwnerHook(ctx, ctx.Doer.ID, ctx.ParamsInt64("id"))
	if err != nil {
		return
	}

	if !ctx.Doer.IsAdmin && hook.OwnerID != ctx.Doer.ID {
		ctx.NotFound()
		return
	}

	apiHook, err := webhook_service.ToHook(ctx.Doer.HomeLink(), hook)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, apiHook)
}

// CreateHook create a hook for the authenticated user
func CreateHook(ctx *context.APIContext) {
	// swagger:operation POST /user/hooks user userCreateHook
	// ---
	// summary: Create a hook
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/CreateHookOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/Hook"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	utils.AddOwnerHook(
		ctx,
		ctx.Doer,
		web.GetForm(ctx).(*api.CreateHookOption),
	)
}

// EditHook modify a hook of the authenticated user
func EditHook(ctx *context.APIContext) {
	// swagger:operation PATCH /user/hooks/{id} user userEditHook
	// ---
	// summary: Update a hook
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: id of the hook to update
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/EditHookOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/Hook"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	utils.EditOwnerHook(
		ctx,
		ctx.Doer,
		web.GetForm(ctx).(*api.EditHookOption),
		ctx.ParamsInt64("id"),
	)
}

// DeleteHook delete a hook of the authenticated user
func DeleteHook(ctx *context.APIContext) {
	// swagger:operation DELETE /user/hooks/{id} user userDeleteHook
	// ---
	// summary: Delete a hook
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: id of the hook to delete
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	utils.DeleteOwnerHook(
		ctx,
		ctx.Doer,
		ctx.ParamsInt64("id"),
	)
}
