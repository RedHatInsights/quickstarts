package git

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
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
	CheckoutExistingBranch(name string) error
	WriteFiles(dir string, files []File) error
	CommitChanges(message, authorName, authorEmail, dir string, files []File) (string, error)
	PushBranch(branch string) error
	PushBranchForce(branch string) error
	Cleanup(branch string) error
	GetBaseBranch() string
	ListDirectories(basePath string) ([]string, error)
	ListFiles(basePath string) ([]string, error)
	ReadFile(path string) (string, error)
}

type RepoManager struct {
	Repo       *git.Repository
	RepoPath   string
	Token      string
	ForkToken  string
	BaseBranch string
	SignKey    *openpgp.Entity
}

func validateRepoURL(repoURL string) error {
	trimmed := strings.TrimSuffix(repoURL, ".git")
	if strings.HasPrefix(trimmed, "git@github.com:") {
		return nil
	}
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, ".") {
		return nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("invalid repository URL: %w", err)
	}
	if parsed.Scheme == "file" {
		return nil
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("unsupported scheme %q, only HTTPS is supported", parsed.Scheme)
	}
	if parsed.Host != "github.com" {
		return fmt.Errorf("unsupported host %q, only github.com is supported", parsed.Host)
	}
	return nil
}

func parseGPGKey(armoredKey string) (*openpgp.Entity, error) {
	if armoredKey == "" {
		return nil, nil
	}
	entities, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armoredKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse GPG signing key: %w", err)
	}
	if len(entities) == 0 {
		return nil, fmt.Errorf("GPG key contains no entities")
	}
	logrus.Info("GPG signing key loaded")
	return entities[0], nil
}

func InitRepo(repoURL, repoPath, token, baseBranch, forkRepoURL, forkToken, gpgKey string) (*RepoManager, error) {
	mgr := &RepoManager{
		RepoPath:   repoPath,
		Token:      token,
		ForkToken:  forkToken,
		BaseBranch: baseBranch,
	}

	signKey, err := parseGPGKey(gpgKey)
	if err != nil {
		return nil, err
	}
	mgr.SignKey = signKey

	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		logrus.WithField("path", repoPath).Info("Repo directory exists, opening")
		repo, err := git.PlainOpen(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open existing repo at %s: %w", repoPath, err)
		}
		mgr.Repo = repo

		if forkRepoURL != "" {
			if err := mgr.EnsureRemote("fork", forkRepoURL); err != nil {
				return nil, fmt.Errorf("failed to add fork remote: %w", err)
			}
		}

		return mgr, nil
	}

	if err := validateRepoURL(repoURL); err != nil {
		return nil, err
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

	if forkRepoURL != "" {
		if err := mgr.EnsureRemote("fork", forkRepoURL); err != nil {
			return nil, fmt.Errorf("failed to add fork remote: %w", err)
		}
	}

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

func (m *RepoManager) CheckoutExistingBranch(name string) error {
	remoteName := "origin"
	auth := m.auth()
	if m.ForkToken != "" {
		remoteName = "fork"
		auth = &http.BasicAuth{Username: "git", Password: m.ForkToken}
	}

	err := m.Repo.Fetch(&git.FetchOptions{
		RemoteName: remoteName,
		RefSpecs:   []config.RefSpec{config.RefSpec("+refs/heads/" + name + ":refs/remotes/" + remoteName + "/" + name)},
		Auth:       auth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch branch %s: %w", name, err)
	}

	remoteRef, err := m.Repo.Reference(plumbing.NewRemoteReferenceName(remoteName, name), true)
	if err != nil {
		return fmt.Errorf("remote branch %s not found: %w", name, err)
	}

	localRef := plumbing.NewBranchReferenceName(name)
	if err := m.Repo.Storer.SetReference(plumbing.NewHashReference(localRef, remoteRef.Hash())); err != nil {
		return fmt.Errorf("failed to set local ref for %s: %w", name, err)
	}

	w, err := m.Repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if err := w.Checkout(&git.CheckoutOptions{Branch: localRef}); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", name, err)
	}

	logrus.WithField("branch", name).Info("Checked out existing remote branch")
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

func (m *RepoManager) CommitChanges(message, authorName, authorEmail, dir string, files []File) (string, error) {
	w, err := m.Repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	dir = strings.TrimPrefix(dir, "/")
	for _, f := range files {
		relPath := filepath.Join(dir, f.Name)
		if _, err := w.Add(relPath); err != nil {
			return "", fmt.Errorf("failed to stage %s: %w", relPath, err)
		}
	}

	hash, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
		SignKey: m.SignKey,
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	logrus.WithField("sha", hash.String()[:8]).Info("Changes committed")
	return hash.String(), nil
}

func (m *RepoManager) PushBranch(branch string) error {
	return m.pushBranch(branch, false)
}

func (m *RepoManager) PushBranchForce(branch string) error {
	return m.pushBranch(branch, true)
}

func (m *RepoManager) pushBranch(branch string, force bool) error {
	ref := plumbing.NewBranchReferenceName(branch)

	remoteName := "origin"
	auth := m.auth()
	if m.ForkToken != "" {
		remoteName = "fork"
		auth = &http.BasicAuth{Username: "git", Password: m.ForkToken}
	}

	err := m.Repo.Push(&git.PushOptions{
		RemoteName: remoteName,
		RefSpecs:   []config.RefSpec{config.RefSpec(ref + ":" + ref)},
		Auth:       auth,
		Force:      force,
	})
	if err != nil {
		return fmt.Errorf("failed to push branch %s: %w", branch, err)
	}

	logrus.WithFields(logrus.Fields{
		"branch": branch,
		"remote": remoteName,
	}).Info("Branch pushed")
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

func (m *RepoManager) ListDirectories(basePath string) ([]string, error) {
	absPath := filepath.Clean(filepath.Join(m.RepoPath, basePath))
	repoRoot := filepath.Clean(m.RepoPath) + string(os.PathSeparator)
	if !strings.HasPrefix(absPath+string(os.PathSeparator), repoRoot) {
		return nil, fmt.Errorf("path escapes repository root: %s", basePath)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", basePath, err)
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}
	return dirs, nil
}

func (m *RepoManager) ListFiles(basePath string) ([]string, error) {
	absPath := filepath.Clean(filepath.Join(m.RepoPath, basePath))
	repoRoot := filepath.Clean(m.RepoPath) + string(os.PathSeparator)
	if !strings.HasPrefix(absPath+string(os.PathSeparator), repoRoot) {
		return nil, fmt.Errorf("path escapes repository root: %s", basePath)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", basePath, err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

func (m *RepoManager) ReadFile(path string) (string, error) {
	absPath := filepath.Clean(filepath.Join(m.RepoPath, path))
	repoRoot := filepath.Clean(m.RepoPath) + string(os.PathSeparator)
	if !strings.HasPrefix(absPath, repoRoot) {
		return "", fmt.Errorf("path escapes repository root: %s", path)
	}

	info, err := os.Lstat(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symlinks are not allowed: %s", path)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return string(data), nil
}

func (m *RepoManager) auth() *http.BasicAuth {
	return &http.BasicAuth{
		Username: "git",
		Password: m.Token,
	}
}
