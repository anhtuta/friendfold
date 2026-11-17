package profile

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	minUsernameLen   = 3
	maxUsernameLen   = 20
	maxBioRunes      = 160
	maxDisplayRunes  = 80
	usernameCooldown = 14 * 24 * time.Hour
)

// reservedUsernames is the F-03-02 denylist (lowercase).
var reservedUsernames = map[string]struct{}{
	"admin": {}, "support": {}, "help": {}, "friendfold": {},
	"settings": {}, "setting": {}, "api": {}, "u": {}, "me": {},
	"login": {}, "logout": {}, "signup": {}, "register": {},
	"health": {}, "ready": {}, "onboarding": {}, "username": {},
	"profile": {}, "profiles": {}, "account": {}, "accounts": {},
	"auth": {}, "oauth": {}, "google": {}, "password": {}, "reset": {},
	"verify": {}, "email": {}, "null": {}, "undefined": {}, "www": {},
	"mail": {}, "static": {}, "assets": {}, "media": {}, "about": {},
	"privacy": {}, "terms": {}, "legal": {}, "safety": {}, "moderation": {},
	"mod": {}, "root": {}, "system": {}, "status": {},
}

func normalizeUsername(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if n := utf8.RuneCountInString(s); n < minUsernameLen || n > maxUsernameLen {
		return "", ErrUsernameInvalid
	}
	for _, r := range s {
		if r != '_' && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return "", ErrUsernameInvalid
		}
	}
	if _, reserved := reservedUsernames[s]; reserved {
		return "", ErrUsernameReserved
	}
	return s, nil
}

func normalizeDisplayName(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if hasControls(s) {
		return "", ValidationError{Msg: "display name must not contain control characters"}
	}
	n := utf8.RuneCountInString(s)
	if n < 1 || n > maxDisplayRunes {
		return "", ValidationError{Msg: "display name must be 1–80 characters"}
	}
	return s, nil
}

func normalizeBio(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if hasControls(s) {
		return "", ValidationError{Msg: "bio must not contain control characters"}
	}
	if utf8.RuneCountInString(s) > maxBioRunes {
		return "", ValidationError{Msg: "bio must be at most 160 characters"}
	}
	return s, nil
}

func hasControls(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
