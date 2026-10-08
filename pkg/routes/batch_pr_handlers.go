package routes

import (
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/clients"
	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/RedHatInsights/quickstarts/pkg/utils"
)

// PostPullRequestBatch handles POST /pull-request/batch
func (s *ServerAdapter) PostPullRequestBatch(w http.ResponseWriter, r *http.Request) {
	if !s.gitServiceEnabled {
		utils.ErrorResponse(w, http.StatusNotFound, "git-service is not available")
		return
	}

	const maxRequestSize = 5 * 1024 * 1024 // 5MB
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)

	var reqBody generated.BatchPrRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		if err.Error() == "http: request body too large" {
			utils.ErrorResponse(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(reqBody.Resources) == 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "resources are required")
		return
	}
	if reqBody.Metadata.CommitMessage == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "commitMessage is required")
		return
	}

	resources := make([]clients.GitServiceBatchResource, len(reqBody.Resources))
	for i, res := range reqBody.Resources {
		files := make([]clients.GitServiceFile, len(res.Files))
		for j, f := range res.Files {
			files[j] = clients.GitServiceFile{Name: f.Name, Content: f.Content}
		}
		resources[i] = clients.GitServiceBatchResource{
			Name:   res.Name,
			Create: res.Create != nil && *res.Create,
			Files:  files,
		}
	}

	m := reqBody.Metadata
	result, err := s.gitServiceClient.SubmitBatchPR(r.Context(), clients.GitServiceBatchRequest{
		Metadata: clients.GitServiceBatchMetadata{
			PRNumber:        derefInt(m.PrNumber),
			ExpectedHeadSHA: derefString(m.ExpectedHeadSha),
			BranchName:      derefString(m.BranchName),
			BaseSHA:         derefString(m.BaseSha),
			CommitMessage:   m.CommitMessage,
			PRTitle:         derefString(m.PrTitle),
			PRBody:          derefString(m.PrBody),
			UserEmail:       derefString(m.UserEmail),
		},
		Resources: resources,
	})
	if err != nil {
		writeGitServiceError(w, err)
		return
	}

	paths := result.Paths
	if paths == nil {
		paths = []string{}
	}
	utils.DataResponse(w, http.StatusOK, generated.BatchPrResponse{
		PrUrl:      &result.PRURL,
		PrNumber:   &result.PRNumber,
		BranchName: &result.BranchName,
		HeadSha:    &result.HeadSHA,
		Status:     &result.Status,
		Paths:      &paths,
	})
}

// GetCreatorPrsNumberWorkspace handles GET /creator-prs/{number}/workspace
func (s *ServerAdapter) GetCreatorPrsNumberWorkspace(w http.ResponseWriter, r *http.Request, number int) {
	if !s.gitServiceEnabled {
		utils.ErrorResponse(w, http.StatusNotFound, "git-service is not available")
		return
	}

	result, err := s.gitServiceClient.GetCreatorPRWorkspace(r.Context(), number)
	if err != nil {
		writeGitServiceError(w, err)
		return
	}

	resources := make([]generated.WorkspaceResource, len(result.Resources))
	for i, res := range result.Resources {
		resources[i] = toGeneratedResource(res)
	}
	missing := result.Missing
	if missing == nil {
		missing = []string{}
	}

	utils.DataResponse(w, http.StatusOK, generated.CreatorPrWorkspaceResponse{
		Number:     &result.Number,
		Title:      &result.Title,
		HtmlUrl:    &result.HTMLURL,
		BranchName: &result.BranchName,
		HeadSha:    &result.HeadSHA,
		Resources:  &resources,
		Missing:    &missing,
	})
}

// GetCreatorPrsNumberResourcesName handles GET /creator-prs/{number}/resources/{name}
func (s *ServerAdapter) GetCreatorPrsNumberResourcesName(w http.ResponseWriter, r *http.Request, number int, name string, params generated.GetCreatorPrsNumberResourcesNameParams) {
	if !s.gitServiceEnabled {
		utils.ErrorResponse(w, http.StatusNotFound, "git-service is not available")
		return
	}

	result, err := s.gitServiceClient.GetCreatorPRResource(r.Context(), number, name, derefString(params.HeadSha))
	if err != nil {
		writeGitServiceError(w, err)
		return
	}

	resource := toGeneratedResource(result.Resource)
	utils.DataResponse(w, http.StatusOK, generated.CreatorPrResourceResponse{
		HeadSha:  &result.HeadSHA,
		Resource: &resource,
	})
}

func toGeneratedResource(res clients.GitServiceWorkspaceResource) generated.WorkspaceResource {
	files := make([]generated.ResourceFile, len(res.Files))
	for i, f := range res.Files {
		name, path, content := f.Name, f.Path, f.Content
		files[i] = generated.ResourceFile{Name: &name, Path: &path, Content: &content}
	}
	name, path := res.Name, res.Path
	return generated.WorkspaceResource{Name: &name, Path: &path, Files: &files}
}

// optionalString omits an empty value from a response rather than sending "".
func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
