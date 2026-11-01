package httpx

import (
	"net/http"
	"time"
)

// SessionCookieName is the opaque session id cookie (feature 02).
const SessionCookieName = "friendfold_session"

// SessionCookieMaxAge is 30 days in seconds, matching the Redis TTL.
const SessionCookieMaxAge = int(30 * 24 * time.Hour / time.Second)

// SetSessionCookie writes friendfold_session with HttpOnly; SameSite=Lax; Path=/.
// Secure follows cfg.CookieSecure (off only for local HTTP).
func SetSessionCookie(w http.ResponseWriter, value string, secure bool) {
	http.SetCookie(w, sessionCookie(value, secure, SessionCookieMaxAge))
}

// ClearSessionCookie expires friendfold_session.
func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, sessionCookie("", secure, -1))
}

func sessionCookie(value string, secure bool, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}
