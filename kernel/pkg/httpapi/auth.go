package httpapi

import (
	"net/http"
	"os"
	"strings"
)

// APITokenFromEnv returns HERMES_API_TOKEN if set (empty = open local-dev mode).
func APITokenFromEnv() string {
	return strings.TrimSpace(os.Getenv("HERMES_API_TOKEN"))
}

// withAuth enforces Bearer token when HERMES_API_TOKEN is set.
// Health endpoints remain open for probes.
func withAuth(next http.Handler) http.Handler {
	token := APITokenFromEnv()
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always allow CORS preflight
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		// Public probes
		if r.URL.Path == "/health" || r.URL.Path == "/api/v1/health" {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		const p = "Bearer "
		if !strings.HasPrefix(auth, p) || subtleConstantTimeEq(strings.TrimPrefix(auth, p), token) == false {
			// also accept X-Hermes-Token
			if r.Header.Get("X-Hermes-Token") != token {
				writeErr(w, 401, "unauthorized", "missing or invalid token", "Set Authorization: Bearer $HERMES_API_TOKEN")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func subtleConstantTimeEq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
