// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"net/http"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/modules/base"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/forms"
)

const (
	tplSettingsApplications base.TplName = "user/settings/applications"
)

// Applications render manage access token page
func Applications(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.applications")
	ctx.Data["PageIsSettingsApplications"] = true

	loadApplicationsData(ctx)

	ctx.HTML(http.StatusOK, tplSettingsApplications)
}

// ApplicationsPost response for add user's access token
func ApplicationsPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.NewAccessTokenForm)
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsApplications"] = true

	if ctx.HasError() {
		loadApplicationsData(ctx)

		ctx.HTML(http.StatusOK, tplSettingsApplications)
		return
	}

	scope, err := form.GetScope()
	if err != nil {
		ctx.ServerError("GetScope", err)
		return
	}
	if !scope.HasPermissionScope() {
		ctx.Flash.Error(ctx.Tr("settings.at_least_one_permission"), true)
	}
	t := &auth_model.AccessToken{
		UID:   ctx.Doer.ID,
		Name:  form.Name,
		Scope: scope,
	}

	exist, err := auth_model.AccessTokenByNameExists(ctx, t)
	if err != nil {
		ctx.ServerError("AccessTokenByNameExists", err)
		return
	}
	if exist {
		ctx.Flash.Error(ctx.Tr("settings.generate_token_name_duplicate", t.Name))
		ctx.Redirect(setting.AppSubURL + "/user/settings/applications")
		return
	}

	if err := auth_model.NewAccessToken(ctx, t); err != nil {
		ctx.ServerError("NewAccessToken", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.generate_token_success"))
	ctx.Flash.Info(t.Token)

	ctx.Redirect(setting.AppSubURL + "/user/settings/applications")
}

// DeleteApplication response for delete user access token
func DeleteApplication(ctx *context.Context) {
	if err := auth_model.DeleteAccessTokenByID(ctx, ctx.FormInt64("id"), ctx.Doer.ID); err != nil {
		ctx.Flash.Error("DeleteAccessTokenByID: " + err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("settings.delete_token_success"))
	}

	ctx.JSONRedirect(setting.AppSubURL + "/user/settings/applications")
}

// RegenerateApplication response for regenerating user access token
func RegenerateApplication(ctx *context.Context) {
	if t, err := auth_model.RegenerateAccessTokenByID(ctx, ctx.FormInt64("id"), ctx.Doer.ID); err != nil {
		if auth_model.IsErrAccessTokenNotExist(err) {
			ctx.Flash.Error(ctx.Tr("error.not_found"))
		} else {
			ctx.Flash.Error(ctx.Tr("error.server_internal"))
			log.Error("DeleteAccessTokenByID", err)
		}
	} else {
		ctx.Flash.Success(ctx.Tr("settings.regenerate_token_success"))
		ctx.Flash.Info(t.Token)
	}

	ctx.JSONRedirect(setting.AppSubURL + "/user/settings/applications")
}

func loadApplicationsData(ctx *context.Context) {
	ctx.Data["AccessTokenScopePublicOnly"] = auth_model.AccessTokenScopePublicOnly
	tokens, err := db.Find[auth_model.AccessToken](ctx, auth_model.ListAccessTokensOptions{UserID: ctx.Doer.ID})
	if err != nil {
		ctx.ServerError("ListAccessTokens", err)
		return
	}
	ctx.Data["Tokens"] = tokens
	ctx.Data["EnableOAuth2"] = setting.OAuth2.Enabled
	ctx.Data["IsAdmin"] = ctx.Doer.IsAdmin
	if setting.OAuth2.Enabled {
		ctx.Data["Applications"], err = db.Find[auth_model.OAuth2Application](ctx, auth_model.FindOAuth2ApplicationsOptions{
			OwnerID: ctx.Doer.ID,
		})
		if err != nil {
			ctx.ServerError("GetOAuth2ApplicationsByUserID", err)
			return
		}
		ctx.Data["Grants"], err = auth_model.GetOAuth2GrantsByUserID(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.ServerError("GetOAuth2GrantsByUserID", err)
			return
		}
		ctx.Data["EnableAdditionalGrantScopes"] = setting.OAuth2.EnableAdditionalGrantScopes
	}
}
