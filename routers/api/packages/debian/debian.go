// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package debian

import (
	stdctx "context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"code.gitea.io/gitea/models/db"
	packages_model "code.gitea.io/gitea/models/packages"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/routers/api/packages/helper"
	"code.gitea.io/gitea/services/context"
	notify_service "code.gitea.io/gitea/services/notify"
	packages_service "code.gitea.io/gitea/services/packages"
	debian_packages_service "code.gitea.io/gitea/services/packages/debian"
)

func GetRepositoryKey(ctx *context.Context) {
	_, pub, err := debian_packages_service.GetOrCreateKeyPair(ctx, ctx.Package.Owner.ID)
	if err != nil {
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.ServeContent(strings.NewReader(pub), &context.ServeHeaderOptions{
		ContentType: "application/pgp-keys",
		Filename:    "repository.key",
	})
}

// https://wiki.debian.org/DebianRepository/Format#A.22Release.22_files
// https://wiki.debian.org/DebianRepository/Format#A.22Packages.22_Indices
func GetRepositoryFile(ctx *context.Context) {
	pv, err := debian_packages_service.GetOrCreateRepositoryVersion(ctx, ctx.Package.Owner.ID)
	if err != nil {
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}

	key := ctx.Params("distribution")

	component := ctx.Params("component")
	architecture := strings.TrimPrefix(ctx.Params("architecture"), "binary-")
	if component != "" && architecture != "" {
		key += "|" + component + "|" + architecture
	}

	s, u, pf, err := packages_service.GetFileStreamByPackageVersion(
		ctx,
		pv,
		&packages_service.PackageFileInfo{
			Filename:     ctx.Params("filename"),
			CompositeKey: key,
		},
	)
	if err != nil {
		if err == packages_model.ErrPackageNotExist || err == packages_model.ErrPackageFileNotExist {
			helper.APIError(ctx, http.StatusNotFound, err)
		} else {
			helper.APIError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	helper.ServePackageFile(ctx, s, u, pf)
}

// https://wiki.debian.org/DebianRepository/Format#indices_acquisition_via_hashsums_.28by-hash.29
func GetRepositoryFileByHash(ctx *context.Context) {
	pv, err := debian_packages_service.GetOrCreateRepositoryVersion(ctx, ctx.Package.Owner.ID)
	if err != nil {
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}

	algorithm := strings.ToLower(ctx.Params("algorithm"))
	if algorithm == "md5sum" {
		algorithm = "md5"
	}

	pfs, _, err := packages_model.SearchFiles(ctx, &packages_model.PackageFileSearchOptions{
		VersionID:     pv.ID,
		Hash:          strings.ToLower(ctx.Params("hash")),
		HashAlgorithm: algorithm,
	})
	if err != nil {
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}
	if len(pfs) != 1 {
		helper.APIError(ctx, http.StatusNotFound, nil)
		return
	}

	s, u, pf, err := packages_service.GetPackageFileStream(ctx, pfs[0])
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
	distribution := strings.TrimSpace(ctx.Params("distribution"))
	component := strings.TrimSpace(ctx.Params("component"))
	if distribution == "" || component == "" {
		helper.APIError(ctx, http.StatusBadRequest, "invalid distribution or component")
		return
	}

	upload, needToClose, err := ctx.UploadStream()
	if err != nil {
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}
	if needToClose {
		defer upload.Close()
	}

	_, err = debian_packages_service.UploadPackage(ctx, distribution, component, upload, ctx.Package.Owner, ctx.Doer)
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
			PackageType: packages_model.TypeDebian,
			Name:        name,
			Version:     version,
		},
		&packages_service.PackageFileInfo{
			Filename:     fmt.Sprintf("%s_%s_%s.deb", name, version, ctx.Params("architecture")),
			CompositeKey: fmt.Sprintf("%s|%s", ctx.Params("distribution"), ctx.Params("component")),
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

	helper.ServePackageFile(ctx, s, u, pf, &context.ServeHeaderOptions{
		ContentType:  "application/vnd.debian.binary-package",
		Filename:     pf.Name,
		LastModified: pf.CreatedUnix.AsLocalTime(),
	})
}

func DeletePackageFile(ctx *context.Context) {
	distribution := ctx.Params("distribution")
	component := ctx.Params("component")
	name := ctx.Params("name")
	version := ctx.Params("version")
	architecture := ctx.Params("architecture")

	owner := ctx.Package.Owner

	var pd *packages_model.PackageDescriptor

	err := db.WithTx(ctx, func(ctx stdctx.Context) error {
		pv, err := packages_model.GetVersionByNameAndVersion(ctx, owner.ID, packages_model.TypeDebian, name, version)
		if err != nil {
			return err
		}

		pf, err := packages_model.GetFileForVersionByName(
			ctx,
			pv.ID,
			fmt.Sprintf("%s_%s_%s.deb", name, version, architecture),
			fmt.Sprintf("%s|%s", distribution, component),
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
			helper.APIError(ctx, http.StatusNotFound, err)
		} else {
			helper.APIError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	if pd != nil {
		notify_service.PackageDelete(ctx, ctx.Doer, pd)
	}

	if err := debian_packages_service.BuildSpecificRepositoryFiles(ctx, ctx.Package.Owner.ID, distribution, component, architecture); err != nil {
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
