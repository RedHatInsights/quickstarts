package routes

import (
	"os"

	"github.com/RedHatInsights/quickstarts/config"
	"github.com/RedHatInsights/quickstarts/pkg/clients"
	"github.com/RedHatInsights/quickstarts/pkg/services"
)

// Pagination holds pagination parameters
type Pagination struct {
	Limit  int
	Offset int
}

// ServerAdapter implements the generated.ServerInterface with business logic services
type ServerAdapter struct {
	quickstartService *services.QuickstartService
	helpTopicService  *services.HelpTopicService
	favoriteService   *services.FavoriteService
	progressService   *services.ProgressService
	gitServiceClient  *clients.GitService
	gitServiceEnabled bool
}

// NewServerAdapter creates a new server adapter with service dependencies
func NewServerAdapter() *ServerAdapter {
	cfg := config.Get()
	gitEnabled := os.Getenv("GIT_SERVICE_ENABLED") == "true"
	var gitClient *clients.GitService
	if gitEnabled {
		gitClient = clients.NewGitService(cfg.GitServiceURL, cfg.PSKToken)
	}
	return &ServerAdapter{
		quickstartService: services.NewQuickstartService(),
		helpTopicService:  services.NewHelpTopicService(),
		favoriteService:   services.NewFavoriteService(),
		progressService:   services.NewProgressService(),
		gitServiceClient:  gitClient,
		gitServiceEnabled: gitEnabled,
	}
}