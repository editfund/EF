// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package packages

import (
	"context"
	"fmt"

	org_model "forgejo.org/models/organization"
	packages_model "forgejo.org/models/packages"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/util"
)

func LinkToRepository(ctx context.Context, pkg *packages_model.Package, repo *repo_model.Repository, doer *user_model.User) error {
	if pkg.OwnerID != repo.OwnerID {
		return util.ErrPermissionDenied
	}
	if pkg.RepoID > 0 {
		return util.ErrInvalidArgument
	}

	perms, err := access_model.GetUserRepoPermission(ctx, repo, doer)
	if err != nil {
		return fmt.Errorf("error getting permissions for user %d on repository %d: %w", doer.ID, repo.ID, err)
	}
	if !perms.CanWrite(unit.TypePackages) {
		return util.ErrPermissionDenied
	}

	if err := packages_model.SetRepositoryLink(ctx, pkg.ID, repo.ID); err != nil {
		return fmt.Errorf("error while linking package '%v' to repo '%v' : %w", pkg.Name, repo.FullName(), err)
	}
	return nil
}

func UnlinkFromRepository(ctx context.Context, pkg *packages_model.Package, doer *user_model.User) error {
	if pkg.RepoID == 0 {
		return util.ErrInvalidArgument
	}

	repo, err := repo_model.GetRepositoryByID(ctx, pkg.RepoID)
	if err != nil && !repo_model.IsErrRepoNotExist(err) {
		return fmt.Errorf("error getting repository %d: %w", pkg.RepoID, err)
	}
	if err == nil {
		perms, err := access_model.GetUserRepoPermission(ctx, repo, doer)
		if err != nil {
			return fmt.Errorf("error getting permissions for user %d on repository %d: %w", doer.ID, repo.ID, err)
		}
		if !perms.CanWrite(unit.TypePackages) {
			return util.ErrPermissionDenied
		}
	}

	user, err := user_model.GetUserByID(ctx, pkg.OwnerID)
	if err != nil {
		return err
	}
	if !doer.IsAdmin {
		if !user.IsOrganization() {
			if doer.ID != pkg.OwnerID {
				return fmt.Errorf("no permission to unlink package '%v' from its repository, or packages are disabled", pkg.Name)
			}
		} else {
			isOrgAdmin, err := org_model.OrgFromUser(user).IsOrgAdmin(ctx, doer.ID)
			if err != nil {
				return err
			} else if !isOrgAdmin {
				return fmt.Errorf("no permission to unlink package '%v' from its repository, or packages are disabled", pkg.Name)
			}
		}
	}
	return packages_model.UnlinkRepository(ctx, pkg.ID)
}
