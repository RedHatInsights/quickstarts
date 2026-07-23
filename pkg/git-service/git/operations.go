package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/sirupsen/logrus"
)

type File struct {
	Name    string
	Content string
}

type RepoOperations interface {
	PullLatest() error
	CreateBranch(name string) error
	WriteFiles(dir string, files []File) error
	CommitChanges(message, authorName, authorEmail string) (string, error)
	PushBranch(branch string) error
	Cleanup(branch string) error
	GetBaseBranch() string
}

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

func (m *RepoManager) CreateBranch(name string) error {
	w, err := m.Repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	ref := plumbing.NewBranchReferenceName(name)
	err = w.Checkout(&git.CheckoutOptions{
		Branch: ref,
		Create: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create branch %s: %w", name, err)
	}

	logrus.WithField("branch", name).Info("Branch created")
	return nil
}

func (m *RepoManager) WriteFiles(dir string, files []File) error {
	repoRoot := filepath.Clean(m.RepoPath) + string(os.PathSeparator)
	absDir := filepath.Clean(filepath.Join(m.RepoPath, dir))
	if !strings.HasPrefix(absDir+string(os.PathSeparator), repoRoot) {
		return fmt.Errorf("directory path escapes repository root: %s", dir)
	}

	if err := os.MkdirAll(absDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	for _, f := range files {
		path := filepath.Clean(filepath.Join(absDir, f.Name))
		if !strings.HasPrefix(path, repoRoot) {
			return fmt.Errorf("file path escapes repository root: %s", f.Name)
		}
		if err := os.WriteFile(path, []byte(f.Content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", f.Name, err)
		}
	}

	logrus.WithFields(logrus.Fields{
		"dir":   dir,
		"count": len(files),
	}).Info("Files written")
	return nil
}

func (m *RepoManager) CommitChanges(message, authorName, authorEmail string) (string, error) {
	w, err := m.Repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	if _, err := w.Add("."); err != nil {
		return "", fmt.Errorf("failed to stage changes: %w", err)
	}

	hash, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	logrus.WithField("sha", hash.String()[:8]).Info("Changes committed")
	return hash.String(), nil
}

func (m *RepoManager) PushBranch(branch string) error {
	ref := plumbing.NewBranchReferenceName(branch)
	err := m.Repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec(ref + ":" + ref)},
		Auth:       m.auth(),
	})
	if err != nil {
		return fmt.Errorf("failed to push branch %s: %w", branch, err)
	}

	logrus.WithField("branch", branch).Info("Branch pushed")
	return nil
}

func (m *RepoManager) Cleanup(branch string) error {
	w, err := m.Repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if err := w.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
		return fmt.Errorf("failed to reset worktree: %w", err)
	}

	if err := w.Clean(&git.CleanOptions{Dir: true}); err != nil {
		return fmt.Errorf("failed to clean worktree: %w", err)
	}

	baseRef := plumbing.NewBranchReferenceName(m.BaseBranch)
	if err := w.Checkout(&git.CheckoutOptions{Branch: baseRef}); err != nil {
		return fmt.Errorf("failed to checkout %s: %w", m.BaseBranch, err)
	}

	if err := m.Repo.Storer.RemoveReference(plumbing.NewBranchReferenceName(branch)); err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branch, err)
	}

	logrus.WithField("branch", branch).Info("Branch cleaned up")
	return nil
}

func (m *RepoManager) GetBaseBranch() string {
	return m.BaseBranch
}

func (m *RepoManager) auth() *http.BasicAuth {
	return &http.BasicAuth{
		Username: "git",
		Password: m.Token,
	}
}
