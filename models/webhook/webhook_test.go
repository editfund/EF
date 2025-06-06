// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"testing"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/json"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/timeutil"
	webhook_module "forgejo.org/modules/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHookContentType_Name(t *testing.T) {
	assert.Equal(t, "json", ContentTypeJSON.Name())
	assert.Equal(t, "form", ContentTypeForm.Name())
}

func TestIsValidHookContentType(t *testing.T) {
	assert.True(t, IsValidHookContentType("json"))
	assert.True(t, IsValidHookContentType("form"))
	assert.False(t, IsValidHookContentType("invalid"))
}

func TestWebhook_History(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	webhook := unittest.AssertExistsAndLoadBean(t, &Webhook{ID: 1})
	tasks, err := webhook.History(db.DefaultContext, 0)
	require.NoError(t, err)
	if assert.Len(t, tasks, 3) {
		assert.Equal(t, int64(3), tasks[0].ID)
		assert.Equal(t, int64(2), tasks[1].ID)
		assert.Equal(t, int64(1), tasks[2].ID)
	}

	webhook = unittest.AssertExistsAndLoadBean(t, &Webhook{ID: 2})
	tasks, err = webhook.History(db.DefaultContext, 0)
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestWebhook_UpdateEvent(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	webhook := unittest.AssertExistsAndLoadBean(t, &Webhook{ID: 1})
	hookEvent := &webhook_module.HookEvent{
		PushOnly:       true,
		SendEverything: false,
		ChooseEvents:   false,
		HookEvents: webhook_module.HookEvents{
			Create:      false,
			Push:        true,
			PullRequest: false,
		},
	}
	webhook.HookEvent = hookEvent
	require.NoError(t, webhook.UpdateEvent())
	assert.NotEmpty(t, webhook.Events)
	actualHookEvent := &webhook_module.HookEvent{}
	require.NoError(t, json.Unmarshal([]byte(webhook.Events), actualHookEvent))
	assert.Equal(t, *hookEvent, *actualHookEvent)
}

func TestWebhook_EventsArray(t *testing.T) {
	assert.Equal(t, []string{
		"create", "delete", "fork", "push",
		"issues", "issue_assign", "issue_label", "issue_milestone", "issue_comment",
		"pull_request", "pull_request_assign", "pull_request_label", "pull_request_milestone",
		"pull_request_comment", "pull_request_review_approved", "pull_request_review_rejected",
		"pull_request_review_comment", "pull_request_sync", "wiki", "repository", "release",
		"package", "pull_request_review_request",
	},
		(&Webhook{
			HookEvent: &webhook_module.HookEvent{SendEverything: true},
		}).EventsArray(),
	)

	assert.Equal(t, []string{"push"},
		(&Webhook{
			HookEvent: &webhook_module.HookEvent{PushOnly: true},
		}).EventsArray(),
	)
}

func TestCreateWebhook(t *testing.T) {
	hook := &Webhook{
		RepoID:      3,
		URL:         "https://www.example.com/unit_test",
		ContentType: ContentTypeJSON,
		Events:      `{"push_only":false,"send_everything":false,"choose_events":false,"events":{"create":false,"push":true,"pull_request":true}}`,
	}
	unittest.AssertNotExistsBean(t, hook)
	require.NoError(t, CreateWebhook(db.DefaultContext, hook))
	unittest.AssertExistsAndLoadBean(t, hook)
}

func TestGetWebhookByRepoID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	hook, err := GetWebhookByRepoID(db.DefaultContext, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), hook.ID)

	_, err = GetWebhookByRepoID(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID)
	require.Error(t, err)
	assert.True(t, IsErrWebhookNotExist(err))
}

func TestGetWebhookByOwnerID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	hook, err := GetWebhookByOwnerID(db.DefaultContext, 3, 3)
	require.NoError(t, err)
	assert.Equal(t, int64(3), hook.ID)

	_, err = GetWebhookByOwnerID(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID)
	require.Error(t, err)
	assert.True(t, IsErrWebhookNotExist(err))
}

func TestGetActiveWebhooksByRepoID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	activateWebhook(t, 1)

	hooks, err := db.Find[Webhook](db.DefaultContext, ListWebhookOptions{RepoID: 1, IsActive: optional.Some(true)})
	require.NoError(t, err)
	if assert.Len(t, hooks, 1) {
		assert.Equal(t, int64(1), hooks[0].ID)
		assert.True(t, hooks[0].IsActive)
	}
}

func TestGetWebhooksByRepoID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	hooks, err := db.Find[Webhook](db.DefaultContext, ListWebhookOptions{RepoID: 1})
	require.NoError(t, err)
	if assert.Len(t, hooks, 2) {
		assert.Equal(t, int64(1), hooks[0].ID)
		assert.Equal(t, int64(2), hooks[1].ID)
	}
}

func TestGetActiveWebhooksByOwnerID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	activateWebhook(t, 3)

	hooks, err := db.Find[Webhook](db.DefaultContext, ListWebhookOptions{OwnerID: 3, IsActive: optional.Some(true)})
	require.NoError(t, err)
	if assert.Len(t, hooks, 1) {
		assert.Equal(t, int64(3), hooks[0].ID)
		assert.True(t, hooks[0].IsActive)
	}
}

func activateWebhook(t *testing.T, hookID int64) {
	t.Helper()
	updated, err := db.GetEngine(db.DefaultContext).ID(hookID).Cols("is_active").Update(Webhook{IsActive: true})
	assert.Equal(t, int64(1), updated)
	require.NoError(t, err)
}

