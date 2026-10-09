package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	gitops "github.com/RedHatInsights/quickstarts/pkg/git-service/git"
	ghclient "github.com/RedHatInsights/quickstarts/pkg/git-service/github"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests run SubmitBatchPR against a real RepoManager whose origin is a
// local bare repository, so the fetch, branch, commit, push and rollback steps
// all go through go-git. Only GitHub itself is mocked.

const integrationBranch = "qs-update-batch-integration"

var seedFiles = map[string]string{
	"docs/quickstarts/alpha/metadata.yml": "kind: QuickStarts\nname: alpha\n",
	"docs/quickstarts/alpha/alpha.yml":    "# keep this comment\nspec:\n  displayName: Alpha\n",
	"docs/quickstarts/beta/metadata.yml":  "kind: QuickStarts\nname: beta\n",
	"docs/quickstarts/beta/beta.yml":      "spec:\n  displayName: Beta\n",
	"docs/quickstarts/gamma/gamma.yml":    "spec:\n  displayName: Gamma\n",
}

func commitAll(t *testing.T, repo *gogit.Repository, msg string) plumbing.Hash {
	t.Helper()
	w, err := repo.Worktree()
	require.NoError(t, err)
	require.NoError(t, w.AddGlob("."))
	hash, err := w.Commit(msg, &gogit.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
	})
	require.NoError(t, err)
	return hash
}

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for p, content := range files {
		abs := filepath.Join(root, p)
		require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0755))
		require.NoError(t, os.WriteFile(abs, []byte(content), 0644))
	}
}

// seedOrigin creates a bare repository whose master branch holds seedFiles and
// returns its path with the master commit.
func seedOrigin(t *testing.T) (string, string) {
	t.Helper()
	work := t.TempDir()
	repo, err := gogit.PlainInit(work, false)
	require.NoError(t, err)
	writeTree(t, work, seedFiles)
	commitAll(t, repo, "seed quickstarts")

	bare := t.TempDir()
	origin, err := gogit.PlainClone(bare, true, &gogit.CloneOptions{URL: work})
	require.NoError(t, err)
	head, err := origin.Head()
	require.NoError(t, err)
	return bare, head.Hash().String()
}

// pushToOrigin commits files on top of branch in a separate clone and pushes,
// standing in for another save or a merge landing on the remote.
func pushToOrigin(t *testing.T, bare, branch string, files map[string]string) string {
	t.Helper()
	work := t.TempDir()
	repo, err := gogit.PlainClone(work, false, &gogit.CloneOptions{
		URL:           bare,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
	})
	require.NoError(t, err)
	writeTree(t, work, files)
	hash := commitAll(t, repo, "concurrent change")
	require.NoError(t, repo.Push(&gogit.PushOptions{}))
	return hash.String()
}

func originBranchHead(t *testing.T, bare, branch string) (string, bool) {
	t.Helper()
	repo, err := gogit.PlainOpen(bare)
	require.NoError(t, err)
	ref, err := repo.Reference(plumbing.NewBranchReferenceName(branch), true)
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return "", false
	}
	require.NoError(t, err)
	return ref.Hash().String(), true
}

// commitOnOrigin returns a commit's single parent and the paths it changed.
func commitOnOrigin(t *testing.T, bare, sha string) (string, []string) {
	t.Helper()
	repo, err := gogit.PlainOpen(bare)
	require.NoError(t, err)
	commit, err := repo.CommitObject(plumbing.NewHash(sha))
	require.NoError(t, err)
	require.Equal(t, 1, commit.NumParents())
	parent, err := commit.Parent(0)
	require.NoError(t, err)

	from, err := parent.Tree()
	require.NoError(t, err)
	to, err := commit.Tree()
	require.NoError(t, err)
	changes, err := from.Diff(to)
	require.NoError(t, err)

	// Rename detection pairs a deletion with an addition; list both paths.
	var paths []string
	for _, c := range changes {
		if c.From.Name != "" {
			paths = append(paths, c.From.Name)
		}
		if c.To.Name != "" && c.To.Name != c.From.Name {
			paths = append(paths, c.To.Name)
		}
	}
	sort.Strings(paths)
	return parent.Hash.String(), paths
}

