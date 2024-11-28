// Copyright 2021 Gitea. All rights reserved.
// SPDX-License-Identifier: MIT

package automerge

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/gitrepo"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/queue"
)

// PRAutoMergeQueue represents a queue to handle update pull request tests
var PRAutoMergeQueue *queue.WorkerPoolQueue[string]

func addToQueue(pr *issues_model.PullRequest, sha string) {
	log.Trace("Adding pullID: %d to the pull requests patch checking queue with sha %s", pr.ID, sha)
	if err := PRAutoMergeQueue.Push(fmt.Sprintf("%d_%s", pr.ID, sha)); err != nil {
		log.Error("Error adding pullID: %d to the pull requests patch checking queue %v", pr.ID, err)
	}
}

// StartPRCheckAndAutoMergeBySHA start an automerge check and auto merge task for all pull requests of repository and SHA
func StartPRCheckAndAutoMergeBySHA(ctx context.Context, sha string, repo *repo_model.Repository) error {
	pulls, err := getPullRequestsByHeadSHA(ctx, sha, repo, func(pr *issues_model.PullRequest) bool {
		return !pr.HasMerged && pr.CanAutoMerge()
	})
	if err != nil {
		return err
	}

	for _, pr := range pulls {
		addToQueue(pr, sha)
	}

	return nil
}

// StartPRCheckAndAutoMerge start an automerge check and auto merge task for a pull request
func StartPRCheckAndAutoMerge(ctx context.Context, pull *issues_model.PullRequest) {
	if pull == nil || pull.HasMerged || !pull.CanAutoMerge() {
		return
	}

	if err := pull.LoadBaseRepo(ctx); err != nil {
		log.Error("LoadBaseRepo: %v", err)
		return
	}

	gitRepo, err := gitrepo.OpenRepository(ctx, pull.BaseRepo)
	if err != nil {
		log.Error("OpenRepository: %v", err)
		return
	}
	defer gitRepo.Close()
	commitID, err := gitRepo.GetRefCommitID(pull.GetGitRefName())
	if err != nil {
		log.Error("GetRefCommitID: %v", err)
		return
	}

	addToQueue(pull, commitID)
}

func getPullRequestsByHeadSHA(ctx context.Context, sha string, repo *repo_model.Repository, filter func(*issues_model.PullRequest) bool) (map[int64]*issues_model.PullRequest, error) {
	gitRepo, err := gitrepo.OpenRepository(ctx, repo)
	if err != nil {
		return nil, err
	}
	defer gitRepo.Close()

	refs, err := gitRepo.GetRefsBySha(sha, "")
	if err != nil {
		return nil, err
	}

	pulls := make(map[int64]*issues_model.PullRequest)

	for _, ref := range refs {
		// Each pull branch starts with refs/pull/ we then go from there to find the index of the pr and then
		// use that to get the pr.
		if strings.HasPrefix(ref, git.PullPrefix) {
			parts := strings.Split(ref[len(git.PullPrefix):], "/")

			// e.g. 'refs/pull/1/head' would be []string{"1", "head"}
			if len(parts) != 2 {
				log.Error("getPullRequestsByHeadSHA found broken pull ref [%s] on repo [%-v]", ref, repo)
				continue
			}

			prIndex, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				log.Error("getPullRequestsByHeadSHA found broken pull ref [%s] on repo [%-v]", ref, repo)
				continue
			}

			p, err := issues_model.GetPullRequestByIndex(ctx, repo.ID, prIndex)
			if err != nil {
				// If there is no pull request for this branch, we don't try to merge it.
				if issues_model.IsErrPullRequestNotExist(err) {
					continue
				}
				return nil, err
			}

			if filter(p) {
				pulls[p.ID] = p
			}
		}
	}

	return pulls, nil
}
