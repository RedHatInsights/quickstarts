package git

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/sirupsen/logrus"
)

type RepoManager struct {
	Repo       *git.Repository
	RepoPath   string
	Token      string
	BaseBranch string
}

func InitRepo(repoURL, repoPath, token, baseBranch string) (*RepoManager, error) {
	mgr := &RepoManager{
		RepoPath:   repoPath,
		Token:      token,
		BaseBranch: baseBranch,
	}

	if _, err := os.Stat(repoPath); err == nil {
		logrus.WithField("path", repoPath).Info("Repo directory exists, opening")
		repo, err := git.PlainOpen(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open existing repo at %s: %w", repoPath, err)
		}
		mgr.Repo = repo
		return mgr, nil
	}

	logrus.WithFields(logrus.Fields{
		"url":  repoURL,
		"path": repoPath,
	}).Info("Cloning repository")

	repo, err := git.PlainClone(repoPath, false, &git.CloneOptions{
		URL:           repoURL,
		Auth:          mgr.auth(),
		ReferenceName: plumbing.NewBranchReferenceName(baseBranch),
		SingleBranch:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone %s: %w", repoURL, err)
	}

	mgr.Repo = repo
	logrus.Info("Repository cloned successfully")
	return mgr, nil
}

func (m *RepoManager) PullLatest() error {
	w, err := m.Repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	ref := plumbing.NewBranchReferenceName(m.BaseBranch)

	if err := w.Checkout(&git.CheckoutOptions{Branch: ref}); err != nil {
		return fmt.Errorf("failed to checkout %s: %w", m.BaseBranch, err)
	}

	err = w.Pull(&git.PullOptions{
		RemoteName:    "origin",
		ReferenceName: ref,
		Auth:          m.auth(),
		Force:         true,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull latest: %w", err)
	}

	logrus.WithField("branch", m.BaseBranch).Info("Base branch up to date")
	return nil
}

func (m *RepoManager) EnsureRemote(name, url string) error {
	_, err := m.Repo.Remote(name)
	if err == git.ErrRemoteNotFound {
		_, err = m.Repo.CreateRemote(&config.RemoteConfig{
			Name: name,
			URLs: []string{url},
		})
		if err != nil {
			return fmt.Errorf("failed to create remote %s: %w", name, err)
		}
		logrus.WithField("remote", name).Info("Remote added")
		return nil
	}
	return err
}

func (m *RepoManager) auth() *http.BasicAuth {
	return &http.BasicAuth{
		Username: "git",
		Password: m.Token,
	}
}
