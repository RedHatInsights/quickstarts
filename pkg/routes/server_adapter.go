package routes

import (
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
}

// NewServerAdapter creates a new server adapter with service dependencies
func NewServerAdapter() *ServerAdapter {
	return &ServerAdapter{
		quickstartService: services.NewQuickstartService(),
		helpTopicService:  services.NewHelpTopicService(),
		favoriteService:   services.NewFavoriteService(),
		progressService:   services.NewProgressService(),
	}
}