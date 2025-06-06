// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package runner

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	actions_model "forgejo.org/models/actions"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/actions"
	"forgejo.org/modules/log"
	"forgejo.org/modules/util"
	actions_service "forgejo.org/services/actions"

	runnerv1 "code.gitea.io/actions-proto-go/runner/v1"
	"code.gitea.io/actions-proto-go/runner/v1/runnerv1connect"
	"connectrpc.com/connect"
	gouuid "github.com/google/uuid"
)

func NewRunnerServiceHandler() (string, http.Handler) {
	return runnerv1connect.NewRunnerServiceHandler(
		&Service{},
		connect.WithCompressMinBytes(1024),
		withRunner,
	)
}

var _ runnerv1connect.RunnerServiceClient = (*Service)(nil)

type Service struct {
	runnerv1connect.UnimplementedRunnerServiceHandler
}

// Register for new runner.
func (s *Service) Register(
	ctx context.Context,
	req *connect.Request[runnerv1.RegisterRequest],
) (*connect.Response[runnerv1.RegisterResponse], error) {
	if req.Msg.Token == "" || req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing runner token, name"))
	}

	runnerToken, err := actions_model.GetRunnerToken(ctx, req.Msg.Token)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("runner registration token not found"))
	}

	if !runnerToken.IsActive {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("runner registration token has been invalidated, please use the latest one"))
	}

	if runnerToken.OwnerID > 0 {
		if _, err := user_model.GetUserByID(ctx, runnerToken.OwnerID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("owner of the token not found"))
		}
	}

	if runnerToken.RepoID > 0 {
		if _, err := repo_model.GetRepositoryByID(ctx, runnerToken.RepoID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("repository of the token not found"))
		}
	}

	labels := req.Msg.Labels

	// create new runner
	name, _ := util.SplitStringAtByteN(req.Msg.Name, 255)
	runner := &actions_model.ActionRunner{
		UUID:        gouuid.New().String(),
		Name:        name,
		OwnerID:     runnerToken.OwnerID,
		RepoID:      runnerToken.RepoID,
		Version:     req.Msg.Version,
		AgentLabels: labels,
	}
	if err := runner.GenerateToken(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("can't generate token"))
	}

	// create new runner
	if err := actions_model.CreateRunner(ctx, runner); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("can't create new runner"))
	}

	// update token status
	runnerToken.IsActive = true
	if err := actions_model.UpdateRunnerToken(ctx, runnerToken, "is_active"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("can't update runner token status"))
	}

	res := connect.NewResponse(&runnerv1.RegisterResponse{
		Runner: &runnerv1.Runner{
			Id:      runner.ID,
			Uuid:    runner.UUID,
			Token:   runner.Token,
			Name:    runner.Name,
			Version: runner.Version,
			Labels:  runner.AgentLabels,
		},
	})

	return res, nil
}

func (s *Service) Declare(
	ctx context.Context,
	req *connect.Request[runnerv1.DeclareRequest],
) (*connect.Response[runnerv1.DeclareResponse], error) {
	runner := GetRunner(ctx)
	runner.AgentLabels = req.Msg.Labels
	runner.Version = req.Msg.Version
	if err := actions_model.UpdateRunner(ctx, runner, "agent_labels", "version"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update runner: %w", err))
	}

	return connect.NewResponse(&runnerv1.DeclareResponse{
		Runner: &runnerv1.Runner{
			Id:      runner.ID,
			Uuid:    runner.UUID,
			Token:   runner.Token,
			Name:    runner.Name,
			Version: runner.Version,
			Labels:  runner.AgentLabels,
		},
	}), nil
}

// FetchTask assigns a task to the runner
func (s *Service) FetchTask(
	ctx context.Context,
	req *connect.Request[runnerv1.FetchTaskRequest],
) (*connect.Response[runnerv1.FetchTaskResponse], error) {
	runner := GetRunner(ctx)

	var task *runnerv1.Task
	tasksVersion := req.Msg.TasksVersion // task version from runner
	latestVersion, err := actions_model.GetTasksVersionByScope(ctx, runner.OwnerID, runner.RepoID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("query tasks version failed: %w", err))
	} else if latestVersion == 0 {
		if err := actions_model.IncreaseTaskVersion(ctx, runner.OwnerID, runner.RepoID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fail to increase task version: %w", err))
		}
		// if we don't increase the value of `latestVersion` here,
		// the response of FetchTask will return tasksVersion as zero.
		// and the runner will treat it as an old version of Gitea.
		latestVersion++
	}

	if tasksVersion != latestVersion {
		// if the task version in request is not equal to the version in db,
		// it means there may still be some tasks not be assigned.
		// try to pick a task for the runner that send the request.
		if t, ok, err := actions_service.PickTask(ctx, runner); err != nil {
			log.Error("pick task failed: %v", err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("pick task: %w", err))
		} else if ok {
			task = t
		}
	}
	res := connect.NewResponse(&runnerv1.FetchTaskResponse{
		Task:         task,
		TasksVersion: latestVersion,
	})
	return res, nil
}

