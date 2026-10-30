package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetSessionCookieFlags(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	SetSessionCookie(rec, "sid-value", false)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	c := cookies[0]
	if c.Name != SessionCookieName {
		t.Errorf("Name = %q", c.Name)
	}
	if c.Value != "sid-value" {
		t.Errorf("Value = %q", c.Value)
	}
	if !c.HttpOnly {
		t.Error("HttpOnly = false")
	}
	if c.Secure {
		t.Error("Secure = true, want false for local HTTP")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", c.SameSite)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q", c.Path)
	}
	if c.MaxAge != SessionCookieMaxAge {
		t.Errorf("MaxAge = %d", c.MaxAge)
	}
}

func TestSetSessionCookieSecureWhenConfigured(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	SetSessionCookie(rec, "sid", true)
	c := rec.Result().Cookies()[0]
	if !c.Secure {
		t.Error("Secure = false, want true")
	}
}

func TestClearSessionCookieExpires(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	ClearSessionCookie(rec, false)
	c := rec.Result().Cookies()[0]
	if c.Name != SessionCookieName {
		t.Errorf("Name = %q", c.Name)
	}
	if c.Value != "" {
		t.Errorf("Value = %q, want empty", c.Value)
	}
	if c.MaxAge >= 0 {
		t.Errorf("MaxAge = %d, want negative", c.MaxAge)
	}
	if !c.HttpOnly {
		t.Error("HttpOnly = false on clear")
	}
}
