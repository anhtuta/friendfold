package profile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	httpx "github.com/anhtuta/friendfold/backend/internal/platform/http"
	"github.com/anhtuta/friendfold/backend/internal/platform/session"
)

func testProfileHandler(t *testing.T) (http.Handler, *session.Store, *memAccounts) {
	t.Helper()
	svc, _, acct, _ := testProfileService(t)
	sess := session.NewStore(svc.cache.rdb)
	h := NewHandler(svc, sess, "http://localhost:3000")
	r := chi.NewRouter()
	r.Use(httpx.RequestID)
	h.Mount(r)
	return r, sess, acct
}

func doProfileJSON(t *testing.T, h http.Handler, method, path, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Origin", "http://localhost:3000")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandlerMeRequiresSession(t *testing.T) {
	t.Parallel()
	h, _, _ := testProfileHandler(t)
	rec := doProfileJSON(t, h, http.MethodGet, "/v1/me/profile", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerPatchOrigin(t *testing.T) {
	t.Parallel()
	h, sess, acct := testProfileHandler(t)
	acct.put(Account{ID: 9, Email: "minh@example.com", DisplayName: "Minh"})
	sid, err := sess.Create(context.Background(), session.Record{UserID: 9, UserAgent: "t", IP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/v1/me/profile", strings.NewReader(`{"bio":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://evil.example")
	req.AddCookie(&http.Cookie{Name: httpx.SessionCookieName, Value: sid})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRoundTrip(t *testing.T) {
	t.Parallel()
	h, sess, acct := testProfileHandler(t)
	acct.put(Account{ID: 11, Email: "minh@example.com", EmailVerified: true, DisplayName: "Minh"})
	sid, err := sess.Create(context.Background(), session.Record{UserID: 11, UserAgent: "t", IP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	cookies := []*http.Cookie{{Name: httpx.SessionCookieName, Value: sid}}
	patch := doProfileJSON(t, h, http.MethodPatch, "/v1/me/profile", `{"username":"minh","bio":"Hello"}`, cookies)
	if patch.Code != http.StatusOK {
		t.Fatalf("patch %d %s", patch.Code, patch.Body.String())
	}
	var me meJSON
	if err := json.Unmarshal(patch.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me.Email != "minh@example.com" || me.Username == nil || *me.Username != "minh" {
		t.Fatalf("%+v", me)
	}
	pub := doProfileJSON(t, h, http.MethodGet, "/v1/users/minh", "", nil)
	if pub.Code != http.StatusOK {
		t.Fatalf("public %d %s", pub.Code, pub.Body.String())
	}
	if strings.Contains(pub.Body.String(), "email") || strings.Contains(pub.Body.String(), "minh@example.com") {
		t.Fatalf("leaked email: %s", pub.Body.String())
	}
	missing := doProfileJSON(t, h, http.MethodGet, "/v1/users/nobodyhere", "", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing %d %s", missing.Code, missing.Body.String())
	}
}

func TestHandlerReservedUsernameHTTP(t *testing.T) {
	t.Parallel()
	h, sess, acct := testProfileHandler(t)
	acct.put(Account{ID: 13, Email: "a@example.com", DisplayName: "A"})
	sid, err := sess.Create(context.Background(), session.Record{UserID: 13})
	if err != nil {
		t.Fatal(err)
	}
	cookies := []*http.Cookie{{Name: httpx.SessionCookieName, Value: sid}}
	rec := doProfileJSON(t, h, http.MethodPatch, "/v1/me/profile", `{"username":"admin"}`, cookies)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("reserved %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "username_reserved") {
		t.Fatalf("code: %s", rec.Body.String())
	}
	missing := doProfileJSON(t, h, http.MethodGet, "/v1/users/admin", "", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("reserved public %d %s", missing.Code, missing.Body.String())
	}
}

func TestHandlerUsernameCooldownHTTP(t *testing.T) {
	t.Parallel()
	h, sess, acct := testProfileHandler(t)
	acct.put(Account{ID: 12, Email: "a@example.com", DisplayName: "A"})
	sid, err := sess.Create(context.Background(), session.Record{UserID: 12})
	if err != nil {
		t.Fatal(err)
	}
	cookies := []*http.Cookie{{Name: httpx.SessionCookieName, Value: sid}}
	if rec := doProfileJSON(t, h, http.MethodPatch, "/v1/me/profile", `{"username":"minh"}`, cookies); rec.Code != http.StatusOK {
		t.Fatalf("first %d %s", rec.Code, rec.Body.String())
	}
	rec := doProfileJSON(t, h, http.MethodPatch, "/v1/me/profile", `{"username":"minh2"}`, cookies)
	if rec.Code != http.StatusConflict {
		t.Fatalf("cooldown %d %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Error.Code != "username_cooldown" {
		t.Fatalf("code %s", env.Error.Code)
	}
}