func fileOnOrigin(t *testing.T, bare, sha, path string) string {
	t.Helper()
	repo, err := gogit.PlainOpen(bare)
	require.NoError(t, err)
	commit, err := repo.CommitObject(plumbing.NewHash(sha))
	require.NoError(t, err)
	f, err := commit.File(path)
	require.NoError(t, err)
	content, err := f.Contents()
	require.NoError(t, err)
	return content
}

// assertRolledBack checks the clone is back on a clean base branch with no
// leftover batch branch, which is what the next request relies on.
func assertRolledBack(t *testing.T, mgr *gitops.RepoManager) {
	t.Helper()
	head, err := mgr.Repo.Head()
	require.NoError(t, err)
	assert.Equal(t, plumbing.NewBranchReferenceName("master"), head.Name())

	w, err := mgr.Repo.Worktree()
	require.NoError(t, err)
	status, err := w.Status()
	require.NoError(t, err)
	assert.True(t, status.IsClean(), "worktree has leftovers:\n%s", status)

	_, err = mgr.Repo.Reference(plumbing.NewBranchReferenceName(integrationBranch), false)
	assert.ErrorIs(t, err, plumbing.ErrReferenceNotFound, "the local batch branch is deleted")
}

func newIntegrationHandler(t *testing.T) (*Handler, *gitops.RepoManager, *mockGitHubClient, string, string) {
	t.Helper()
	bare, base := seedOrigin(t)
	mgr, err := gitops.InitRepo(bare, filepath.Join(t.TempDir(), "clone"), "", "master", "", "", "")
	require.NoError(t, err)
	gh := &mockGitHubClient{createPRURL: "https://github.com/o/r/pull/7", createPRNumber: 7}
	return NewHandler(mgr, gh, "", "/docs/quickstarts/"), mgr, gh, bare, base
}

func batchBody(t *testing.T, meta BatchPRMetadata, resources ...BatchResource) string {
	t.Helper()
	b, err := json.Marshal(BatchPRRequest{Metadata: meta, Resources: resources})
	require.NoError(t, err)
	return string(b)
}

func createMeta(base string) BatchPRMetadata {
	return BatchPRMetadata{
		BranchName:    integrationBranch,
		BaseSHA:       base,
		CommitMessage: "docs: update alpha and beta",
		PRTitle:       "Update alpha and beta",
	}
}

var (
	alphaEdit = BatchResource{Name: "alpha", Files: []File{{Name: "alpha.yml", Content: "# keep this comment\nspec:\n  displayName: Alpha v2\n"}}}
	betaEdit  = BatchResource{Name: "beta", Files: []File{{Name: "beta.yml", Content: "spec:\n  displayName: Beta v2\n"}}}
)