func TestGetWebhooksByOwnerID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	activateWebhook(t, 3)

	hooks, err := db.Find[Webhook](db.DefaultContext, ListWebhookOptions{OwnerID: 3})
	require.NoError(t, err)
	if assert.Len(t, hooks, 1) {
		assert.Equal(t, int64(3), hooks[0].ID)
		assert.True(t, hooks[0].IsActive)
	}
}

func TestUpdateWebhook(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	hook := unittest.AssertExistsAndLoadBean(t, &Webhook{ID: 2})
	hook.IsActive = true
	hook.ContentType = ContentTypeForm
	unittest.AssertNotExistsBean(t, hook)
	require.NoError(t, UpdateWebhook(db.DefaultContext, hook))
	unittest.AssertExistsAndLoadBean(t, hook)
}

func TestDeleteWebhookByRepoID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	unittest.AssertExistsAndLoadBean(t, &Webhook{ID: 2, RepoID: 1})
	require.NoError(t, DeleteWebhookByRepoID(db.DefaultContext, 1, 2))
	unittest.AssertNotExistsBean(t, &Webhook{ID: 2, RepoID: 1})

	err := DeleteWebhookByRepoID(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID)
	require.Error(t, err)
	assert.True(t, IsErrWebhookNotExist(err))
}

func TestDeleteWebhookByOwnerID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	unittest.AssertExistsAndLoadBean(t, &Webhook{ID: 3, OwnerID: 3})
	require.NoError(t, DeleteWebhookByOwnerID(db.DefaultContext, 3, 3))
	unittest.AssertNotExistsBean(t, &Webhook{ID: 3, OwnerID: 3})

	err := DeleteWebhookByOwnerID(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID)
	require.Error(t, err)
	assert.True(t, IsErrWebhookNotExist(err))
}

func TestHookTasks(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	hookTasks, err := HookTasks(db.DefaultContext, 1, 1)
	require.NoError(t, err)
	if assert.Len(t, hookTasks, 3) {
		assert.Equal(t, int64(3), hookTasks[0].ID)
		assert.Equal(t, int64(2), hookTasks[1].ID)
		assert.Equal(t, int64(1), hookTasks[2].ID)
	}

	hookTasks, err = HookTasks(db.DefaultContext, unittest.NonexistentID, 1)
	require.NoError(t, err)
	assert.Empty(t, hookTasks)
}

func TestCreateHookTask(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	hookTask := &HookTask{
		HookID:         3,
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(db.DefaultContext, hookTask)
	require.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)
}

func TestUpdateHookTask(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	hook := unittest.AssertExistsAndLoadBean(t, &HookTask{ID: 1})
	hook.PayloadContent = "new payload content"
	hook.IsDelivered = true
	unittest.AssertNotExistsBean(t, hook)
	require.NoError(t, UpdateHookTask(db.DefaultContext, hook))
	unittest.AssertExistsAndLoadBean(t, hook)
}

func TestCleanupHookTaskTable_PerWebhook_DeletesDelivered(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	hookTask := &HookTask{
		HookID:         3,
		IsDelivered:    true,
		Delivered:      timeutil.TimeStampNanoNow(),
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(db.DefaultContext, hookTask)
	require.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)

	require.NoError(t, CleanupHookTaskTable(t.Context(), PerWebhook, 168*time.Hour, 0))
	unittest.AssertNotExistsBean(t, hookTask)
}

func TestCleanupHookTaskTable_PerWebhook_LeavesUndelivered(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	hookTask := &HookTask{
		HookID:         4,
		IsDelivered:    false,
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(db.DefaultContext, hookTask)
	require.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)

	require.NoError(t, CleanupHookTaskTable(t.Context(), PerWebhook, 168*time.Hour, 0))
	unittest.AssertExistsAndLoadBean(t, hookTask)
}

func TestCleanupHookTaskTable_PerWebhook_LeavesMostRecentTask(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	hookTask := &HookTask{
		HookID:         4,
		IsDelivered:    true,
		Delivered:      timeutil.TimeStampNanoNow(),
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(db.DefaultContext, hookTask)
	require.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)

	require.NoError(t, CleanupHookTaskTable(t.Context(), PerWebhook, 168*time.Hour, 1))
	unittest.AssertExistsAndLoadBean(t, hookTask)
}

func TestCleanupHookTaskTable_OlderThan_DeletesDelivered(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	hookTask := &HookTask{
		HookID:         3,
		IsDelivered:    true,
		Delivered:      timeutil.TimeStampNano(time.Now().AddDate(0, 0, -8).UnixNano()),
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(db.DefaultContext, hookTask)
	require.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)

	require.NoError(t, CleanupHookTaskTable(t.Context(), OlderThan, 168*time.Hour, 0))
	unittest.AssertNotExistsBean(t, hookTask)
}

func TestCleanupHookTaskTable_OlderThan_LeavesUndelivered(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	hookTask := &HookTask{
		HookID:         4,
		IsDelivered:    false,
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(db.DefaultContext, hookTask)
	require.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)

	require.NoError(t, CleanupHookTaskTable(t.Context(), OlderThan, 168*time.Hour, 0))
	unittest.AssertExistsAndLoadBean(t, hookTask)
}

func TestCleanupHookTaskTable_OlderThan_LeavesTaskEarlierThanAgeToDelete(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	hookTask := &HookTask{
		HookID:         4,
		IsDelivered:    true,
		Delivered:      timeutil.TimeStampNano(time.Now().AddDate(0, 0, -6).UnixNano()),
		PayloadVersion: 2,
	}
	unittest.AssertNotExistsBean(t, hookTask)
	_, err := CreateHookTask(db.DefaultContext, hookTask)
	require.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, hookTask)

	require.NoError(t, CleanupHookTaskTable(t.Context(), OlderThan, 168*time.Hour, 0))
	unittest.AssertExistsAndLoadBean(t, hookTask)
}
