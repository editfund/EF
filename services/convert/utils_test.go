// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToCorrectPageSize(t *testing.T) {
	assert.Equal(t, 30, ToCorrectPageSize(0))
	assert.Equal(t, 30, ToCorrectPageSize(-10))
	assert.Equal(t, 20, ToCorrectPageSize(20))
	assert.Equal(t, 50, ToCorrectPageSize(100))
}

func TestToGitServiceType(t *testing.T) {
	tc := []struct {
		typ  string
		enum int
	}{{
		typ: "github", enum: 2,
	}, {
		typ: "gitea", enum: 3,
	}, {
		typ: "gitlab", enum: 4,
	}, {
		typ: "gogs", enum: 5,
	}, {
		typ: "forgejo", enum: 9,
	}, {
		typ: "trash", enum: 1,
	}}
	for _, test := range tc {
		assert.EqualValues(t, test.enum, ToGitServiceType(test.typ))
	}
}
