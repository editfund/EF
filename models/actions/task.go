// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"context"
	"crypto/subtle"
	"fmt"
	"time"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/models/unit"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/nektos/act/pkg/jobparser"
	"xorm.io/builder"
)

// ActionTask represents a distribution of job
type ActionTask struct {
	ID       int64
	JobID    int64
	Job      *ActionRunJob     `xorm:"-"`
	Steps    []*ActionTaskStep `xorm:"-"`
	Attempt  int64
	RunnerID int64              `xorm:"index"`
	Status   Status             `xorm:"index"`
	Started  timeutil.TimeStamp `xorm:"index"`
	Stopped  timeutil.TimeStamp `xorm:"index(stopped_log_expired)"`

	RepoID            int64  `xorm:"index"`
	OwnerID           int64  `xorm:"index"`
	CommitSHA         string `xorm:"index"`
	IsForkPullRequest bool

	Token          string `xorm:"-"`
	TokenHash      string `xorm:"UNIQUE"` // sha256 of token
	TokenSalt      string
	TokenLastEight string `xorm:"index token_last_eight"`

	LogFilename  string     // file name of log
	LogInStorage bool       // read log from database or from storage
	LogLength    int64      // lines count
	LogSize      int64      // blob size
	LogIndexes   LogIndexes `xorm:"LONGBLOB"`                   // line number to offset
	LogExpired   bool       `xorm:"index(stopped_log_expired)"` // files that are too old will be deleted

	Created timeutil.TimeStamp `xorm:"created"`
	Updated timeutil.TimeStamp `xorm:"updated index"`
}

var successfulTokenTaskCache *lru.Cache[string, any]

func init() {
	db.RegisterModel(new(ActionTask), func() error {
		if setting.SuccessfulTokensCacheSize > 0 {
			var err error
			successfulTokenTaskCache, err = lru.New[string, any](setting.SuccessfulTokensCacheSize)
			if err != nil {
				return fmt.Errorf("unable to allocate Task cache: %v", err)
			}
		} else {
			successfulTokenTaskCache = nil
		}
		return nil
	})
}

func (task *ActionTask) Duration() time.Duration {
	return calculateDuration(task.Started, task.Stopped, task.Status)
}

func (task *ActionTask) IsStopped() bool {
	return task.Stopped > 0
}

func (task *ActionTask) GetRunLink() string {
	if task.Job == nil || task.Job.Run == nil {
		return ""
	}
	return task.Job.Run.Link()
}

func (task *ActionTask) GetCommitLink() string {
	if task.Job == nil || task.Job.Run == nil || task.Job.Run.Repo == nil {
		return ""
	}
	return task.Job.Run.Repo.CommitLink(task.CommitSHA)
}

func (task *ActionTask) GetRepoName() string {
	if task.Job == nil || task.Job.Run == nil || task.Job.Run.Repo == nil {
		return ""
	}
	return task.Job.Run.Repo.FullName()
}

func (task *ActionTask) GetRepoLink() string {
	if task.Job == nil || task.Job.Run == nil || task.Job.Run.Repo == nil {
		return ""
	}
	return task.Job.Run.Repo.Link()
}

func (task *ActionTask) LoadJob(ctx context.Context) error {
	if task.Job == nil {
		job, err := GetRunJobByID(ctx, task.JobID)
		if err != nil {
			return err
		}
		task.Job = job
	}
	return nil
}

// LoadAttributes load Job Steps if not loaded
func (task *ActionTask) LoadAttributes(ctx context.Context) error {
	if task == nil {
		return nil
	}
	if err := task.LoadJob(ctx); err != nil {
		return err
	}

	if err := task.Job.LoadAttributes(ctx); err != nil {
		return err
	}

	if task.Steps == nil { // be careful, an empty slice (not nil) also means loaded
		steps, err := GetTaskStepsByTaskID(ctx, task.ID)
		if err != nil {
			return err
		}
		task.Steps = steps
	}

	return nil
}

func (task *ActionTask) GenerateToken() (err error) {
	task.Token, task.TokenSalt, task.TokenHash, task.TokenLastEight, err = generateSaltedToken()
	return err
}

func GetTaskByID(ctx context.Context, id int64) (*ActionTask, error) {
	var task ActionTask
	has, err := db.GetEngine(ctx).Where("id=?", id).Get(&task)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, fmt.Errorf("task with id %d: %w", id, util.ErrNotExist)
	}

	return &task, nil
}

func GetRunningTaskByToken(ctx context.Context, token string) (*ActionTask, error) {
	errNotExist := fmt.Errorf("task with token %q: %w", token, util.ErrNotExist)
	if token == "" {
		return nil, errNotExist
	}
	// A token is defined as being SHA1 sum these are 40 hexadecimal bytes long
	if len(token) != 40 {
		return nil, errNotExist
	}
	for _, x := range []byte(token) {
		if x < '0' || (x > '9' && x < 'a') || x > 'f' {
			return nil, errNotExist
		}
	}

	lastEight := token[len(token)-8:]

	if id := getTaskIDFromCache(token); id > 0 {
		task := &ActionTask{
			TokenLastEight: lastEight,
		}
		// Re-get the task from the db in case it has been deleted in the intervening period
		has, err := db.GetEngine(ctx).ID(id).Get(task)
		if err != nil {
			return nil, err
		}
		if has {
			return task, nil
		}
		successfulTokenTaskCache.Remove(token)
	}

	var tasks []*ActionTask
	err := db.GetEngine(ctx).Where("token_last_eight = ? AND status = ?", lastEight, StatusRunning).Find(&tasks)
	if err != nil {
		return nil, err
	} else if len(tasks) == 0 {
		return nil, errNotExist
	}

	for _, t := range tasks {
		tempHash := auth_model.HashToken(token, t.TokenSalt)
		if subtle.ConstantTimeCompare([]byte(t.TokenHash), []byte(tempHash)) == 1 {
			if successfulTokenTaskCache != nil {
				successfulTokenTaskCache.Add(token, t.ID)
			}
			return t, nil
		}
	}
	return nil, errNotExist
}