func decodeBatch(t *testing.T, body []byte) BatchPRResponse {
	t.Helper()
	var resp BatchPRResponse
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func TestBatchIntegration_CreateThenUpdate(t *testing.T) {
	h, mgr, gh, bare, base := newIntegrationHandler(t)

	// First save: one commit on a new branch, across two directories.
	rr := postBatch(h, batchBody(t, createMeta(base), alphaEdit, betaEdit))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	created := decodeBatch(t, rr.Body.Bytes())
	assert.Equal(t, "created", created.Status)
	assert.Equal(t, 7, created.PRNumber)
	assert.Equal(t, integrationBranch, gh.createdHead)
	assert.Equal(t, "master", gh.createdBase)

	first, ok := originBranchHead(t, bare, integrationBranch)
	require.True(t, ok, "the branch was pushed")
	assert.Equal(t, created.HeadSHA, first, "the response reports the pushed head")

	parent, changed := commitOnOrigin(t, bare, first)
	assert.Equal(t, base, parent, "the branch starts from the loaded base")
	assert.Equal(t, []string{"docs/quickstarts/alpha/alpha.yml", "docs/quickstarts/beta/beta.yml"}, changed,
		"one commit, only the edited files, nothing pruned")
	assert.Equal(t, alphaEdit.Files[0].Content, fileOnOrigin(t, bare, first, "docs/quickstarts/alpha/alpha.yml"),
		"content is written byte for byte, comments included")
	assertRolledBack(t, mgr)

	// Second save: appended to the same branch, history kept.
	gh.getPR = &ghclient.CreatorPR{Number: 7, HTMLURL: created.PRURL, BranchName: integrationBranch, HeadSHA: first}
	gh.createdTitle = ""
	update := BatchPRMetadata{PRNumber: 7, ExpectedHeadSHA: first, CommitMessage: "docs: second pass"}
	gammaEdit := BatchResource{Name: "gamma", Files: []File{{Name: "gamma.yml", Content: "spec:\n  displayName: Gamma v2\n"}}}

	rr = postBatch(h, batchBody(t, update, gammaEdit))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	updated := decodeBatch(t, rr.Body.Bytes())
	assert.Equal(t, "updated", updated.Status)
	assert.Equal(t, 7, updated.PRNumber)
	assert.Empty(t, gh.createdTitle, "no second PR is opened")

	second, _ := originBranchHead(t, bare, integrationBranch)
	assert.Equal(t, updated.HeadSHA, second)
	parent, changed = commitOnOrigin(t, bare, second)
	assert.Equal(t, first, parent, "the first commit is kept, not rewritten")
	assert.Equal(t, []string{"docs/quickstarts/gamma/gamma.yml"}, changed)
	assert.Equal(t, alphaEdit.Files[0].Content, fileOnOrigin(t, bare, second, "docs/quickstarts/alpha/alpha.yml"),
		"resources not sent in the second save keep their first-save content")
	assertRolledBack(t, mgr)
}

func TestBatchIntegration_StaleHeadLeavesRemoteUnchanged(t *testing.T) {
	h, mgr, gh, bare, base := newIntegrationHandler(t)

	rr := postBatch(h, batchBody(t, createMeta(base), alphaEdit))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	first := decodeBatch(t, rr.Body.Bytes()).HeadSHA

	// Another tab saves to the PR. GitHub's PR head still reports the old
	// commit (it lags the push), so only the head of the fetched branch can
	// catch the conflict.
	moved := pushToOrigin(t, bare, integrationBranch, map[string]string{"docs/quickstarts/beta/beta.yml": "spec: {}\n"})
	gh.getPR = &ghclient.CreatorPR{Number: 7, BranchName: integrationBranch, HeadSHA: first}

	update := BatchPRMetadata{PRNumber: 7, ExpectedHeadSHA: first, CommitMessage: "docs: stale save"}
	rr = postBatch(h, batchBody(t, update, betaEdit))
	assert.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())

	head, _ := originBranchHead(t, bare, integrationBranch)
	assert.Equal(t, moved, head, "the other tab's commit is not overwritten")
	assertRolledBack(t, mgr)
}

// racingRepo pushes a competing commit after the head check has passed but
// before the batch is committed, the window a non-force push has to guard.
type racingRepo struct {
	*gitops.RepoManager
	race func()
}

func (r *racingRepo) CommitPaths(message, authorName, authorEmail string, paths []string) (string, error) {
	r.race()
	return r.RepoManager.CommitPaths(message, authorName, authorEmail, paths)
}

func TestBatchIntegration_PushRaceIsConflict(t *testing.T) {
	_, mgr, gh, bare, base := newIntegrationHandler(t)
	h := NewHandler(mgr, gh, "", "/docs/quickstarts/")

	rr := postBatch(h, batchBody(t, createMeta(base), alphaEdit))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	first := decodeBatch(t, rr.Body.Bytes()).HeadSHA

	var moved string
	racing := &racingRepo{RepoManager: mgr, race: func() {
		moved = pushToOrigin(t, bare, integrationBranch, map[string]string{"docs/quickstarts/gamma/gamma.yml": "spec: {}\n"})
	}}
	h = NewHandler(racing, gh, "", "/docs/quickstarts/")
	gh.getPR = &ghclient.CreatorPR{Number: 7, BranchName: integrationBranch, HeadSHA: first}

	update := BatchPRMetadata{PRNumber: 7, ExpectedHeadSHA: first, CommitMessage: "docs: racing save"}
	rr = postBatch(h, batchBody(t, update, betaEdit))
	assert.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())

	head, _ := originBranchHead(t, bare, integrationBranch)
	assert.Equal(t, moved, head, "the push is rejected rather than forced")
	assertRolledBack(t, mgr)
}

