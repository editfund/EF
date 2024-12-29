// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package rpm

import (
	stdctx "context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"code.gitea.io/gitea/models/db"
	packages_model "code.gitea.io/gitea/models/packages"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/routers/api/packages/helper"
	"code.gitea.io/gitea/services/context"
	notify_service "code.gitea.io/gitea/services/notify"
	packages_service "code.gitea.io/gitea/services/packages"
	rpm_packages_service "code.gitea.io/gitea/services/packages/rpm"
)

// https://dnf.readthedocs.io/en/latest/conf_ref.html
func GetRepositoryConfig(ctx *context.Context) {
	group := ctx.Params("group")

	var groupParts []string
	if group != "" {
		groupParts = strings.Split(group, "/")
	}

	url := fmt.Sprintf("%sapi/packages/%s/rpm", setting.AppURL, ctx.Package.Owner.Name)

	ctx.PlainText(http.StatusOK, `[gitea-`+strings.Join(append([]string{ctx.Package.Owner.LowerName}, groupParts...), "-")+`]
name=`+strings.Join(append([]string{ctx.Package.Owner.Name, setting.AppName}, groupParts...), " - ")+`
baseurl=`+strings.Join(append([]string{url}, groupParts...), "/")+`
enabled=1
gpgcheck=1
gpgkey=`+url+`/repository.key`)
}

// Gets or creates the PGP public key used to sign repository metadata files
func GetRepositoryKey(ctx *context.Context) {
	_, pub, err := rpm_packages_service.GetOrCreateKeyPair(ctx, ctx.Package.Owner.ID)
	if err != nil {
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.ServeContent(strings.NewReader(pub), &context.ServeHeaderOptions{
		ContentType: "application/pgp-keys",
		Filename:    "repository.key",
	})
}

func CheckRepositoryFileExistence(ctx *context.Context) {
	pv, err := rpm_packages_service.GetOrCreateRepositoryVersion(ctx, ctx.Package.Owner.ID)
	if err != nil {
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}

	pf, err := packages_model.GetFileForVersionByName(ctx, pv.ID, ctx.Params("filename"), ctx.Params("group"))
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			ctx.Status(http.StatusNotFound)
		} else {
			helper.APIError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.SetServeHeaders(&context.ServeHeaderOptions{
		Filename:     pf.Name,
		LastModified: pf.CreatedUnix.AsLocalTime(),
	})
	ctx.Status(http.StatusOK)
}

// Gets a pre-generated repository metadata file
func GetRepositoryFile(ctx *context.Context) {
	pv, err := rpm_packages_service.GetOrCreateRepositoryVersion(ctx, ctx.Package.Owner.ID)
	if err != nil {
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}

	s, u, pf, err := packages_service.GetFileStreamByPackageVersion(
		ctx,
		pv,
		&packages_service.PackageFileInfo{
			Filename:     ctx.Params("filename"),
			CompositeKey: ctx.Params("group"),
		},
	)
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			helper.APIError(ctx, http.StatusNotFound, err)
		} else {
			helper.APIError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	helper.ServePackageFile(ctx, s, u, pf)
}

func UploadPackageFile(ctx *context.Context) {
	upload, needToClose, err := ctx.UploadStream()
	if err != nil {
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}
	if needToClose {
		defer upload.Close()
	}

	sign := ctx.FormBool("sign")
	group := ctx.Params("group")

	_, err = rpm_packages_service.UploadPackage(ctx, sign, group, upload, ctx.Package.Owner, ctx.Doer)
	if err != nil {
		helper.PackageUploadError(ctx, err)
		return
	}

	ctx.Status(http.StatusCreated)
}

func DownloadPackageFile(ctx *context.Context) {
	name := ctx.Params("name")
	version := ctx.Params("version")

	s, u, pf, err := packages_service.GetFileStreamByPackageNameAndVersion(
		ctx,
		&packages_service.PackageInfo{
			Owner:       ctx.Package.Owner,
			PackageType: packages_model.TypeRpm,
			Name:        name,
			Version:     version,
		},
		&packages_service.PackageFileInfo{
			Filename:     fmt.Sprintf("%s-%s.%s.rpm", name, version, ctx.Params("architecture")),
			CompositeKey: ctx.Params("group"),
		},
	)
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			helper.APIError(ctx, http.StatusNotFound, err)
		} else {
			helper.APIError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	helper.ServePackageFile(ctx, s, u, pf)
}

func DeletePackageFile(webctx *context.Context) {
	group := webctx.Params("group")
	name := webctx.Params("name")
	version := webctx.Params("version")
	architecture := webctx.Params("architecture")

	var pd *packages_model.PackageDescriptor

	err := db.WithTx(webctx, func(ctx stdctx.Context) error {
		pv, err := packages_model.GetVersionByNameAndVersion(ctx,
			webctx.Package.Owner.ID,
			packages_model.TypeRpm,
			name,
			version,
		)
		if err != nil {
			return err
		}

		pf, err := packages_model.GetFileForVersionByName(
			ctx,
			pv.ID,
			fmt.Sprintf("%s-%s.%s.rpm", name, version, architecture),
			group,
		)
		if err != nil {
			return err
		}

		if err := packages_service.DeletePackageFile(ctx, pf); err != nil {
			return err
		}

		has, err := packages_model.HasVersionFileReferences(ctx, pv.ID)
		if err != nil {
			return err
		}
		if !has {
			pd, err = packages_model.GetPackageDescriptor(ctx, pv)
			if err != nil {
				return err
			}

			if err := packages_service.DeletePackageVersionAndReferences(ctx, pv); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			helper.APIError(webctx, http.StatusNotFound, err)
		} else {
			helper.APIError(webctx, http.StatusInternalServerError, err)
		}
		return
	}

	if pd != nil {
		notify_service.PackageDelete(webctx, webctx.Doer, pd)
	}

	if err := rpm_packages_service.BuildSpecificRepositoryFiles(webctx, webctx.Package.Owner.ID, group); err != nil {
		helper.APIError(webctx, http.StatusInternalServerError, err)
		return
	}

	webctx.Status(http.StatusNoContent)
}
