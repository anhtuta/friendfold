package profile

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	httpx "github.com/anhtuta/friendfold/backend/internal/platform/http"
	"github.com/anhtuta/friendfold/backend/internal/platform/session"
)

type ctxKey struct{}

// Handler serves /v1/me/profile and /v1/users/{username}.
type Handler struct {
	svc       *Service
	sessions  *session.Store
	webOrigin string
}

// NewHandler returns a profile HTTP handler.
func NewHandler(svc *Service, sessions *session.Store, webOrigin string) *Handler {
	return &Handler{svc: svc, sessions: sessions, webOrigin: webOrigin}
}

// Mount registers profile routes on r.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/v1/me/profile", func(r chi.Router) {
		r.Use(httpx.RequireOrigin(h.webOrigin))
		r.Use(h.requireSession)
		r.Get("/", h.getMe)
		r.Patch("/", h.patchMe)
	})
	r.Get("/v1/users/{username}", h.getPublic)
}

type patchBody struct {
	DisplayName *string `json:"displayName"`
	Bio         *string `json:"bio"`
	Username    *string `json:"username"`
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.GetMe(r.Context(), sessionUserID(r))
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeMe(w, out)
}

func (h *Handler) patchMe(w http.ResponseWriter, r *http.Request) {
	var body patchBody
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "Request is invalid")
		return
	}
	out, err := h.svc.PatchMe(r.Context(), PatchInput{
		UserID:      sessionUserID(r),
		IP:          httpx.ClientIP(r),
		DisplayName: body.DisplayName,
		Bio:         body.Bio,
		Username:    body.Username,
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeMe(w, out)
}

func (h *Handler) getPublic(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.GetPublic(r.Context(), chi.URLParam(r, "username"), h.optionalUserID(r), httpx.ClientIP(r))
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writePublic(w, out)
}

func (h *Handler) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(httpx.SessionCookieName)
		if err != nil || c.Value == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
			return
		}
		rec, err := h.sessions.GetAndTouch(r.Context(), c.Value)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
			return
		}
		httpx.SetLogUserID(r.Context(), rec.UserID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, rec.UserID)))
	})
}

func (h *Handler) optionalUserID(r *http.Request) int64 {
	c, err := r.Cookie(httpx.SessionCookieName)
	if err != nil || c.Value == "" {
		return 0
	}
	rec, err := h.sessions.GetAndTouch(r.Context(), c.Value)
	if err != nil {
		return 0
	}
	return rec.UserID
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	var ve ValidationError
	if errors.As(err, &ve) {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", ve.Msg)
		return
	}
	var rl RateLimitError
	if errors.As(err, &rl) {
		if rl.RetryAfter > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(rl.RetryAfter.Seconds())))
		}
		httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "Too many attempts. Try again later")
		return
	}
	switch {
	case errors.Is(err, ErrUsernameInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "username_invalid", "Username must be 3–20 characters [a-z0-9_]")
	case errors.Is(err, ErrUsernameReserved):
		httpx.WriteError(w, http.StatusBadRequest, "username_reserved", "This username is not available")
	case errors.Is(err, ErrUsernameTaken):
		httpx.WriteError(w, http.StatusConflict, "username_taken", "Username is already taken")
	case errors.Is(err, ErrUsernameCooldown):
		httpx.WriteError(w, http.StatusConflict, "username_cooldown", "Username can be changed again after the 14-day cooldown")
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Profile not found")
	case errors.Is(err, ErrUnauthenticated):
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Internal error")
	}
}

type meJSON struct {
	ID                string  `json:"id"`
	Username          *string `json:"username"`
	DisplayName       string  `json:"displayName"`
	Bio               string  `json:"bio"`
	AvatarURL         *string `json:"avatarUrl"`
	CoverURL          *string `json:"coverUrl"`
	Email             string  `json:"email"`
	EmailVerified     bool    `json:"emailVerified"`
	UsernameChangedAt *string `json:"usernameChangedAt"`
}

type publicJSON struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"displayName"`
	Bio         string  `json:"bio"`
	AvatarURL   *string `json:"avatarUrl"`
	CoverURL    *string `json:"coverUrl"`
}

func writeMe(w http.ResponseWriter, m Me) {
	body := meJSON{
		ID:            strconv.FormatInt(m.ID, 10),
		Username:      strPtr(m.Username),
		DisplayName:   m.DisplayName,
		Bio:           m.Bio,
		AvatarURL:     strPtr(m.AvatarURL),
		CoverURL:      strPtr(m.CoverURL),
		Email:         m.Email,
		EmailVerified: m.EmailVerified,
	}
	if m.HasUsernameChanged {
		ts := m.UsernameChangedAt.UTC().Format(time.RFC3339)
		body.UsernameChangedAt = &ts
	}
	httpx.WriteJSON(w, http.StatusOK, body)
}

func writePublic(w http.ResponseWriter, p Public) {
	httpx.WriteJSON(w, http.StatusOK, publicJSON{
		ID:          strconv.FormatInt(p.ID, 10),
		Username:    p.Username,
		DisplayName: p.DisplayName,
		Bio:         p.Bio,
		AvatarURL:   strPtr(p.AvatarURL),
		CoverURL:    strPtr(p.CoverURL),
	})
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func sessionUserID(r *http.Request) int64 {
	id, _ := r.Context().Value(ctxKey{}).(int64)
	return id
}
