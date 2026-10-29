package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteErrorEnvelope(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	var body errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.Error.Code != "invalid_credentials" {
		t.Errorf("code = %q", body.Error.Code)
	}
	if body.Error.Message != "Email or password is incorrect" {
		t.Errorf("message = %q", body.Error.Message)
	}
}

func TestDecodeJSONRejectsUnknownField(t *testing.T) {
	t.Parallel()
	var dst struct {
		Email string `json:"email"`
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@b.c","nope":1}`))
	err := DecodeJSON(rec, req, &dst)
	if !errors.Is(err, ErrInvalidJSON) {
		t.Fatalf("err = %v, want ErrInvalidJSON", err)
	}
}

func TestDecodeJSONAcceptsKnownFields(t *testing.T) {
	t.Parallel()
	var dst struct {
		Email string `json:"email"`
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@b.c"}`))
	if err := DecodeJSON(rec, req, &dst); err != nil {
		t.Fatalf("DecodeJSON: %v", err)
	}
	if dst.Email != "a@b.c" {
		t.Errorf("email = %q", dst.Email)
	}
}

func TestDecodeJSONRejectsEmptyBody(t *testing.T) {
	t.Parallel()
	var dst struct{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	if err := DecodeJSON(rec, req, &dst); !errors.Is(err, ErrInvalidJSON) {
		t.Fatalf("err = %v, want ErrInvalidJSON", err)
	}
}
