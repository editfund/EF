// Copyright 2016 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"net/http"

	"forgejo.org/models/db"
	"forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	repo_module "forgejo.org/modules/repository"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
	repo_service "forgejo.org/services/repository"
)

// ListCollaborators list a repository's collaborators
func ListCollaborators(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/collaborators repository repoListCollaborators
	// ---
	// summary: List a repository's collaborators
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/UserList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	count, err := db.Count[repo_model.Collaboration](ctx, repo_model.FindCollaborationOptions{
		RepoID: ctx.Repo.Repository.ID,
	})
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	collaborators, err := repo_model.GetCollaborators(ctx, ctx.Repo.Repository.ID, utils.GetListOptions(ctx))
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListCollaborators", err)
		return
	}

	users := make([]*api.User, len(collaborators))
	for i, collaborator := range collaborators {
		users[i] = convert.ToUser(ctx, collaborator.User, ctx.Doer)
	}

	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, users)
}

// IsCollaborator check if a user is a collaborator of a repository
func IsCollaborator(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/collaborators/{collaborator} repository repoCheckCollaborator
	// ---
	// summary: Check if a user is a collaborator of a repository
	// description: If the user is a collaborator, return 204. If the user is not a collaborator, return 404.
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: collaborator
	//   in: path
	//   description: username of the collaborator
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	user, err := user_model.GetUserByName(ctx, ctx.Params(":collaborator"))
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			ctx.Error(http.StatusUnprocessableEntity, "", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetUserByName", err)
		}
		return
	}
	isColab, err := repo_model.IsCollaborator(ctx, ctx.Repo.Repository.ID, user.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "IsCollaborator", err)
		return
	}
	if isColab {
		ctx.Status(http.StatusNoContent)
	} else {
		ctx.NotFound()
	}
}

// AddCollaborator add a collaborator to a repository
func AddCollaborator(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/collaborators/{collaborator} repository repoAddCollaborator
	// ---
	// summary: Add a collaborator to a repository
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: collaborator
	//   in: path
	//   description: username of the collaborator to add
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/AddCollaboratorOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	form := web.GetForm(ctx).(*api.AddCollaboratorOption)

	collaborator, err := user_model.GetUserByName(ctx, ctx.Params(":collaborator"))
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			ctx.Error(http.StatusUnprocessableEntity, "", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetUserByName", err)
		}
		return
	}

	if !collaborator.IsActive {
		ctx.Error(http.StatusInternalServerError, "InactiveCollaborator", errors.New("collaborator's account is inactive"))
		return
	}

	if err := repo_module.AddCollaborator(ctx, ctx.Repo.Repository, collaborator); err != nil {
		if errors.Is(err, user_model.ErrBlockedByUser) {
			ctx.Error(http.StatusForbidden, "AddCollaborator", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "AddCollaborator", err)
		}
		return
	}

	if form.Permission != nil {
		if err := repo_model.ChangeCollaborationAccessMode(ctx, ctx.Repo.Repository, collaborator.ID, perm.ParseAccessMode(*form.Permission)); err != nil {
			ctx.Error(http.StatusInternalServerError, "ChangeCollaborationAccessMode", err)
			return
		}
	}

	ctx.Status(http.StatusNoContent)
}

// DeleteCollaborator delete a collaborator from a repository
func DeleteCollaborator(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/collaborators/{collaborator} repository repoDeleteCollaborator
	// ---
	// summary: Delete a collaborator from a repository
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: collaborator
	//   in: path
	//   description: username of the collaborator to delete
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	collaborator, err := user_model.GetUserByName(ctx, ctx.Params(":collaborator"))
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			ctx.Error(http.StatusUnprocessableEntity, "", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetUserByName", err)
		}
		return
	}

	if err := repo_service.DeleteCollaboration(ctx, ctx.Repo.Repository, collaborator.ID); err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteCollaboration", err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// GetRepoPermissions gets repository permissions for a user
func GetRepoPermissions(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/collaborators/{collaborator}/permission repository repoGetRepoPermissions
	// ---
	// summary: Get repository permissions for a user
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: collaborator
	//   in: path
	//   description: username of the collaborator
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/RepoCollaboratorPermission"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	collaborator, err := user_model.GetUserByName(ctx, ctx.Params(":collaborator"))
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			ctx.Error(http.StatusNotFound, "GetUserByName", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetUserByName", err)
		}
		return
	}

	// Only allow the request in any of the following situations:
	// - The user is the instance admin.
	// - The user is the repository admin.
	// - The user is querying the permissions of themselves.
	if !ctx.IsUserSiteAdmin() && ctx.Doer.ID != collaborator.ID && !ctx.IsUserRepoAdmin() {
		ctx.Error(http.StatusForbidden, "User", "Only admins can query all permissions, repo admins can query all repo permissions, collaborators can query only their own")
		return
	}

	permission, err := access_model.GetUserRepoPermission(ctx, ctx.Repo.Repository, collaborator)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetUserRepoPermission", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToUserAndPermission(ctx, collaborator, ctx.ContextUser, permission.AccessMode))
}

// GetReviewers return all users that can be requested to review in this repo
func GetReviewers(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/reviewers repository repoGetReviewers
	// ---
	// summary: Return all users that can be requested to review in this repo
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/UserList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	reviewers, err := repo_model.GetReviewers(ctx, ctx.Repo.Repository, ctx.Doer.ID, 0)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListCollaborators", err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToUsers(ctx, ctx.Doer, reviewers))
}

// GetAssignees return all users that have write access and can be assigned to issues
func GetAssignees(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/assignees repository repoGetAssignees
	// ---
	// summary: Return all users that have write access and can be assigned to issues
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/UserList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	assignees, err := repo_model.GetRepoAssignees(ctx, ctx.Repo.Repository)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListCollaborators", err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToUsers(ctx, ctx.Doer, assignees))
}
