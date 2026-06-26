package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateRequest_MissingFiles(t *testing.T) {
	req := &SubmitPRRequest{
		Files: []File{},
		Metadata: PRMetadata{
			BranchName:    "test",
			CommitMessage: "msg",
			PRTitle:       "title",
			PRBody:        "body",
			UserEmail:     "test@test.com",
		},
	}
	err := validateRequest(req)
	assert.EqualError(t, err, "files are required")
}

func TestValidateRequest_MissingBranchName(t *testing.T) {
	req := &SubmitPRRequest{
		Files: []File{{Name: "f", Content: "c"}},
		Metadata: PRMetadata{
			CommitMessage: "msg",
			PRTitle:       "title",
			PRBody:        "body",
			UserEmail:     "test@test.com",
		},
	}
	err := validateRequest(req)
	assert.EqualError(t, err, "branchName is required")
}

func TestValidateRequest_MissingExistingPathOnUpdate(t *testing.T) {
	req := &SubmitPRRequest{
		Files: []File{{Name: "f", Content: "c"}},
		Metadata: PRMetadata{
			BranchName:    "test",
			CommitMessage: "msg",
			PRTitle:       "title",
			PRBody:        "body",
			UserEmail:     "test@test.com",
			IsUpdate:      true,
		},
	}
	err := validateRequest(req)
	assert.EqualError(t, err, "existingPath is required when isUpdate is true")
}

func TestValidateRequest_Valid(t *testing.T) {
	req := &SubmitPRRequest{
		Files: []File{{Name: "f", Content: "c"}},
		Metadata: PRMetadata{
			BranchName:    "test",
			CommitMessage: "msg",
			PRTitle:       "title",
			PRBody:        "body",
			UserEmail:     "test@test.com",
		},
	}
	err := validateRequest(req)
	assert.NoError(t, err)
}

func TestSubmitPR_InvalidJSON(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString("not json"))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.Equal(t, "invalid request body", resp["msg"])
}

func TestSubmitPR_MissingFields(t *testing.T) {
	handler := &Handler{}
	body := `{"files":[], "metadata":{"branchName":"test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
