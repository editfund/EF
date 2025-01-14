// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package shared

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/container"
	"code.gitea.io/gitea/modules/structs"
	"errors"
	"log"
	"net/http"
	"strings"

	actions_model "code.gitea.io/gitea/models/actions"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/services/context"
)

// RegistrationToken is a string used to register a runner with a server
// swagger:response RegistrationToken
type RegistrationToken struct {
	Token string `json:"token"`
}

func GetRegistrationToken(ctx *context.APIContext, ownerID, repoID int64) {
	token, err := actions_model.GetLatestRunnerToken(ctx, ownerID, repoID)
	if errors.Is(err, util.ErrNotExist) || (token != nil && !token.IsActive) {
		token, err = actions_model.NewRunnerToken(ctx, ownerID, repoID)
	}
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	ctx.JSON(http.StatusOK, RegistrationToken{Token: token.Token})
}

// RunJobList is a list of action run jobs
// swagger:response RunJobList
type RunJobList struct {
	Jobs []*structs.ActionRunJob `json:"jobs,omitempty"`
}

func GetActionRunJobs(ctx *context.APIContext, ownerID int64, repoID int64) {
	labels := strings.Split(ctx.FormTrim("labels"), ",")
	log.Printf("labels: %v", labels)

	total, err := db.Find[actions_model.ActionRunJob](ctx, &actions_model.FindTaskOptions{
		Status:  []actions_model.Status{actions_model.StatusWaiting, actions_model.StatusRunning},
		OwnerID: ownerID,
		RepoID:  repoID,
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "CountWaitingActionRunJobs", err)
		return
	}

	res := new(RunJobList)
	res.Jobs = fromRunJobModelToResponse(total, labels)

	ctx.JSON(http.StatusOK, res)
}

func fromRunJobModelToResponse(job []*actions_model.ActionRunJob, labels []string) []*structs.ActionRunJob {
	var res []*structs.ActionRunJob
	for i := range job {
		if len(labels) > 0 && isSubset(labels, job[i].RunsOn) {
			res = append(res, &structs.ActionRunJob{
				ID:      job[i].ID,
				RepoID:  job[i].RepoID,
				OwnerID: job[i].OwnerID,
				Name:    job[i].Name,
				Needs:   job[i].Needs,
				RunsOn:  job[i].RunsOn,
				TaskID:  job[i].TaskID,
				Status:  job[i].Status.String(),
			})
		}
	}
	return res
}

func isSubset(set, subset []string) bool {
	m := make(container.Set[string], len(set))
	for _, v := range set {
		m.Add(v)
	}

	for _, v := range subset {
		if !m.Contains(v) {
			return false
		}
	}
	return true
}