// UpdateTask updates the task status.
func (s *Service) UpdateTask(
	ctx context.Context,
	req *connect.Request[runnerv1.UpdateTaskRequest],
) (*connect.Response[runnerv1.UpdateTaskResponse], error) {
	runner := GetRunner(ctx)

	task, err := actions_service.UpdateTaskByState(ctx, runner.ID, req.Msg.State)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update task: %w", err))
	}

	for k, v := range req.Msg.Outputs {
		if len(k) > 255 {
			log.Warn("Ignore the output of task %d because the key is too long: %q", task.ID, k)
			continue
		}
		// The value can be a maximum of 1 MB
		if l := len(v); l > 1024*1024 {
			log.Warn("Ignore the output %q of task %d because the value is too long: %v", k, task.ID, l)
			continue
		}
		// There's another limitation on GitHub that the total of all outputs in a workflow run can be a maximum of 50 MB.
		// We don't check the total size here because it's not easy to do, and it doesn't really worth it.
		// See https://docs.github.com/en/actions/using-jobs/defining-outputs-for-jobs

		if err := actions_model.InsertTaskOutputIfNotExist(ctx, task.ID, k, v); err != nil {
			log.Warn("Failed to insert the output %q of task %d: %v", k, task.ID, err)
			// It's ok not to return errors, the runner will resend the outputs.
		}
	}
	sentOutputs, err := actions_model.FindTaskOutputKeyByTaskID(ctx, task.ID)
	if err != nil {
		log.Warn("Failed to find the sent outputs of task %d: %v", task.ID, err)
		// It's not to return errors, it can be handled when the runner resends sent outputs.
	}

	if err := task.LoadJob(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load job: %w", err))
	}
	if err := task.Job.LoadRun(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load run: %w", err))
	}

	// don't create commit status for cron job
	if task.Job.Run.ScheduleID == 0 {
		actions_service.CreateCommitStatus(ctx, task.Job)
	}

	if req.Msg.State.Result != runnerv1.Result_RESULT_UNSPECIFIED {
		if err := actions_service.EmitJobsIfReady(task.Job.RunID); err != nil {
			log.Error("Emit ready jobs of run %d: %v", task.Job.RunID, err)
		}
	}

	return connect.NewResponse(&runnerv1.UpdateTaskResponse{
		State: &runnerv1.TaskState{
			Id:     req.Msg.State.Id,
			Result: task.Status.AsResult(),
		},
		SentOutputs: sentOutputs,
	}), nil
}

// UpdateLog uploads log of the task.
func (s *Service) UpdateLog(
	ctx context.Context,
	req *connect.Request[runnerv1.UpdateLogRequest],
) (*connect.Response[runnerv1.UpdateLogResponse], error) {
	runner := GetRunner(ctx)

	res := connect.NewResponse(&runnerv1.UpdateLogResponse{})

	task, err := actions_model.GetTaskByID(ctx, req.Msg.TaskId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get task: %w", err))
	} else if runner.ID != task.RunnerID {
		return nil, connect.NewError(connect.CodeInternal, errors.New("invalid runner for task"))
	}
	ack := task.LogLength

	if len(req.Msg.Rows) == 0 || req.Msg.Index > ack || int64(len(req.Msg.Rows))+req.Msg.Index <= ack {
		res.Msg.AckIndex = ack
		return res, nil
	}

	if task.LogInStorage {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("log file has been archived"))
	}

	rows := req.Msg.Rows[ack-req.Msg.Index:]
	ns, err := actions.WriteLogs(ctx, task.LogFilename, task.LogSize, rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("write logs: %w", err))
	}
	task.LogLength += int64(len(rows))
	for _, n := range ns {
		task.LogIndexes = append(task.LogIndexes, task.LogSize)
		task.LogSize += int64(n)
	}

	res.Msg.AckIndex = task.LogLength

	var remove func()
	if req.Msg.NoMore {
		task.LogInStorage = true
		remove, err = actions.TransferLogs(ctx, task.LogFilename)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("transfer logs: %w", err))
		}
	}

	if err := actions_model.UpdateTask(ctx, task, "log_indexes", "log_length", "log_size", "log_in_storage"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update task: %w", err))
	}
	if remove != nil {
		remove()
	}

	return res, nil
}
