package routes

import (
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/RedHatInsights/quickstarts/pkg/utils"
)

// GetHelptopics handles GET /helptopics
func (s *ServerAdapter) GetHelptopics(w http.ResponseWriter, r *http.Request, params generated.GetHelptopicsParams) {
	// Extract parameters from generated params
	var bundleQueries []string
	var applicationQueries []string
	var nameQueries []string

	if params.Bundle != nil {
		bundleQueries = utils.ConvertStringSlice(params.Bundle)
	}
	if params.Application != nil {
		applicationQueries = utils.ConvertStringSlice(params.Application)
	}
	if params.Name != nil {
		nameQueries = utils.ConvertStringSlice(params.Name)
	}

	// Use service layer for data access
	helpTopics, err := s.helpTopicService.FindWithFilters(bundleQueries, applicationQueries, nameQueries)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Convert to generated types and respond
	genHelpTopics := make([]generated.HelpTopic, len(helpTopics))
	for i, topic := range helpTopics {
		genHelpTopics[i] = topic.ToAPI()
	}

	utils.DataResponse(w, http.StatusOK, genHelpTopics)
}

// GetHelptopicsName handles GET /helptopics/{name}
func (s *ServerAdapter) GetHelptopicsName(w http.ResponseWriter, r *http.Request, name string) {
	// Find the help topic by name using service
	helpTopic, err := s.helpTopicService.FindByName(name)
	if err != nil {
		utils.NotFoundResponse(w, "Help topic")
		return
	}

	// Convert to generated type and respond
	genHelpTopic := helpTopic.ToAPI()
	utils.DataResponse(w, http.StatusOK, genHelpTopic)
}