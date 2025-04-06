// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"context"
	"fmt"

	"forgejo.org/models"
	"forgejo.org/models/db"
	org_model "forgejo.org/models/organization"
	packages_model "forgejo.org/models/packages"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/storage"
	"forgejo.org/modules/util"
	repo_service "forgejo.org/services/repository"
)

// DeleteOrganization completely and permanently deletes everything of organization.
func DeleteOrganization(ctx context.Context, org *org_model.Organization, purge bool) error {
	ctx, commiter, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer commiter.Close()

	if purge {
		err := repo_service.DeleteOwnerRepositoriesDirectly(ctx, org.AsUser())
		if err != nil {
			return err
		}
	}

	// Check ownership of repository.
	count, err := repo_model.CountRepositories(ctx, repo_model.CountRepositoryOptions{OwnerID: org.ID})
	if err != nil {
		return fmt.Errorf("GetRepositoryCount: %w", err)
	} else if count > 0 {
		return models.ErrUserOwnRepos{UID: org.ID}
	}

	// Check ownership of packages.
	if ownsPackages, err := packages_model.HasOwnerPackages(ctx, org.ID); err != nil {
		return fmt.Errorf("HasOwnerPackages: %w", err)
	} else if ownsPackages {
		return models.ErrUserOwnPackages{UID: org.ID}
	}

	if err := org_model.DeleteOrganization(ctx, org); err != nil {
		return fmt.Errorf("DeleteOrganization: %w", err)
	}

	if err := commiter.Commit(); err != nil {
		return err
	}

	// FIXME: system notice
	// Note: There are something just cannot be roll back,
	//	so just keep error logs of those operations.
	path := user_model.UserPath(org.Name)

	if err := util.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to RemoveAll %s: %w", path, err)
	}

	if len(org.Avatar) > 0 {
		avatarPath := org.CustomAvatarRelativePath()
		if err := storage.Avatars.Delete(avatarPath); err != nil {
			return fmt.Errorf("failed to remove %s: %w", avatarPath, err)
		}
	}

	return nil
}
