// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"os/exec"
	"strings"
	"testing"
)

func getWhoamiOutput() (string, error) {
	output, err := exec.Command("whoami").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func TestCurrentUsername(t *testing.T) {
	user := CurrentUsername()
	if len(user) == 0 {
		t.Errorf("expected non-empty user, got: %s", user)
	}
	whoami, err := getWhoamiOutput()
	if err != nil {
		t.Errorf("failed to run whoami to test current user: %f", err)
	}
	user = CurrentUsername()
	if user != whoami {
		t.Errorf("expected %s as user, got: %s", whoami, user)
	}
	t.Setenv("USER", "spoofed")
	user = CurrentUsername()
	if user != whoami {
		t.Errorf("expected %s as user, got: %s", whoami, user)
	}
}
