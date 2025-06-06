// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package access_test

import (
	"testing"

	"forgejo.org/models/db"
	perm_model "forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessLevel(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	user29 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 29})
	// A public repository owned by User 2
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	assert.False(t, repo1.IsPrivate)
	// A private repository owned by Org 3
	repo3 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	assert.True(t, repo3.IsPrivate)

	// Another public repository
	repo4 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})
	assert.False(t, repo4.IsPrivate)
	// org. owned private repo
	repo24 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 24})

	level, err := access_model.AccessLevel(db.DefaultContext, user2, repo1)
	require.NoError(t, err)
	assert.Equal(t, perm_model.AccessModeOwner, level)

	level, err = access_model.AccessLevel(db.DefaultContext, user2, repo3)
	require.NoError(t, err)
	assert.Equal(t, perm_model.AccessModeOwner, level)

	level, err = access_model.AccessLevel(db.DefaultContext, user5, repo1)
	require.NoError(t, err)
	assert.Equal(t, perm_model.AccessModeRead, level)

	level, err = access_model.AccessLevel(db.DefaultContext, user5, repo3)
	require.NoError(t, err)
	assert.Equal(t, perm_model.AccessModeNone, level)

	// restricted user has no access to a public repo
	level, err = access_model.AccessLevel(db.DefaultContext, user29, repo1)
	require.NoError(t, err)
	assert.Equal(t, perm_model.AccessModeNone, level)

	// ... unless he's a collaborator
	level, err = access_model.AccessLevel(db.DefaultContext, user29, repo4)
	require.NoError(t, err)
	assert.Equal(t, perm_model.AccessModeWrite, level)

	// ... or a team member
	level, err = access_model.AccessLevel(db.DefaultContext, user29, repo24)
	require.NoError(t, err)
	assert.Equal(t, perm_model.AccessModeRead, level)
}

func TestHasAccess(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	// A public repository owned by User 2
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	assert.False(t, repo1.IsPrivate)
	// A private repository owned by Org 3
	repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	assert.True(t, repo2.IsPrivate)

	has, err := access_model.HasAccess(db.DefaultContext, user1.ID, repo1)
	require.NoError(t, err)
	assert.True(t, has)

	_, err = access_model.HasAccess(db.DefaultContext, user1.ID, repo2)
	require.NoError(t, err)

	_, err = access_model.HasAccess(db.DefaultContext, user2.ID, repo1)
	require.NoError(t, err)

	_, err = access_model.HasAccess(db.DefaultContext, user2.ID, repo2)
	require.NoError(t, err)
}

func TestRepository_RecalculateAccesses(t *testing.T) {
	// test with organization repo
	require.NoError(t, unittest.PrepareTestDatabase())
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	require.NoError(t, repo1.LoadOwner(db.DefaultContext))

	_, err := db.GetEngine(db.DefaultContext).Delete(&repo_model.Collaboration{UserID: 2, RepoID: 3})
	require.NoError(t, err)
	require.NoError(t, access_model.RecalculateAccesses(db.DefaultContext, repo1))

	access := &access_model.Access{UserID: 2, RepoID: 3}
	has, err := db.GetEngine(db.DefaultContext).Get(access)
	require.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, perm_model.AccessModeOwner, access.Mode)
}

func TestRepository_RecalculateAccesses2(t *testing.T) {
	// test with non-organization repo
	require.NoError(t, unittest.PrepareTestDatabase())
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})
	require.NoError(t, repo1.LoadOwner(db.DefaultContext))

	_, err := db.GetEngine(db.DefaultContext).Delete(&repo_model.Collaboration{UserID: 4, RepoID: 4})
	require.NoError(t, err)
	require.NoError(t, access_model.RecalculateAccesses(db.DefaultContext, repo1))

	has, err := db.GetEngine(db.DefaultContext).Get(&access_model.Access{UserID: 4, RepoID: 4})
	require.NoError(t, err)
	assert.False(t, has)
}
