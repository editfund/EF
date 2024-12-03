// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"testing"

	actions_model "code.gitea.io/gitea/models/actions"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanup(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Deletes no longer existing logs", func(t *testing.T) {
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionTask{ID: 1001, LogExpired: false, LogIndexes: []int64{1, 2, 3, 4}, LogFilename: "does-not-exist", Stopped: timeutil.TimeStamp(1)})

		require.NoError(t, CleanupLogs(db.DefaultContext))

		task := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 1001})
		assert.EqualValues(t, "does-not-exist", task.LogFilename)
		assert.True(t, task.LogExpired)
		assert.Nil(t, task.LogIndexes)
	})
}
