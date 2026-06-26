package config

import "os"

type GitServiceConfig struct {
	GitHubToken       string
	RepoURL           string
	BaseBranch        string
	RepoPath          string
	Port              string
	ReviewersTeam     string
	QuickstartsDirPath string
}

var cfg *GitServiceConfig

func Init() {
	cfg = &GitServiceConfig{
		GitHubToken: os.Getenv("GITHUB_TOKEN"),
		RepoURL:     os.Getenv("GITHUB_REPO_URL"),
		BaseBranch:  getEnvOrDefault("GITHUB_BASE_BRANCH", "main"),
		RepoPath:    getEnvOrDefault("QUICKSTARTS_REPO_PATH", "/var/quickstarts-repo"),
		Port:               getEnvOrDefault("PORT", "8000"),
		ReviewersTeam:      os.Getenv("PR_REVIEWERS_TEAM"),
		QuickstartsDirPath: getEnvOrDefault("QUICKSTARTS_DIR_PATH", "/docs/quickstarts/"),
	}
}

func Get() *GitServiceConfig {
	return cfg
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
