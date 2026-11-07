package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPSplitsHostPort(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.9:54321"
	if got := ClientIP(req); got != "203.0.113.9" {
		t.Fatalf("ClientIP = %q", got)
	}
}