func TestBatchIntegration_InvalidFileRollsBackEarlierWrites(t *testing.T) {
	h, mgr, _, bare, base := newIntegrationHandler(t)

	// alpha is written to the worktree before beta's unknown file is found.
	missing := BatchResource{Name: "beta", Files: []File{{Name: "nope.yml", Content: "a: 1\n"}}}
	rr := postBatch(h, batchBody(t, createMeta(base), alphaEdit, missing))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "docs/quickstarts/beta/nope.yml does not exist")

	_, pushed := originBranchHead(t, bare, integrationBranch)
	assert.False(t, pushed, "nothing reaches the remote")
	assertRolledBack(t, mgr)

	content, err := os.ReadFile(filepath.Join(mgr.RepoPath, "docs/quickstarts/alpha/alpha.yml"))
	require.NoError(t, err)
	assert.Equal(t, seedFiles["docs/quickstarts/alpha/alpha.yml"], string(content), "the partial write is reverted")
}

func TestBatchIntegration_BaseMovedKeepsLoadedSnapshot(t *testing.T) {
	h, mgr, _, bare, base := newIntegrationHandler(t)

	// Someone merges a change to beta after the browser loaded the base.
	newMaster := pushToOrigin(t, bare, "master", map[string]string{"docs/quickstarts/beta/beta.yml": "spec:\n  displayName: Beta merged\n"})

	rr := postBatch(h, batchBody(t, createMeta(base), alphaEdit))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	head := decodeBatch(t, rr.Body.Bytes()).HeadSHA

	parent, changed := commitOnOrigin(t, bare, head)
	assert.Equal(t, base, parent)
	assert.NotEqual(t, newMaster, parent)
	assert.Equal(t, []string{"docs/quickstarts/alpha/alpha.yml"}, changed,
		"the merged beta change is not reverted by the batch commit")
	assertRolledBack(t, mgr)
}

func TestBatchIntegration_UnknownBaseIsConflict(t *testing.T) {
	h, mgr, _, bare, _ := newIntegrationHandler(t)

	rr := postBatch(h, batchBody(t, createMeta("0123456789abcdef0123456789abcdef01234567"), alphaEdit))
	assert.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())

	_, pushed := originBranchHead(t, bare, integrationBranch)
	assert.False(t, pushed)
	assertRolledBack(t, mgr)
}

func TestBatchIntegration_NoChangesIsRejected(t *testing.T) {
	h, mgr, _, bare, base := newIntegrationHandler(t)

	same := BatchResource{Name: "alpha", Files: []File{{Name: "alpha.yml", Content: seedFiles["docs/quickstarts/alpha/alpha.yml"]}}}
	rr := postBatch(h, batchBody(t, createMeta(base), same))
	assert.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "nothing to save")

	_, pushed := originBranchHead(t, bare, integrationBranch)
	assert.False(t, pushed)
	assertRolledBack(t, mgr)
}

func TestBatchIntegration_RetryAfterPRCreationFailure(t *testing.T) {
	h, mgr, gh, bare, base := newIntegrationHandler(t)

	gh.createPRErr = errors.New("github is down")
	rr := postBatch(h, batchBody(t, createMeta(base), alphaEdit))
	require.Equal(t, http.StatusBadGateway, rr.Code, rr.Body.String())
	orphan, pushed := originBranchHead(t, bare, integrationBranch)
	require.True(t, pushed, "the branch was pushed before PR creation failed")
	assertRolledBack(t, mgr)

	// The retry with the same branch name replaces the orphaned branch.
	gh.createPRErr = nil
	rr = postBatch(h, batchBody(t, createMeta(base), alphaEdit, betaEdit))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	resp := decodeBatch(t, rr.Body.Bytes())
	assert.Equal(t, "created", resp.Status)

	head, _ := originBranchHead(t, bare, integrationBranch)
	assert.Equal(t, resp.HeadSHA, head)
	assert.NotEqual(t, orphan, head)
	parent, changed := commitOnOrigin(t, bare, head)
	assert.Equal(t, base, parent, "the retry is one commit on the base, not stacked on the orphan")
	assert.Equal(t, []string{"docs/quickstarts/alpha/alpha.yml", "docs/quickstarts/beta/beta.yml"}, changed)
	assertRolledBack(t, mgr)
}

