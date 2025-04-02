// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package organization

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/models/perm"
	"forgejo.org/models/unit"
)

// TeamUnit describes all units of a repository
type TeamUnit struct {
	ID         int64     `xorm:"pk autoincr"`
	OrgID      int64     `xorm:"INDEX"`
	TeamID     int64     `xorm:"UNIQUE(s)"`
	Type       unit.Type `xorm:"UNIQUE(s)"`
	AccessMode perm.AccessMode
}

// Unit returns Unit
func (t *TeamUnit) Unit() unit.Unit {
	return unit.Units[t.Type]
}

func getUnitsByTeamID(ctx context.Context, teamID int64) (units []*TeamUnit, err error) {
	return units, db.GetEngine(ctx).Where("team_id = ?", teamID).Find(&units)
}
