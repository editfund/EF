// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package install

import (
	"fmt"
	"html"
	"net/http"

	"forgejo.org/modules/public"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/web"
	"forgejo.org/routers/common"
	"forgejo.org/routers/web/healthcheck"
	"forgejo.org/services/forms"
)

// Routes registers the installation routes
func Routes() *web.Route {
	base := web.NewRoute()
	base.Use(common.ProtocolMiddlewares()...)
	base.Methods("GET, HEAD", "/assets/*", public.FileHandlerFunc())

	r := web.NewRoute()
	r.Use(common.Sessioner(), Contexter())
	r.Get("/", Install) // it must be on the root, because the "install.js" use the window.location to replace the "localhost" AppURL
	r.Post("/", web.Bind(forms.InstallForm{}), SubmitInstall)
	r.Get("/post-install", InstallDone)
	r.Get("/api/healthz", healthcheck.Check)
	r.NotFound(installNotFound)

	base.Mount("", r)
	return base
}

func installNotFound(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Refresh", fmt.Sprintf("1; url=%s", setting.AppSubURL+"/"))
	// do not use 30x status, because the "post-install" page needs to use 404/200 to detect if Gitea has been installed.
	// the fetch API could follow 30x requests to the page with 200 status.
	w.WriteHeader(http.StatusNotFound)
	_, _ = fmt.Fprintf(w, `Not Found. <a href="%s">Go to default page</a>.`, html.EscapeString(setting.AppSubURL+"/"))
}
