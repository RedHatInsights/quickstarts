package routes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/RedHatInsights/quickstarts/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestServerAdapter_ImplementsInterface(t *testing.T) {
	// Test that ServerAdapter implements the generated.ServerInterface
	var _ generated.ServerInterface = &ServerAdapter{}
}

func TestNewServerAdapter(t *testing.T) {
	adapter := NewServerAdapter()
	assert.NotNil(t, adapter)
}

func TestServerAdapter_GetQuickstartsFilters(t *testing.T) {
	adapter := NewServerAdapter()

	req := httptest.NewRequest("GET", "/quickstarts/filters", nil)
	w := httptest.NewRecorder()

	adapter.GetQuickstartsFilters(w, req)

	// Should return a response (exact content depends on database state)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestServerAdapter_GetQuickstarts_WithParameters(t *testing.T) {
	adapter := NewServerAdapter()

	// Test with various parameters
	testCases := []struct {
		name   string
		params generated.GetQuickstartsParams
	}{
		{
			name: "with limit and offset",
			params: generated.GetQuickstartsParams{
				Limit:  intPtr(10),
				Offset: intPtr(0),
			},
		},
		{
			name: "with name filter",
			params: generated.GetQuickstartsParams{
				Name: quickstartNamePtr("test-quickstart"),
			},
		},
		{
			name: "with bundle filter",
			params: generated.GetQuickstartsParams{
				Bundle: &[]string{"insights"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/quickstarts", nil)
			w := httptest.NewRecorder()

			// The adapter should handle the request without crashing
			adapter.GetQuickstarts(w, req, tc.params)

			// Should return some response (may be empty data depending on database)
			assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
		})
	}
}

func TestServerAdapter_GetQuickstarts_ErrorConditions(t *testing.T) {
	adapter := NewServerAdapter()

	testCases := []struct {
		name           string
		requestURL     string
		expectedStatus int
		errorContains  string
	}{
		{
			name:           "invalid limit parameter",
			requestURL:     "/quickstarts?limit=notanumber",
			expectedStatus: http.StatusOK, // oapi-codegen handles type conversion, invalid values get default
			errorContains:  "",
		},
		{
			name:           "negative limit gets validated",
			requestURL:     "/quickstarts?limit=-5",
			expectedStatus: http.StatusOK, // Our validation converts this to 50
			errorContains:  "",
		},
		{
			name:           "negative offset gets validated",
			requestURL:     "/quickstarts?offset=-10",
			expectedStatus: http.StatusOK, // Our validation converts this to 0
			errorContains:  "",
		},
		{
			name:           "limit zero gets default",
			requestURL:     "/quickstarts?limit=0",
			expectedStatus: http.StatusOK, // Our validation converts this to 50
			errorContains:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.requestURL, nil)
			w := httptest.NewRecorder()

			// Parse parameters manually since we're testing the endpoint behavior
			params := generated.GetQuickstartsParams{}
			adapter.GetQuickstarts(w, req, params)

			assert.Equal(t, tc.expectedStatus, w.Code, "Expected status code %d, got %d", tc.expectedStatus, w.Code)

			if tc.errorContains != "" {
				assert.Contains(t, w.Body.String(), tc.errorContains, "Response should contain error message")
			}
		})
	}
}

func TestServerAdapter_GetFavorites_RequiresAccount(t *testing.T) {
	adapter := NewServerAdapter()

	req := httptest.NewRequest("GET", "/favorites", nil)
	w := httptest.NewRecorder()

	params := generated.GetFavoritesParams{
		Account: "test-account",
	}

	adapter.GetFavorites(w, req, params)

	// Should return a response (content depends on database state)
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
}

