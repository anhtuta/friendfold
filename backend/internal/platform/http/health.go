package httpx

import (
	"net/http"
)

const healthBody = `{"status":"ok"}`

// Health is GET /v1/health: process liveness, no downstream checks.
func Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(healthBody))
}
