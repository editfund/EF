// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_18 //nolint

import (
	"forgejo.org/modules/setting"

	"xorm.io/xorm"
)

func AlterPublicGPGKeyContentFieldsToMediumText(x *xorm.Engine) error {
	sess := x.NewSession()
	defer sess.Close()
	if err := sess.Begin(); err != nil {
		return err
	}

	if setting.Database.Type.IsMySQL() {
		if _, err := sess.Exec("ALTER TABLE `gpg_key` CHANGE `content` `content` MEDIUMTEXT"); err != nil {
			return err
		}
		if _, err := sess.Exec("ALTER TABLE `public_key` CHANGE `content` `content` MEDIUMTEXT"); err != nil {
			return err
		}
	}
	return sess.Commit()
}
