// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package settings

import (
	"net/http"

	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/services/context"
)

// GetGeneralUISettings returns instance's global settings for ui
func GetGeneralUISettings(ctx *context.APIContext) {
	// swagger:operation GET /settings/ui settings getGeneralUISettings
	// ---
	// summary: Get instance's global settings for ui
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/GeneralUISettings"
	ctx.JSON(http.StatusOK, api.GeneralUISettings{
		DefaultTheme:     setting.UI.DefaultTheme,
		AllowedReactions: setting.UI.Reactions,
		CustomEmojis:     setting.UI.CustomEmojis,
	})
}

// GetGeneralAPISettings returns instance's global settings for api
func GetGeneralAPISettings(ctx *context.APIContext) {
	// swagger:operation GET /settings/api settings getGeneralAPISettings
	// ---
	// summary: Get instance's global settings for api
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/GeneralAPISettings"
	ctx.JSON(http.StatusOK, api.GeneralAPISettings{
		MaxResponseItems:       setting.API.MaxResponseItems,
		DefaultPagingNum:       setting.API.DefaultPagingNum,
		DefaultGitTreesPerPage: setting.API.DefaultGitTreesPerPage,
		DefaultMaxBlobSize:     setting.API.DefaultMaxBlobSize,
	})
}

// GetGeneralRepoSettings returns instance's global settings for repositories
func GetGeneralRepoSettings(ctx *context.APIContext) {
	// swagger:operation GET /settings/repository settings getGeneralRepositorySettings
	// ---
	// summary: Get instance's global settings for repositories
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/GeneralRepoSettings"
	ctx.JSON(http.StatusOK, api.GeneralRepoSettings{
		MirrorsDisabled:      !setting.Mirror.Enabled,
		HTTPGitDisabled:      setting.Repository.DisableHTTPGit,
		MigrationsDisabled:   setting.Repository.DisableMigrations,
		StarsDisabled:        setting.Repository.DisableStars,
		ForksDisabled:        setting.Repository.DisableForks,
		TimeTrackingDisabled: !setting.Service.EnableTimetracking,
		LFSDisabled:          !setting.LFS.StartServer,
	})
}

// GetGeneralAttachmentSettings returns instance's global settings for Attachment
func GetGeneralAttachmentSettings(ctx *context.APIContext) {
	// swagger:operation GET /settings/attachment settings getGeneralAttachmentSettings
	// ---
	// summary: Get instance's global settings for Attachment
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/GeneralAttachmentSettings"
	ctx.JSON(http.StatusOK, api.GeneralAttachmentSettings{
		Enabled:      setting.Attachment.Enabled,
		AllowedTypes: setting.Attachment.AllowedTypes,
		MaxFiles:     setting.Attachment.MaxFiles,
		MaxSize:      setting.Attachment.MaxSize,
	})
}
