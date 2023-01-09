package repo

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pkg/errors"
	"os"
	"regexp"
)

const (
	repoReTemplate = "^(.+)/(.+)/commit/(.+)"
)

type Info struct {
	RepoLink, CommitHash, ProjectName string
}

func (inf *Info) String() string {
	return fmt.Sprintf("Repo: %s, Commit: %s, Project: %s", inf.RepoLink, inf.CommitHash, inf.ProjectName)
}

func ParseRepoLink(repoLink string) (Info, error) {
	r := regexp.MustCompile(repoReTemplate)
	groups := r.FindSubmatch([]byte(repoLink))
	if len(groups) != 4 {
		return Info{}, errors.Errorf("Failed to parse repo link")
	}

	return Info{
		RepoLink:    string(groups[1]) + "/" + string(groups[2]),
		ProjectName: string(groups[2]),
		CommitHash:  string(groups[3]),
	}, nil
}

func GenTempDirPath() string {
	return fmt.Sprintf("/tmp/sc_deployer.%d", os.Getpid())
}

func FetchRepo(repoInfo *Info, repoDir string) (func(), error) {
	cleanupHandler := func() {
		err := os.RemoveAll(repoDir)
		if err != nil {
			panic(err)
		}
	}

	repo, err := git.PlainClone(
		repoDir,
		false,
		&git.CloneOptions{
			URL:        repoInfo.RepoLink,
			NoCheckout: true,
		},
	)
	if err != nil {
		return cleanupHandler, err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return cleanupHandler, err
	}

	if err = worktree.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(repoInfo.CommitHash)}); err != nil {
		return cleanupHandler, err
	}

	return cleanupHandler, nil
}
