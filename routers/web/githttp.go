// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package web

import (
	"net/http"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/web"
	"forgejo.org/routers/web/repo"
	"forgejo.org/services/context"
)

func requireSignIn(ctx *context.Context) {
	if !setting.Service.RequireSignInView {
		return
	}

	// rely on the results of Contexter
	if !ctx.IsSigned {
		// TODO: support digit auth - which would be Authorization header with digit
		ctx.Resp.Header().Set("WWW-Authenticate", `Basic realm="Gitea"`)
		ctx.Error(http.StatusUnauthorized)
	}
}

func gitHTTPRouters(m *web.Route) {
	m.Group("", func() {
		m.Methods("POST,OPTIONS", "/git-upload-pack", repo.ServiceUploadPack)
		m.Methods("POST,OPTIONS", "/git-receive-pack", repo.ServiceReceivePack)
		m.Methods("GET,OPTIONS", "/info/refs", repo.GetInfoRefs)
		m.Methods("GET,OPTIONS", "/HEAD", repo.GetTextFile("HEAD"))
		m.Methods("GET,OPTIONS", "/objects/info/alternates", repo.GetTextFile("objects/info/alternates"))
		m.Methods("GET,OPTIONS", "/objects/info/http-alternates", repo.GetTextFile("objects/info/http-alternates"))
		m.Methods("GET,OPTIONS", "/objects/info/packs", repo.GetInfoPacks)
		m.Methods("GET,OPTIONS", "/objects/info/{file:[^/]*}", repo.GetTextFile(""))
		m.Methods("GET,OPTIONS", "/objects/{head:[0-9a-f]{2}}/{hash:[0-9a-f]{38,62}}", repo.GetLooseObject)
		m.Methods("GET,OPTIONS", "/objects/pack/pack-{file:[0-9a-f]{40,64}}.pack", repo.GetPackFile)
		m.Methods("GET,OPTIONS", "/objects/pack/pack-{file:[0-9a-f]{40,64}}.idx", repo.GetIdxFile)
	}, ignSignInAndCsrf, requireSignIn, repo.HTTPGitEnabledHandler, repo.CorsHandler(), context.UserAssignmentWeb())
}
