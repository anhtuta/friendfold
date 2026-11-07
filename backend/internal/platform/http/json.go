package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const maxJSONBody = 64 << 10 // 64 KiB

// ErrInvalidJSON is a malformed or oversized body, or unknown JSON field.
var ErrInvalidJSON = errors.New("invalid json")

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// WriteJSON encodes v as application/json.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes the shared error envelope.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, errorEnvelope{Error: errorBody{Code: code, Message: message}})
}

// DecodeJSON reads a JSON object into dst, rejecting unknown fields and bodies
// larger than 64 KiB.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.Body == nil {
		return fmt.Errorf("decode json: %w", ErrInvalidJSON)
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("decode json: %w", ErrInvalidJSON)
	}
	// Trailing junk is invalid.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("decode json: %w", ErrInvalidJSON)
	}
	return nil
}
