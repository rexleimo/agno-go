package middleware

import (
	"crypto/subtle"
	"log"
	"net/http"
	"strings"
)

// DefaultAPIKeyHeader is the accepted header for FR-004.
const DefaultAPIKeyHeader = "X-API-Key"

// APIKeyAuth enforces API Key header authentication and explicitly rejects
// other auth schemes such as Basic/OAuth/JWT. The expected key should be
// provided by the caller (e.g., env/config). If expectedKey is empty the
// middleware allows the request to proceed but still blocks unsupported
// Authorization headers.
func APIKeyAuth(expectedKey, header string, logger *log.Logger) func(http.Handler) http.Handler {
	expectedKey = strings.TrimSpace(expectedKey)
	header = strings.TrimSpace(header)
	if header == "" {
		header = DefaultAPIKeyHeader
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := strings.TrimSpace(r.Header.Get("Authorization"))
			if authz != "" {
				reason := "unsupported auth scheme"
				http.Error(w, reason, http.StatusUnauthorized)
				if logger != nil {
					logger.Printf("auth rejected: %s", reason)
				}
				return
			}

			provided := strings.TrimSpace(r.Header.Get(header))
			if expectedKey == "" {
				next.ServeHTTP(w, r)
				return
			}
			if provided == "" {
				http.Error(w, "missing API key", http.StatusUnauthorized)
				return
			}
			if subtle.ConstantTimeCompare([]byte(provided), []byte(expectedKey)) != 1 {
				http.Error(w, "invalid API key", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