var deltaNew = BatchResource{Name: "delta", Create: true, Files: []File{
	{Name: "metadata.yml", Content: "kind: QuickStarts\nname: delta\ntags: []\n"},
	{Name: "delta.yml", Content: "metadata:\n  name: delta\nspec:\n  displayName: Delta\n"},
}}

func TestBatchIntegration_CreateAlongsideEdits(t *testing.T) {
	h, mgr, gh, bare, base := newIntegrationHandler(t)

	rr := postBatch(h, batchBody(t, createMeta(base), alphaEdit, deltaNew))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	first := decodeBatch(t, rr.Body.Bytes()).HeadSHA
	assert.Contains(t, gh.createdBody, "docs/quickstarts/delta/ (new)")

	_, changed := commitOnOrigin(t, bare, first)
	assert.Equal(t, []string{
		"docs/quickstarts/alpha/alpha.yml",
		"docs/quickstarts/delta/delta.yml",
		"docs/quickstarts/delta/metadata.yml",
	}, changed, "the edit and the new quickstart land in one commit")
	assertRolledBack(t, mgr)

	// Once saved, the new quickstart is edited like any other.
	gh.getPR = &ghclient.CreatorPR{Number: 7, BranchName: integrationBranch, HeadSHA: first}
	update := BatchPRMetadata{PRNumber: 7, ExpectedHeadSHA: first, CommitMessage: "docs: retitle delta"}
	deltaEdit := BatchResource{Name: "delta", Files: []File{{Name: "delta.yml", Content: "metadata:\n  name: delta\nspec:\n  displayName: Delta v2\n"}}}
	rr = postBatch(h, batchBody(t, update, deltaEdit))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	// Creating it a second time is a conflict, not an overwrite.
	second := decodeBatch(t, rr.Body.Bytes()).HeadSHA
	gh.getPR = &ghclient.CreatorPR{Number: 7, BranchName: integrationBranch, HeadSHA: second}
	update = BatchPRMetadata{PRNumber: 7, ExpectedHeadSHA: second, CommitMessage: "docs: again"}
	rr = postBatch(h, batchBody(t, update, deltaNew))
	assert.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "already exists")
	head, _ := originBranchHead(t, bare, integrationBranch)
	assert.Equal(t, second, head)
	assertRolledBack(t, mgr)
}

func TestBatchIntegration_CreateExistingDirectoryRollsBack(t *testing.T) {
	h, mgr, _, bare, base := newIntegrationHandler(t)

	clash := BatchResource{Name: "beta", Create: true, Files: []File{
		{Name: "metadata.yml", Content: "name: beta\n"},
		{Name: "beta.yml", Content: "metadata:\n  name: beta\n"},
	}}
	rr := postBatch(h, batchBody(t, createMeta(base), alphaEdit, clash))
	assert.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())

	_, pushed := originBranchHead(t, bare, integrationBranch)
	assert.False(t, pushed, "nothing reaches the remote")
	assertRolledBack(t, mgr)
	content, err := os.ReadFile(filepath.Join(mgr.RepoPath, "docs/quickstarts/beta/beta.yml"))
	require.NoError(t, err)
	assert.Equal(t, seedFiles["docs/quickstarts/beta/beta.yml"], string(content), "the existing quickstart is untouched")
}

