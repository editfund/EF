// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	"forgejo.org/models/db"
	"forgejo.org/models/packages"
	"forgejo.org/models/unit"
	"forgejo.org/modules/base"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

const (
	tplPackagesList base.TplName = "repo/packages"
)

// Packages displays a list of all packages in the repository
func Packages(ctx *context.Context) {
	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}
	query := ctx.FormTrim("q")
	packageType := ctx.FormTrim("type")

	pvs, total, err := packages.SearchLatestVersions(ctx, &packages.PackageSearchOptions{
		Paginator: &db.ListOptions{
			PageSize: setting.UI.PackagesPagingNum,
			Page:     page,
		},
		OwnerID:    ctx.ContextUser.ID,
		RepoID:     ctx.Repo.Repository.ID,
		Type:       packages.Type(packageType),
		Name:       packages.SearchValue{Value: query},
		IsInternal: optional.Some(false),
	})
	if err != nil {
		ctx.ServerError("SearchLatestVersions", err)
		return
	}

	pds, err := packages.GetPackageDescriptors(ctx, pvs)
	if err != nil {
		ctx.ServerError("GetPackageDescriptors", err)
		return
	}

	hasPackages, err := packages.HasRepositoryPackages(ctx, ctx.Repo.Repository.ID)
	if err != nil {
		ctx.ServerError("HasRepositoryPackages", err)
		return
	}

	ctx.Data["Title"] = ctx.Tr("packages.title")
	ctx.Data["IsPackagesPage"] = true
	ctx.Data["Query"] = query
	ctx.Data["PackageType"] = packageType
	ctx.Data["AvailableTypes"] = packages.TypeList
	ctx.Data["HasPackages"] = hasPackages
	if ctx.Repo != nil {
		ctx.Data["CanWritePackages"] = ctx.IsUserRepoWriter([]unit.Type{unit.TypePackages}) || ctx.IsUserSiteAdmin()
	}
	ctx.Data["PackageDescriptors"] = pds
	ctx.Data["Total"] = total
	ctx.Data["RepositoryAccessMap"] = map[int64]bool{ctx.Repo.Repository.ID: true} // There is only the current repository

	pager := context.NewPagination(int(total), setting.UI.PackagesPagingNum, page, 5)
	pager.AddParam(ctx, "q", "Query")
	pager.AddParam(ctx, "type", "PackageType")
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplPackagesList)
}
