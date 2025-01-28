// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessStacktraces(t *testing.T) {
	_, _, finish := GetManager().AddContext(context.Background(), "Normal process")
	defer finish()
	parentCtx, _, finish := GetManager().AddContext(context.Background(), "Children normal process")
	defer finish()
	_, _, finish = GetManager().AddContext(parentCtx, "Children process")
	defer finish()
	_, _, finish = GetManager().AddTypedContext(context.Background(), "System process", SystemProcessType, true)
	defer finish()

	t.Run("No flat with no system process", func(t *testing.T) {
		processes, processCount, _, err := GetManager().ProcessStacktraces(false, true)
		require.NoError(t, err)
		assert.EqualValues(t, 4, processCount)
		assert.Len(t, processes, 2)

		assert.EqualValues(t, "Children normal process", processes[0].Description)
		assert.EqualValues(t, NormalProcessType, processes[0].Type)
		assert.Empty(t, processes[0].ParentPID)
		assert.Len(t, processes[0].Children, 1)

		assert.EqualValues(t, "Children process", processes[0].Children[0].Description)
		assert.EqualValues(t, processes[0].PID, processes[0].Children[0].ParentPID)

		assert.EqualValues(t, "Normal process", processes[1].Description)
		assert.EqualValues(t, NormalProcessType, processes[1].Type)
		assert.Empty(t, processes[1].ParentPID)
		assert.Empty(t, processes[1].Children)
	})

	t.Run("Flat with no system process", func(t *testing.T) {
		processes, processCount, _, err := GetManager().ProcessStacktraces(true, true)
		require.NoError(t, err)
		assert.EqualValues(t, 4, processCount)
		assert.Len(t, processes, 3)

		assert.EqualValues(t, "Children process", processes[0].Description)
		assert.EqualValues(t, NormalProcessType, processes[0].Type)
		assert.EqualValues(t, processes[1].PID, processes[0].ParentPID)
		assert.Empty(t, processes[0].Children)

		assert.EqualValues(t, "Children normal process", processes[1].Description)
		assert.EqualValues(t, NormalProcessType, processes[1].Type)
		assert.Empty(t, processes[1].ParentPID)
		assert.Empty(t, processes[1].Children)

		assert.EqualValues(t, "Normal process", processes[2].Description)
		assert.EqualValues(t, NormalProcessType, processes[2].Type)
		assert.Empty(t, processes[2].ParentPID)
		assert.Empty(t, processes[2].Children)
	})

	t.Run("System process", func(t *testing.T) {
		processes, processCount, _, err := GetManager().ProcessStacktraces(false, false)
		require.NoError(t, err)
		assert.EqualValues(t, 4, processCount)
		assert.Len(t, processes, 4)

		assert.EqualValues(t, "System process", processes[0].Description)
		assert.EqualValues(t, SystemProcessType, processes[0].Type)
		assert.Empty(t, processes[0].ParentPID)
		assert.Empty(t, processes[0].Children)

		assert.EqualValues(t, "Children normal process", processes[1].Description)
		assert.EqualValues(t, NormalProcessType, processes[1].Type)
		assert.Empty(t, processes[1].ParentPID)
		assert.Len(t, processes[1].Children, 1)

		assert.EqualValues(t, "Normal process", processes[2].Description)
		assert.EqualValues(t, NormalProcessType, processes[2].Type)
		assert.Empty(t, processes[2].ParentPID)
		assert.Empty(t, processes[2].Children)

		// This is the "main" pid, testing code always runs in a goroutine.
		assert.EqualValues(t, "(unassociated)", processes[3].Description)
		assert.EqualValues(t, NoneProcessType, processes[3].Type)
		assert.Empty(t, processes[3].ParentPID)
		assert.Empty(t, processes[3].Children)
	})
}