func TestBatchIntegration_RevertTakesResourcesOutOfThePR(t *testing.T) {
	h, mgr, gh, bare, base := newIntegrationHandler(t)

	rr := postBatch(h, batchBody(t, createMeta(base), alphaEdit, betaEdit, deltaNew))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	first := decodeBatch(t, rr.Body.Bytes()).HeadSHA

	// Main moves on after the branch was cut. Reverting must restore alpha to
	// where the branch started, not pull main's newer content into the PR.
	pushToOrigin(t, bare, "master", map[string]string{"docs/quickstarts/alpha/alpha.yml": "spec:\n  displayName: Alpha on main\n"})

	gh.getPR = &ghclient.CreatorPR{Number: 7, BranchName: integrationBranch, HeadSHA: first}
	update := BatchPRMetadata{PRNumber: 7, ExpectedHeadSHA: first, CommitMessage: "docs: drop alpha and delta"}
	rr = postBatch(h, batchBody(t, update,
		BatchResource{Name: "alpha", Revert: true},
		BatchResource{Name: "delta", Revert: true},
	))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	second := decodeBatch(t, rr.Body.Bytes()).HeadSHA

	parent, changed := commitOnOrigin(t, bare, second)
	assert.Equal(t, first, parent)
	assert.Equal(t, []string{
		"docs/quickstarts/alpha/alpha.yml",
		"docs/quickstarts/delta/delta.yml",
		"docs/quickstarts/delta/metadata.yml",
	}, changed)
	assert.Equal(t, seedFiles["docs/quickstarts/alpha/alpha.yml"], fileOnOrigin(t, bare, second, "docs/quickstarts/alpha/alpha.yml"),
		"alpha is back to its merge-base content")
	assert.Equal(t, betaEdit.Files[0].Content, fileOnOrigin(t, bare, second, "docs/quickstarts/beta/beta.yml"),
		"resources not reverted keep their PR content")

	repo, err := gogit.PlainOpen(bare)
	require.NoError(t, err)
	commit, err := repo.CommitObject(plumbing.NewHash(second))
	require.NoError(t, err)
	_, err = commit.File("docs/quickstarts/delta/metadata.yml")
	assert.ErrorIs(t, err, object.ErrFileNotFound, "a quickstart the PR created is deleted")
	assertRolledBack(t, mgr)
}

func TestBatchIntegration_RevertUnchangedResourceIsNothingToSave(t *testing.T) {
	h, mgr, gh, bare, base := newIntegrationHandler(t)

	rr := postBatch(h, batchBody(t, createMeta(base), alphaEdit))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	first := decodeBatch(t, rr.Body.Bytes()).HeadSHA

	gh.getPR = &ghclient.CreatorPR{Number: 7, BranchName: integrationBranch, HeadSHA: first}
	update := BatchPRMetadata{PRNumber: 7, ExpectedHeadSHA: first, CommitMessage: "docs: drop beta"}
	rr = postBatch(h, batchBody(t, update, BatchResource{Name: "beta", Revert: true}))
	assert.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())

	head, _ := originBranchHead(t, bare, integrationBranch)
	assert.Equal(t, first, head)
	assertRolledBack(t, mgr)
}

func TestBatchIntegration_RenameMovesTheDirectory(t *testing.T) {
	h, mgr, gh, bare, _ := newIntegrationHandler(t)
	// Anything that is not YAML (an image, a README) moves with the quickstart.
	extra := pushToOrigin(t, bare, "master", map[string]string{"docs/quickstarts/alpha/diagram.svg": "<svg/>"})

	omega := BatchResource{Name: "omega", RenameFrom: "alpha", Files: []File{
		{Name: "metadata.yml", Content: "kind: QuickStarts\nname: omega\n"},
		{Name: "omega.yml", Content: "# keep this comment\nmetadata:\n  name: omega\nspec:\n  displayName: Alpha\n"},
	}}
	rr := postBatch(h, batchBody(t, createMeta(extra), omega, betaEdit))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	first := decodeBatch(t, rr.Body.Bytes()).HeadSHA
	assert.Contains(t, gh.createdBody, "docs/quickstarts/omega/ (renamed from alpha)")

	_, changed := commitOnOrigin(t, bare, first)
	assert.Equal(t, []string{
		"docs/quickstarts/alpha/alpha.yml",
		"docs/quickstarts/alpha/diagram.svg",
		"docs/quickstarts/alpha/metadata.yml",
		"docs/quickstarts/beta/beta.yml",
		"docs/quickstarts/omega/diagram.svg",
		"docs/quickstarts/omega/metadata.yml",
		"docs/quickstarts/omega/omega.yml",
	}, changed, "the old directory is deleted and the new one written in the same commit")
	assert.Equal(t, "<svg/>", fileOnOrigin(t, bare, first, "docs/quickstarts/omega/diagram.svg"))
	assert.Equal(t, omega.Files[1].Content, fileOnOrigin(t, bare, first, "docs/quickstarts/omega/omega.yml"))

	repo, err := gogit.PlainOpen(bare)
	require.NoError(t, err)
	commit, err := repo.CommitObject(plumbing.NewHash(first))
	require.NoError(t, err)
	_, err = commit.File("docs/quickstarts/alpha/metadata.yml")
	assert.ErrorIs(t, err, object.ErrFileNotFound, "nothing is left behind in the old directory")
	assertRolledBack(t, mgr)

	// Renaming back and reverting both directories undo it entirely.
	gh.getPR = &ghclient.CreatorPR{Number: 7, BranchName: integrationBranch, HeadSHA: first}
	update := BatchPRMetadata{PRNumber: 7, ExpectedHeadSHA: first, CommitMessage: "docs: drop the rename"}
	rr = postBatch(h, batchBody(t, update,
		BatchResource{Name: "omega", Revert: true},
		BatchResource{Name: "alpha", Revert: true},
	))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	second := decodeBatch(t, rr.Body.Bytes()).HeadSHA
	assert.Equal(t, seedFiles["docs/quickstarts/alpha/alpha.yml"], fileOnOrigin(t, bare, second, "docs/quickstarts/alpha/alpha.yml"))
	assert.Equal(t, "<svg/>", fileOnOrigin(t, bare, second, "docs/quickstarts/alpha/diagram.svg"))
	commit, err = repo.CommitObject(plumbing.NewHash(second))
	require.NoError(t, err)
	_, err = commit.File("docs/quickstarts/omega/omega.yml")
	assert.ErrorIs(t, err, object.ErrFileNotFound)
	assertRolledBack(t, mgr)
}

