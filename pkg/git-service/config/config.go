package config

import (
	"os"
	"strconv"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
)

type GitServiceConfig struct {
	GitHubToken        string
	RepoURL            string
	BaseBranch         string
	RepoPath           string
	Port               string
	MetricsPort        int
	ReviewersTeam      string
	QuickstartsDirPath string
	PSKToken           string
}

var cfg *GitServiceConfig

func Init() {
	cfg = &GitServiceConfig{
		GitHubToken:        os.Getenv("GITHUB_TOKEN"),
		RepoURL:            os.Getenv("GITHUB_REPO_URL"),
		BaseBranch:         getEnvOrDefault("GITHUB_BASE_BRANCH", "main"),
		RepoPath:           getEnvOrDefault("QUICKSTARTS_REPO_PATH", "/var/quickstarts-repo"),
		Port:               getEnvOrDefault("PORT", "8001"),
		MetricsPort:        8080,
		ReviewersTeam:      os.Getenv("PR_REVIEWERS_TEAM"),
		QuickstartsDirPath: getEnvOrDefault("QUICKSTARTS_DIR_PATH", "/docs/quickstarts/"),
		PSKToken:           os.Getenv("PSK_TOKEN"),
	}

	if clowder.IsClowderEnabled() {
		lcfg := clowder.LoadedConfig
		if lcfg.PrivatePort != nil {
			cfg.Port = strconv.Itoa(*lcfg.PrivatePort)
		}
		cfg.MetricsPort = lcfg.MetricsPort
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
