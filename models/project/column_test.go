// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"fmt"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	projectWithoutDefault, err := GetProjectByID(db.DefaultContext, 5)
	require.NoError(t, err)

	// check if default column was added
	column, err := projectWithoutDefault.GetDefaultColumn(db.DefaultContext)
	require.NoError(t, err)
	assert.Equal(t, int64(5), column.ProjectID)
	assert.Equal(t, "Uncategorized", column.Title)

	projectWithMultipleDefaults, err := GetProjectByID(db.DefaultContext, 6)
	require.NoError(t, err)

	// check if multiple defaults were removed
	column, err = projectWithMultipleDefaults.GetDefaultColumn(db.DefaultContext)
	require.NoError(t, err)
	assert.Equal(t, int64(6), column.ProjectID)
	assert.Equal(t, int64(9), column.ID)

	// set 8 as default column
	require.NoError(t, SetDefaultColumn(db.DefaultContext, column.ProjectID, 8))

	// then 9 will become a non-default column
	column, err = GetColumn(db.DefaultContext, 9)
	require.NoError(t, err)
	assert.Equal(t, int64(6), column.ProjectID)
	assert.False(t, column.Default)
}

func Test_moveIssuesToAnotherColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	column1 := unittest.AssertExistsAndLoadBean(t, &Column{ID: 1, ProjectID: 1})

	issues, err := column1.GetIssues(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.EqualValues(t, 1, issues[0].ID)

	column2 := unittest.AssertExistsAndLoadBean(t, &Column{ID: 2, ProjectID: 1})
	issues, err = column2.GetIssues(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.EqualValues(t, 3, issues[0].ID)

	err = column1.moveIssuesToAnotherColumn(db.DefaultContext, column2)
	require.NoError(t, err)

	issues, err = column1.GetIssues(db.DefaultContext)
	require.NoError(t, err)
	assert.Empty(t, issues)

	issues, err = column2.GetIssues(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.EqualValues(t, 3, issues[0].ID)
	assert.EqualValues(t, 0, issues[0].Sorting)
	assert.EqualValues(t, 1, issues[1].ID)
	assert.EqualValues(t, 1, issues[1].Sorting)
}

func Test_MoveColumnsOnProject(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	project1 := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})
	columns, err := project1.GetColumns(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, columns, 3)
	assert.EqualValues(t, 0, columns[0].Sorting) // even if there is no default sorting, the code should also work
	assert.EqualValues(t, 0, columns[1].Sorting)
	assert.EqualValues(t, 0, columns[2].Sorting)

	err = MoveColumnsOnProject(db.DefaultContext, project1, map[int64]int64{
		0: columns[1].ID,
		1: columns[2].ID,
		2: columns[0].ID,
	})
	require.NoError(t, err)

	columnsAfter, err := project1.GetColumns(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, columnsAfter, 3)
	assert.Equal(t, columns[1].ID, columnsAfter[0].ID)
	assert.Equal(t, columns[2].ID, columnsAfter[1].ID)
	assert.Equal(t, columns[0].ID, columnsAfter[2].ID)
}

func Test_NewColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	project1 := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})
	columns, err := project1.GetColumns(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, columns, 3)

	for i := 0; i < maxProjectColumns-3; i++ {
		err := NewColumn(db.DefaultContext, &Column{
			Title:     fmt.Sprintf("column-%d", i+4),
			ProjectID: project1.ID,
		})
		require.NoError(t, err)
	}
	err = NewColumn(db.DefaultContext, &Column{
		Title:     "column-21",
		ProjectID: project1.ID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum number of columns reached")
}
