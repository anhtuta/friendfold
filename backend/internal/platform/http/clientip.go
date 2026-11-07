package httpx

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP returns the remote host, ignoring X-Forwarded-For until a trusted
// proxy is configured (attackers can spoof that header).
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
