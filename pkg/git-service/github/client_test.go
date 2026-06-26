package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRepoURL_HTTPS(t *testing.T) {
	owner, repo, err := ParseRepoURL("https://github.com/RedHatInsights/quickstarts")
	require.NoError(t, err)
	assert.Equal(t, "RedHatInsights", owner)
	assert.Equal(t, "quickstarts", repo)
}

func TestParseRepoURL_WithGitSuffix(t *testing.T) {
	owner, repo, err := ParseRepoURL("https://github.com/RedHatInsights/quickstarts.git")
	require.NoError(t, err)
	assert.Equal(t, "RedHatInsights", owner)
	assert.Equal(t, "quickstarts", repo)
}

func TestParseRepoURL_SSH(t *testing.T) {
	owner, repo, err := ParseRepoURL("git@github.com:hossam-farid/quickstarts-test.git")
	require.NoError(t, err)
	assert.Equal(t, "hossam-farid", owner)
	assert.Equal(t, "quickstarts-test", repo)
}

func TestParseRepoURL_Invalid(t *testing.T) {
	_, _, err := ParseRepoURL("https://gitlab.com/some/repo")
	assert.Error(t, err)
}

func TestParseRepoURL_MissingRepo(t *testing.T) {
	_, _, err := ParseRepoURL("https://github.com/onlyowner")
	assert.Error(t, err)
}
