package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createBareRepo(t *testing.T) string {
	t.Helper()

	// Create a non-bare repo with an initial commit, then clone it as bare
	work := t.TempDir()
	repo, err := gogit.PlainInit(work, false)
	require.NoError(t, err)

	w, err := repo.Worktree()
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(work, "README.md"), []byte("init"), 0644)
	require.NoError(t, err)

	_, err = w.Add("README.md")
	require.NoError(t, err)

	_, err = w.Commit("initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	bare := t.TempDir()
	_, err = gogit.PlainClone(bare, true, &gogit.CloneOptions{URL: work})
	require.NoError(t, err)

	return bare
}

func TestInitRepo_Clone(t *testing.T) {
	bare := createBareRepo(t)
	cloneDest := filepath.Join(t.TempDir(), "repo")

	mgr, err := InitRepo(bare, cloneDest, "", "master")
	require.NoError(t, err)
	assert.NotNil(t, mgr.Repo)
	assert.Equal(t, cloneDest, mgr.RepoPath)

	_, err = os.Stat(filepath.Join(cloneDest, "README.md"))
	assert.NoError(t, err)
}

func TestInitRepo_OpensExisting(t *testing.T) {
	bare := createBareRepo(t)
	cloneDest := filepath.Join(t.TempDir(), "repo")

	mgr1, err := InitRepo(bare, cloneDest, "", "master")
	require.NoError(t, err)

	mgr2, err := InitRepo(bare, cloneDest, "", "master")
	require.NoError(t, err)

	head1, _ := mgr1.Repo.Head()
	head2, _ := mgr2.Repo.Head()
	assert.Equal(t, head1.Hash(), head2.Hash())
}

func TestInitRepo_BadURL(t *testing.T) {
	cloneDest := filepath.Join(t.TempDir(), "repo")
	_, err := InitRepo("/nonexistent/repo", cloneDest, "", "master")
	assert.Error(t, err)
}

func TestPullLatest(t *testing.T) {
	bare := createBareRepo(t)
	cloneDest := filepath.Join(t.TempDir(), "repo")

	mgr, err := InitRepo(bare, cloneDest, "", "master")
	require.NoError(t, err)

	// Push a new commit to the bare repo via a separate clone
	work := t.TempDir()
	repo, err := gogit.PlainClone(work, false, &gogit.CloneOptions{URL: bare})
	require.NoError(t, err)

	w, err := repo.Worktree()
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(work, "new.txt"), []byte("new content"), 0644)
	require.NoError(t, err)

	_, err = w.Add("new.txt")
	require.NoError(t, err)

	_, err = w.Commit("add new file", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	err = repo.Push(&gogit.PushOptions{})
	require.NoError(t, err)

	err = mgr.PullLatest()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(cloneDest, "new.txt"))
	assert.NoError(t, err)
}

func TestPullLatest_AlreadyUpToDate(t *testing.T) {
	bare := createBareRepo(t)
	cloneDest := filepath.Join(t.TempDir(), "repo")

	mgr, err := InitRepo(bare, cloneDest, "", "master")
	require.NoError(t, err)

	err = mgr.PullLatest()
	assert.NoError(t, err)
}