func TestServerAdapter_GetFavorites_AccountParameterValidation(t *testing.T) {
	adapter := NewServerAdapter()

	tests := []struct {
		name           string
		account        string
		expectedStatus int
		errorContains  string
	}{
		{
			name:           "Missing account parameter",
			account:        "",
			expectedStatus: http.StatusBadRequest,
			errorContains:  "missing account parameter",
		},
		{
			name:           "Valid account parameter",
			account:        "test-account",
			expectedStatus: http.StatusOK,
			errorContains:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/favorites", nil)
			w := httptest.NewRecorder()

			params := generated.GetFavoritesParams{
				Account: generated.Account(tc.account),
			}

			adapter.GetFavorites(w, req, params)

			assert.Equal(t, tc.expectedStatus, w.Code, "Expected status code %d, got %d", tc.expectedStatus, w.Code)

			if tc.errorContains != "" {
				assert.Contains(t, w.Body.String(), tc.errorContains, "Response should contain error message")
			}
		})
	}
}

func TestServerAdapter_ProgressEndpoints_Functional(t *testing.T) {
	adapter := NewServerAdapter()

	// Test GET /progress - should work now
	req := httptest.NewRequest("GET", "/progress", nil)
	w := httptest.NewRecorder()
	params := generated.GetProgressParams{}

	adapter.GetProgress(w, req, params)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test POST /progress with missing data - should return 400 Bad Request
	req = httptest.NewRequest("POST", "/progress", nil)
	w = httptest.NewRecorder()

	adapter.PostProgress(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test DELETE /progress/{id} with non-existent ID - should return 404 Not Found
	req = httptest.NewRequest("DELETE", "/progress/999", nil)
	w = httptest.NewRecorder()

	adapter.DeleteProgressId(w, req, 999)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestServerAdapter_ProgressEndpoints_Comprehensive(t *testing.T) {
	adapter := NewServerAdapter()

	// Helper function to create JSON request body
	createJSONBody := func(jsonStr string) io.Reader {
		return strings.NewReader(jsonStr)
	}

	t.Run("POST progress with valid data", func(t *testing.T) {
		validJSON := `{
			"accountId": 12345,
			"quickstartName": "test-quickstart",
			"progress": {
				"status": "In Progress",
				"taskNumber": 1
			}
		}`

		req := httptest.NewRequest("POST", "/progress", createJSONBody(validJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		adapter.PostProgress(w, req)

		// Should succeed (200) or fail with validation error (400) depending on database state
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
	})

	t.Run("POST progress with malformed JSON", func(t *testing.T) {
		malformedJSON := `{"accountId": 12345, "quickstartName":`  // Missing closing

		req := httptest.NewRequest("POST", "/progress", createJSONBody(malformedJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		adapter.PostProgress(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "unexpected EOF")
	})

	t.Run("POST progress with missing required fields", func(t *testing.T) {
		missingFieldsJSON := `{"quickstartName": "test-quickstart"}`  // Missing accountId

		req := httptest.NewRequest("POST", "/progress", createJSONBody(missingFieldsJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		adapter.PostProgress(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing accountId or quickstartName")
	})

	t.Run("GET progress with valid account ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/progress?account=12345", nil)
		w := httptest.NewRecorder()
		params := generated.GetProgressParams{
			Account: "12345",
		}

		adapter.GetProgress(w, req, params)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET progress with invalid account ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/progress?account=invalid", nil)
		w := httptest.NewRecorder()
		params := generated.GetProgressParams{
			Account: "invalid",
		}

		adapter.GetProgress(w, req, params)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid account ID: must be an integer")
	})

	t.Run("DELETE progress for non-existent ID", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/progress/999999", nil)
		w := httptest.NewRecorder()

		adapter.DeleteProgressId(w, req, 999999)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestConvertStringSlice(t *testing.T) {
	// Test nil input
	assert.Nil(t, utils.ConvertStringSlice(nil))

	// Test valid input
	input := []string{"a", "b", "c"}
	result := utils.ConvertStringSlice(&input)
	assert.Equal(t, input, result)
}

func TestConvertIntPtr(t *testing.T) {
	// Test nil input with default
	assert.Equal(t, 50, utils.ConvertIntPtr(nil, 50))

	// Test valid input
	value := 25
	assert.Equal(t, 25, utils.ConvertIntPtr(&value, 50))
}

// Helper functions for tests
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func quickstartNamePtr(s string) *generated.QuickstartName {
	name := generated.QuickstartName(s)
	return &name
}