func CreateTaskForRunner(ctx context.Context, runner *ActionRunner) (*ActionTask, bool, error) {
	ctx, commiter, err := db.TxContext(ctx)
	if err != nil {
		return nil, false, err
	}
	defer commiter.Close()

	e := db.GetEngine(ctx)

	jobCond := builder.NewCond()
	if runner.RepoID != 0 {
		jobCond = builder.Eq{"repo_id": runner.RepoID}
	} else if runner.OwnerID != 0 {
		jobCond = builder.In("repo_id", builder.Select("`repository`.id").From("repository").
			Join("INNER", "repo_unit", "`repository`.id = `repo_unit`.repo_id").
			Where(builder.Eq{"`repository`.owner_id": runner.OwnerID, "`repo_unit`.type": unit.TypeActions}))
	}
	if jobCond.IsValid() {
		jobCond = builder.In("run_id", builder.Select("id").From("action_run").Where(jobCond))
	}

	var jobs []*ActionRunJob
	if err := e.Where("task_id=? AND status=?", 0, StatusWaiting).And(jobCond).Asc("updated", "id").Find(&jobs); err != nil {
		return nil, false, err
	}

	// TODO: a more efficient way to filter labels
	var job *ActionRunJob
	log.Trace("runner labels: %v", runner.AgentLabels)
	for _, v := range jobs {
		if v.ItRunsOn(runner.AgentLabels) {
			job = v
			break
		}
	}
	if job == nil {
		return nil, false, nil
	}
	if err := job.LoadAttributes(ctx); err != nil {
		return nil, false, err
	}

	now := timeutil.TimeStampNow()
	job.Attempt++
	job.Started = now
	job.Status = StatusRunning

	task := &ActionTask{
		JobID:             job.ID,
		Attempt:           job.Attempt,
		RunnerID:          runner.ID,
		Started:           now,
		Status:            StatusRunning,
		RepoID:            job.RepoID,
		OwnerID:           job.OwnerID,
		CommitSHA:         job.CommitSHA,
		IsForkPullRequest: job.IsForkPullRequest,
	}
	if err := task.GenerateToken(); err != nil {
		return nil, false, err
	}

	var workflowJob *jobparser.Job
	if gots, err := jobparser.Parse(job.WorkflowPayload); err != nil {
		return nil, false, fmt.Errorf("parse workflow of job %d: %w", job.ID, err)
	} else if len(gots) != 1 {
		return nil, false, fmt.Errorf("workflow of job %d: not single workflow", job.ID)
	} else { //nolint:revive
		_, workflowJob = gots[0].Job()
	}

	if _, err := e.Insert(task); err != nil {
		return nil, false, err
	}

	task.LogFilename = logFileName(job.Run.Repo.FullName(), task.ID)
	if err := UpdateTask(ctx, task, "log_filename"); err != nil {
		return nil, false, err
	}

	if len(workflowJob.Steps) > 0 {
		steps := make([]*ActionTaskStep, len(workflowJob.Steps))
		for i, v := range workflowJob.Steps {
			name, _ := util.SplitStringAtByteN(v.String(), 255)
			steps[i] = &ActionTaskStep{
				Name:   name,
				TaskID: task.ID,
				Index:  int64(i),
				RepoID: task.RepoID,
				Status: StatusWaiting,
			}
		}
		if _, err := e.Insert(steps); err != nil {
			return nil, false, err
		}
		task.Steps = steps
	}

	job.TaskID = task.ID
	// We never have to send a notification here because the job is started with a not done status.
	if n, err := UpdateRunJobWithoutNotification(ctx, job, builder.Eq{"task_id": 0}); err != nil {
		return nil, false, err
	} else if n != 1 {
		return nil, false, nil
	}

	task.Job = job

	if err := commiter.Commit(); err != nil {
		return nil, false, err
	}

	return task, true, nil
}

func UpdateTask(ctx context.Context, task *ActionTask, cols ...string) error {
	sess := db.GetEngine(ctx).ID(task.ID)
	if len(cols) > 0 {
		sess.Cols(cols...)
	}
	_, err := sess.Update(task)
	return err
}

func FindOldTasksToExpire(ctx context.Context, olderThan timeutil.TimeStamp, limit int) ([]*ActionTask, error) {
	e := db.GetEngine(ctx)

	tasks := make([]*ActionTask, 0, limit)
	// Check "stopped > 0" to avoid deleting tasks that are still running
	return tasks, e.Where("stopped > 0 AND stopped < ? AND log_expired = ?", olderThan, false).
		Limit(limit).
		Find(&tasks)
}

func logFileName(repoFullName string, taskID int64) string {
	ret := fmt.Sprintf("%s/%02x/%d.log", repoFullName, taskID%256, taskID)

	if setting.Actions.LogCompression.IsZstd() {
		ret += ".zst"
	}

	return ret
}

func getTaskIDFromCache(token string) int64 {
	if successfulTokenTaskCache == nil {
		return 0
	}
	tInterface, ok := successfulTokenTaskCache.Get(token)
	if !ok {
		return 0
	}
	t, ok := tInterface.(int64)
	if !ok {
		return 0
	}
	return t
}
