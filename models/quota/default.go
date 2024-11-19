// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package quota

import (
	"code.gitea.io/gitea/modules/setting"
)

func EvaluateDefault(used Used, forSubject LimitSubject) bool {
	groups := GroupList{
		&Group{
			Name: "builtin-default-group",
			Rules: []Rule{
				{
					Name:     "builtin-default-rule",
					Limit:    setting.Quota.Default.Total,
					Subjects: LimitSubjects{LimitSubjectSizeAll},
				},
			},
		},
	}

	return groups.Evaluate(used, forSubject)
}
