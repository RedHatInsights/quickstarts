// Package middleware provides HTTP middleware for the quickstarts service.
package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/securitylog"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/sirupsen/logrus"
)

// isMutatingMethod returns true for HTTP methods that modify data.
// Auth failure logging is restricted to mutating methods to avoid noise
// from health probes, metrics scrapes, and public read endpoints.
func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// ExtractIdentity extracts the X-Rh-Identity header and stores the decoded
// identity in the request context. Unlike identity.EnforceIdentity, this
// middleware does NOT reject requests with missing or invalid headers — it
// logs an authentication failure security event for mutating requests and
// continues processing. This preserves backward compatibility while enabling
// SEC-MON-REQ-1 logging without generating noise from health probes, metrics
// scrapes, and read-only requests.
func ExtractIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawHeaders := r.Header["X-Rh-Identity"]
		if len(rawHeaders) != 1 {
			if isMutatingMethod(r.Method) {
				securitylog.LogWithReason(
					r.Context(),
					"AUTHENTICATE", "api_request", r.URL.Path,
					"failure", "missing x-rh-identity header",
				)
			}
			next.ServeHTTP(w, r)
			return
		}

		idRaw, err := base64.StdEncoding.DecodeString(rawHeaders[0])
		if err != nil {
			if isMutatingMethod(r.Method) {
				securitylog.LogWithReason(
					r.Context(),
					"AUTHENTICATE", "api_request", r.URL.Path,
					"failure", "unable to decode x-rh-identity header",
				)
			}
			next.ServeHTTP(w, r)
			return
		}

		var jsonData identity.XRHID
		if err := json.Unmarshal(idRaw, &jsonData); err != nil {
			logrus.WithError(err).Warn("unable to unmarshal x-rh-identity header")
			if isMutatingMethod(r.Method) {
				securitylog.LogWithReason(
					r.Context(),
					"AUTHENTICATE", "api_request", r.URL.Path,
					"failure", "invalid x-rh-identity JSON",
				)
			}
			next.ServeHTTP(w, r)
			return
		}

		// Fallback: use internal org_id if top-level is empty
		if jsonData.Identity.OrgID == "" && jsonData.Identity.Internal.OrgID != "" {
			jsonData.Identity.OrgID = jsonData.Identity.Internal.OrgID
		}

		ctx := context.WithValue(r.Context(), identity.Key, jsonData)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
