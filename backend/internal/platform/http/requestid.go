package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"unicode"
)

const requestIDHeader = "X-Request-Id"
const maxRequestIDLen = 128

type requestIDKey struct{}

// RequestIDFrom returns the request id stored on ctx, or "" if none.
func RequestIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey{}).(string)
	return v
}

// RequestID echoes a valid incoming X-Request-Id or generates one, sets it on
// the response, and stores it on the request context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get(requestIDHeader)
		if !validRequestID(rid) {
			rid = newRequestID()
		}
		w.Header().Set(requestIDHeader, rid)
		ctx := context.WithValue(r.Context(), requestIDKey{}, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func validRequestID(s string) bool {
	if s == "" || len(s) > maxRequestIDLen {
		return false
	}
	for _, r := range s {
		if r > unicode.MaxASCII || r < 0x21 || r > 0x7E {
			return false
		}
	}
	return true
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b[:])
}