func TestBatchIntegration_RenameOntoExistingDirectoryRollsBack(t *testing.T) {
	h, mgr, _, bare, base := newIntegrationHandler(t)

	clash := BatchResource{Name: "beta", RenameFrom: "alpha", Files: []File{
		{Name: "metadata.yml", Content: "name: beta\n"},
		{Name: "beta.yml", Content: "metadata:\n  name: beta\n"},
	}}
	rr := postBatch(h, batchBody(t, createMeta(base), clash))
	assert.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "already exists")

	_, pushed := originBranchHead(t, bare, integrationBranch)
	assert.False(t, pushed, "nothing reaches the remote")
	assertRolledBack(t, mgr)
	content, err := os.ReadFile(filepath.Join(mgr.RepoPath, "docs/quickstarts/alpha/alpha.yml"))
	require.NoError(t, err)
	assert.Equal(t, seedFiles["docs/quickstarts/alpha/alpha.yml"], string(content), "the renamed quickstart is untouched")
}

func TestBatchIntegration_RevertRenamedNewResource(t *testing.T) {
	h, mgr, gh, bare, base := newIntegrationHandler(t)

	rr := postBatch(h, batchBody(t, createMeta(base), deltaNew))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	first := decodeBatch(t, rr.Body.Bytes()).HeadSHA

	gh.getPR = &ghclient.CreatorPR{Number: 7, BranchName: integrationBranch, HeadSHA: first}
	epsilon := BatchResource{Name: "epsilon", RenameFrom: "delta", Files: []File{
		{Name: "metadata.yml", Content: "kind: QuickStarts\nname: epsilon\ntags: []\n"},
		{Name: "epsilon.yml", Content: "metadata:\n  name: epsilon\nspec:\n  displayName: Delta\n"},
	}}
	rr = postBatch(h, batchBody(t, BatchPRMetadata{PRNumber: 7, ExpectedHeadSHA: first, CommitMessage: "docs: rename delta"}, epsilon))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	second := decodeBatch(t, rr.Body.Bytes()).HeadSHA

	// delta was added by this PR and is already gone from the branch, so its
	// revert has nothing to do; epsilon is deleted.
	gh.getPR = &ghclient.CreatorPR{Number: 7, BranchName: integrationBranch, HeadSHA: second}
	rr = postBatch(h, batchBody(t, BatchPRMetadata{PRNumber: 7, ExpectedHeadSHA: second, CommitMessage: "docs: drop delta"},
		BatchResource{Name: "epsilon", Revert: true},
		BatchResource{Name: "delta", Revert: true},
	))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	_, changed := commitOnOrigin(t, bare, decodeBatch(t, rr.Body.Bytes()).HeadSHA)
	assert.Equal(t, []string{"docs/quickstarts/epsilon/epsilon.yml", "docs/quickstarts/epsilon/metadata.yml"}, changed)
	assertRolledBack(t, mgr)
}
