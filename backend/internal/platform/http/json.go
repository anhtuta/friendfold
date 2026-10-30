package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// WriteJSON encodes v as application/json. extraHeaders are applied before the body.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		return
	}
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
		return wrapJSONError(err)
	}
	// Trailing junk is invalid.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("decode json: %w", ErrInvalidJSON)
	}
	return nil
}

func wrapJSONError(err error) error {
	if err == nil {
		return nil
	}
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return fmt.Errorf("decode json: %w", ErrInvalidJSON)
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("decode json: %w", ErrInvalidJSON)
	}
	msg := err.Error()
	if strings.Contains(msg, "unknown field") {
		return fmt.Errorf("decode json: %w", ErrInvalidJSON)
	}
	return fmt.Errorf("decode json: %w", ErrInvalidJSON)
}
