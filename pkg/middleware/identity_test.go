package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/stretchr/testify/assert"
)

func TestExtractIdentity_ValidHeader(t *testing.T) {
	xrhid := identity.XRHID{
		Identity: identity.Identity{
			OrgID:    "org-123",
			User:     identity.User{UserID: "user-456"},
			Type:     "User",
			AuthType: "basic-auth",
		},
	}
	raw, _ := json.Marshal(xrhid)
	encoded := base64.StdEncoding.EncodeToString(raw)

	var extractedID identity.XRHID
	handler := ExtractIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id, ok := r.Context().Value(identity.Key).(identity.XRHID); ok {
			extractedID = id
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/quickstarts/v1/quickstarts", nil)
	req.Header.Set("X-Rh-Identity", encoded)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "org-123", extractedID.Identity.OrgID)
	assert.Equal(t, "user-456", extractedID.Identity.User.UserID)
}

func TestExtractIdentity_MissingHeader(t *testing.T) {
	called := false
	handler := ExtractIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// Identity should NOT be in context
		_, ok := r.Context().Value(identity.Key).(identity.XRHID)
		// Value type always succeeds in type assertion, check UserID instead
		if ok {
			id := r.Context().Value(identity.Key).(identity.XRHID)
			assert.Empty(t, id.Identity.OrgID, "should not have org_id without header")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/quickstarts/v1/quickstarts", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.True(t, called, "next handler should be called even without identity")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestExtractIdentity_InvalidBase64(t *testing.T) {
	called := false
	handler := ExtractIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/quickstarts/v1/quickstarts", nil)
	req.Header.Set("X-Rh-Identity", "not-valid-base64!!!")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.True(t, called, "next handler should be called even with invalid base64")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestExtractIdentity_InvalidJSON(t *testing.T) {
	called := false
	handler := ExtractIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	encoded := base64.StdEncoding.EncodeToString([]byte("{invalid json"))
	req := httptest.NewRequest(http.MethodGet, "/api/quickstarts/v1/quickstarts", nil)
	req.Header.Set("X-Rh-Identity", encoded)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.True(t, called, "next handler should be called even with invalid JSON")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestExtractIdentity_InternalOrgIDFallback(t *testing.T) {
	xrhid := identity.XRHID{
		Identity: identity.Identity{
			OrgID:    "",
			Internal: identity.Internal{OrgID: "internal-org-999"},
			Type:     "User",
		},
	}
	raw, _ := json.Marshal(xrhid)
	encoded := base64.StdEncoding.EncodeToString(raw)

	var extractedID identity.XRHID
	handler := ExtractIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		extractedID = r.Context().Value(identity.Key).(identity.XRHID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/quickstarts/v1/quickstarts", nil)
	req.Header.Set("X-Rh-Identity", encoded)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "internal-org-999", extractedID.Identity.OrgID)
}
