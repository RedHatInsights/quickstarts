package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func TestPSKAuth_ValidToken(t *testing.T) {
	handler := PSKAuth("test-secret")(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", nil)
	req.Header.Set(PSKHeader, "test-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestPSKAuth_MissingHeader(t *testing.T) {
	handler := PSKAuth("test-secret")(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing authentication token")
}

func TestPSKAuth_InvalidToken(t *testing.T) {
	handler := PSKAuth("test-secret")(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", nil)
	req.Header.Set(PSKHeader, "wrong-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid authentication token")
}

func TestPSKAuth_EmptyConfigToken(t *testing.T) {
	handler := PSKAuth("")(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}
