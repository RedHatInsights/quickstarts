package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/sirupsen/logrus"
)

const PSKHeader = "X-PSK-Token"

func PSKAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				logrus.Warn("PSK_TOKEN is not set; request accepted without authentication")
				next.ServeHTTP(w, r)
				return
			}

			provided := r.Header.Get(PSKHeader)
			if provided == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"status":"error","msg":"missing authentication token"}`))
				return
			}

			if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"status":"error","msg":"invalid authentication token"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
