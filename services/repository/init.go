// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"
	"fmt"
	"os"
	"time"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	repo_module "forgejo.org/modules/repository"
	"forgejo.org/modules/setting"
	asymkey_service "forgejo.org/services/asymkey"
)

// initRepoCommit temporarily changes with work directory.
func initRepoCommit(ctx context.Context, tmpPath string, repo *repo_model.Repository, u *user_model.User, defaultBranch string) (err error) {
	commitTimeStr := time.Now().Format(time.RFC3339)

	sig := u.NewGitSig()
	// Because this may call hooks we should pass in the environment
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME="+sig.Name,
		"GIT_AUTHOR_EMAIL="+sig.Email,
		"GIT_AUTHOR_DATE="+commitTimeStr,
		"GIT_COMMITTER_DATE="+commitTimeStr,
	)
	committerName := sig.Name
	committerEmail := sig.Email

	if stdout, _, err := git.NewCommand(ctx, "add", "--all").
		SetDescription(fmt.Sprintf("initRepoCommit (git add): %s", tmpPath)).
		RunStdString(&git.RunOpts{Dir: tmpPath}); err != nil {
		log.Error("git add --all failed: Stdout: %s\nError: %v", stdout, err)
		return fmt.Errorf("git add --all: %w", err)
	}

	cmd := git.NewCommand(ctx, "commit", "--message=Initial commit").
		AddOptionFormat("--author='%s <%s>'", sig.Name, sig.Email)

	sign, keyID, signer, _ := asymkey_service.SignInitialCommit(ctx, tmpPath, u)
	if sign {
		cmd.AddOptionFormat("-S%s", keyID)

		if repo.GetTrustModel() == repo_model.CommitterTrustModel || repo.GetTrustModel() == repo_model.CollaboratorCommitterTrustModel {
			// need to set the committer to the KeyID owner
			committerName = signer.Name
			committerEmail = signer.Email
		}
	} else {
		cmd.AddArguments("--no-gpg-sign")
	}

	env = append(env,
		"GIT_COMMITTER_NAME="+committerName,
		"GIT_COMMITTER_EMAIL="+committerEmail,
	)

	if stdout, _, err := cmd.
		SetDescription(fmt.Sprintf("initRepoCommit (git commit): %s", tmpPath)).
		RunStdString(&git.RunOpts{Dir: tmpPath, Env: env}); err != nil {
		log.Error("Failed to commit: %v: Stdout: %s\nError: %v", cmd.String(), stdout, err)
		return fmt.Errorf("git commit: %w", err)
	}

	if len(defaultBranch) == 0 {
		defaultBranch = setting.Repository.DefaultBranch
	}

	if stdout, _, err := git.NewCommand(ctx, "push", "origin").AddDynamicArguments("HEAD:" + defaultBranch).
		SetDescription(fmt.Sprintf("initRepoCommit (git push): %s", tmpPath)).
		RunStdString(&git.RunOpts{Dir: tmpPath, Env: repo_module.InternalPushingEnvironment(u, repo)}); err != nil {
		log.Error("Failed to push back to HEAD: Stdout: %s\nError: %v", stdout, err)
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}
