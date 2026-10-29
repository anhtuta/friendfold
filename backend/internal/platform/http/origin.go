package httpx

import (
	"net"
	"net/http"
	"strings"
)

// RequireOrigin rejects mutating requests whose Origin is missing or not
// exactly webOrigin. GET, HEAD, and OPTIONS pass through (CORS handles OPTIONS).
func RequireOrigin(webOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !mutatingMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			origin := r.Header.Get("Origin")
			if origin == "" || origin != webOrigin {
				WriteError(w, http.StatusForbidden, "origin_mismatch", "Request origin is not allowed")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func mutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// ClientIP returns the remote host, ignoring X-Forwarded-For until a trusted
// proxy is configured (attackers can spoof that header).
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
