// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	"forgejo.org/modules/log"
	"forgejo.org/modules/web"
	actions_service "forgejo.org/services/actions"
	"forgejo.org/services/context"
	"forgejo.org/services/forms"
)

func SetVariablesContext(ctx *context.Context, ownerID, repoID int64) {
	variables, err := db.Find[actions_model.ActionVariable](ctx, actions_model.FindVariablesOpts{
		OwnerID: ownerID,
		RepoID:  repoID,
	})
	if err != nil {
		ctx.ServerError("FindVariables", err)
		return
	}
	ctx.Data["Variables"] = variables
}

func CreateVariable(ctx *context.Context, ownerID, repoID int64, redirectURL string) {
	form := web.GetForm(ctx).(*forms.EditVariableForm)

	v, err := actions_service.CreateVariable(ctx, ownerID, repoID, form.Name, form.Data)
	if err != nil {
		log.Error("CreateVariable: %v", err)
		ctx.JSONError(ctx.Tr("actions.variables.creation.failed"))
		return
	}

	ctx.Flash.Success(ctx.Tr("actions.variables.creation.success", v.Name))
	ctx.JSONRedirect(redirectURL)
}

func UpdateVariable(ctx *context.Context, ownerID, repoID int64, redirectURL string) {
	id := ctx.ParamsInt64(":variable_id")
	form := web.GetForm(ctx).(*forms.EditVariableForm)

	if ok, err := actions_service.UpdateVariable(ctx, id, ownerID, repoID, form.Name, form.Data); err != nil || !ok {
		if !ok {
			ctx.JSONError(ctx.Tr("actions.variables.not_found"))
		} else {
			log.Error("UpdateVariable: %v", err)
			ctx.JSONError(ctx.Tr("actions.variables.update.failed"))
		}
		return
	}
	ctx.Flash.Success(ctx.Tr("actions.variables.update.success"))
	ctx.JSONRedirect(redirectURL)
}

func DeleteVariable(ctx *context.Context, ownerID, repoID int64, redirectURL string) {
	id := ctx.ParamsInt64(":variable_id")

	if ok, err := actions_model.DeleteVariable(ctx, id, ownerID, repoID); err != nil || !ok {
		if !ok {
			ctx.JSONError(ctx.Tr("actions.variables.not_found"))
		} else {
			log.Error("Delete variable [%d] failed: %v", id, err)
			ctx.JSONError(ctx.Tr("actions.variables.deletion.failed"))
		}
		return
	}
	ctx.Flash.Success(ctx.Tr("actions.variables.deletion.success"))
	ctx.JSONRedirect(redirectURL)
}
