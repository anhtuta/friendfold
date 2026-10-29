package auth

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

// Handler serves /v1/auth/*.
type Handler struct {
	svc          *Service
	cookieSecure bool
	webOrigin    string
}

// NewHandler returns an auth HTTP handler.
func NewHandler(svc *Service, cookieSecure bool, webOrigin string) *Handler {
	return &Handler{svc: svc, cookieSecure: cookieSecure, webOrigin: webOrigin}
}

// Mount registers auth routes on r.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/v1/auth", func(r chi.Router) {
		r.Use(httpx.RequireOrigin(h.webOrigin))
		r.Post("/register", h.register)
		r.Post("/login", h.login)
		r.Group(func(r chi.Router) {
			r.Use(h.requireSession)
			r.Get("/me", h.me)
			r.Post("/logout", h.logout)
		})
	})
}

type registerBody struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	DisplayName   string `json:"displayName"`
	DateOfBirth   string `json:"dateOfBirth"`
	TermsAccepted bool   `json:"termsAccepted"`
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userBody struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
	DisplayName   string `json:"displayName"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var body registerBody
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "Request is invalid")
		return
	}
	dob, err := time.Parse("2006-01-02", body.DateOfBirth)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "date of birth is invalid")
		return
	}
	out, err := h.svc.Register(r.Context(), RegisterInput{
		Email:         body.Email,
		Password:      body.Password,
		DisplayName:   body.DisplayName,
		DateOfBirth:   dob,
		TermsAccepted: body.TermsAccepted,
		UserAgent:     r.UserAgent(),
		IP:            httpx.ClientIP(r),
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}
	httpx.SetSessionCookie(w, out.SessionID, h.cookieSecure)
	writeUser(w, http.StatusCreated, out.User)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "Request is invalid")
		return
	}
	out, err := h.svc.Login(r.Context(), LoginInput{
		Email:     body.Email,
		Password:  body.Password,
		UserAgent: r.UserAgent(),
		IP:        httpx.ClientIP(r),
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}
	httpx.SetSessionCookie(w, out.SessionID, h.cookieSecure)
	writeUser(w, http.StatusOK, out.User)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	u, _ := r.Context().Value(ctxKey{}).(User)
	writeUser(w, http.StatusOK, u)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(httpx.SessionCookieName)
	if err == nil {
		_ = h.svc.Logout(r.Context(), c.Value)
	}
	httpx.ClearSessionCookie(w, h.cookieSecure)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(httpx.SessionCookieName)
		if err != nil || c.Value == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
			return
		}
		u, err := h.svc.Me(r.Context(), c.Value)
		if err != nil {
			h.writeErr(w, err)
			return
		}
		httpx.SetLogUserID(r.Context(), u.ID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
	})
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
	case errors.Is(err, ErrTermsRequired):
		httpx.WriteError(w, http.StatusBadRequest, "terms_required", "Terms of use and privacy policy must be accepted")
	case errors.Is(err, ErrUnderage):
		httpx.WriteError(w, http.StatusBadRequest, "underage", "You must be at least 13 years old")
	case errors.Is(err, ErrEmailTaken):
		httpx.WriteError(w, http.StatusConflict, "email_taken", "Email is already registered")
	case errors.Is(err, ErrInvalidCredentials):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect")
	case errors.Is(err, ErrUnauthenticated), errors.Is(err, session.ErrNotFound):
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Authentication required")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Internal error")
	}
}

func writeUser(w http.ResponseWriter, status int, u User) {
	httpx.WriteJSON(w, status, userBody{
		ID:            strconv.FormatInt(u.ID, 10),
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		DisplayName:   u.DisplayName,
	})
}
