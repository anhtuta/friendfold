package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	httpx "github.com/anhtuta/friendfold/backend/internal/platform/http"
)

const testOrigin = "http://localhost:3000"

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	svc, _ := testService(t)
	h := NewHandler(svc, false, testOrigin)
	r := chi.NewRouter()
	r.Use(httpx.RequestID)
	h.Mount(r)
	return r
}

func doJSON(t *testing.T, h http.Handler, method, path, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", testOrigin)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == httpx.SessionCookieName {
			return c
		}
	}
	t.Fatal("missing session cookie")
	return nil
}

func TestRegisterLoginMeLogoutRoundTrip(t *testing.T) {
	t.Parallel()
	h := testHandler(t)
	registerBody := `{
		"email":"minh@example.com",
		"password":"correct-horse-battery",
		"displayName":"Minh",
		"dateOfBirth":"1998-04-12",
		"termsAccepted":true
	}`
	reg := doJSON(t, h, http.MethodPost, "/v1/auth/register", registerBody, nil)
	if reg.Code != http.StatusCreated {
		t.Fatalf("register status = %d body=%s", reg.Code, reg.Body.String())
	}
	cookie := sessionCookie(t, reg)
	if !cookie.HttpOnly {
		t.Fatal("cookie not HttpOnly")
	}
	if strings.Contains(reg.Body.String(), "password") {
		t.Fatalf("response leaked password: %s", reg.Body.String())
	}
	var created userBody
	if err := json.Unmarshal(reg.Body.Bytes(), &created); err != nil {
		t.Fatalf("json: %v", err)
	}
	if created.Email != "minh@example.com" || created.EmailVerified || created.ID == "" {
		t.Fatalf("user = %+v", created)
	}

	me := doJSON(t, h, http.MethodGet, "/v1/auth/me", "", []*http.Cookie{cookie})
	if me.Code != http.StatusOK {
		t.Fatalf("me status = %d body=%s", me.Code, me.Body.String())
	}

	logout := doJSON(t, h, http.MethodPost, "/v1/auth/logout", "", []*http.Cookie{cookie})
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d", logout.Code)
	}
	me2 := doJSON(t, h, http.MethodGet, "/v1/auth/me", "", []*http.Cookie{cookie})
	if me2.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout = %d body=%s", me2.Code, me2.Body.String())
	}

	login := doJSON(t, h, http.MethodPost, "/v1/auth/login",
		`{"email":"minh@example.com","password":"correct-horse-battery"}`, nil)
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", login.Code, login.Body.String())
	}
}

func TestLoginUnknownAndBadPasswordSame401(t *testing.T) {
	t.Parallel()
	h := testHandler(t)
	doJSON(t, h, http.MethodPost, "/v1/auth/register", `{
		"email":"minh@example.com",
		"password":"correct-horse-battery",
		"displayName":"Minh",
		"dateOfBirth":"1998-04-12",
		"termsAccepted":true
	}`, nil)

	unknown := doJSON(t, h, http.MethodPost, "/v1/auth/login",
		`{"email":"nobody@example.com","password":"correct-horse-battery"}`, nil)
	bad := doJSON(t, h, http.MethodPost, "/v1/auth/login",
		`{"email":"minh@example.com","password":"definitely-wrong-1"}`, nil)
	if unknown.Code != http.StatusUnauthorized || bad.Code != http.StatusUnauthorized {
		t.Fatalf("status unknown=%d bad=%d", unknown.Code, bad.Code)
	}
	if unknown.Body.String() != bad.Body.String() {
		t.Fatalf("bodies differ:\n%s\n%s", unknown.Body.String(), bad.Body.String())
	}
	if !strings.Contains(unknown.Body.String(), `"invalid_credentials"`) {
		t.Fatalf("body = %s", unknown.Body.String())
	}
}

func TestRegisterValidation(t *testing.T) {
	t.Parallel()
	h := testHandler(t)
	under := doJSON(t, h, http.MethodPost, "/v1/auth/register", `{
		"email":"kid@example.com",
		"password":"correct-horse-battery",
		"displayName":"Kid",
		"dateOfBirth":"2015-01-01",
		"termsAccepted":true
	}`, nil)
	if under.Code != http.StatusBadRequest || !strings.Contains(under.Body.String(), "underage") {
		t.Fatalf("underage: %d %s", under.Code, under.Body.String())
	}

	terms := doJSON(t, h, http.MethodPost, "/v1/auth/register", `{
		"email":"minh@example.com",
		"password":"correct-horse-battery",
		"displayName":"Minh",
		"dateOfBirth":"1998-04-12",
		"termsAccepted":false
	}`, nil)
	if terms.Code != http.StatusBadRequest || !strings.Contains(terms.Body.String(), "terms_required") {
		t.Fatalf("terms: %d %s", terms.Code, terms.Body.String())
	}

	unknown := doJSON(t, h, http.MethodPost, "/v1/auth/register", `{
		"email":"minh@example.com",
		"password":"correct-horse-battery",
		"displayName":"Minh",
		"dateOfBirth":"1998-04-12",
		"termsAccepted":true,
		"extra":true
	}`, nil)
	if unknown.Code != http.StatusBadRequest || !strings.Contains(unknown.Body.String(), "validation_error") {
		t.Fatalf("unknown field: %d %s", unknown.Code, unknown.Body.String())
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	t.Parallel()
	h := testHandler(t)
	body := `{
		"email":"minh@example.com",
		"password":"correct-horse-battery",
		"displayName":"Minh",
		"dateOfBirth":"1998-04-12",
		"termsAccepted":true
	}`
	if rec := doJSON(t, h, http.MethodPost, "/v1/auth/register", body, nil); rec.Code != http.StatusCreated {
		t.Fatalf("first register: %d %s", rec.Code, rec.Body.String())
	}
	dup := doJSON(t, h, http.MethodPost, "/v1/auth/register", body, nil)
	if dup.Code != http.StatusConflict || !strings.Contains(dup.Body.String(), "email_taken") {
		t.Fatalf("dup: %d %s", dup.Code, dup.Body.String())
	}
}

func TestOriginRejectedOnRegister(t *testing.T) {
	t.Parallel()
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "origin_mismatch") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestMeUnauthorizedWithoutCookie(t *testing.T) {
	t.Parallel()
	h := testHandler(t)
	rec := doJSON(t, h, http.MethodGet, "/v1/auth/me", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCookieNotSecureWhenConfiguredOff(t *testing.T) {
	t.Parallel()
	h := testHandler(t)
	reg := doJSON(t, h, http.MethodPost, "/v1/auth/register", `{
		"email":"minh@example.com",
		"password":"correct-horse-battery",
		"displayName":"Minh",
		"dateOfBirth":"1998-04-12",
		"termsAccepted":true
	}`, nil)
	c := sessionCookie(t, reg)
	if c.Secure {
		t.Fatal("Secure should be off when cookieSecure=false")
	}
}
