package profile

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
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
		r.Post("/avatar", h.postAvatar)
		r.Delete("/avatar", h.deleteAvatar)
		r.Post("/cover", h.postCover)
		r.Delete("/cover", h.deleteCover)
	})
	r.Get("/v1/users/{username}", h.getPublic)
	r.Get("/v1/media/*", h.getMedia)
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

func (h *Handler) postAvatar(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, KindAvatar)
}

func (h *Handler) postCover(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, KindCover)
}

func (h *Handler) deleteAvatar(w http.ResponseWriter, r *http.Request) {
	h.deleteMedia(w, r, KindAvatar)
}

func (h *Handler) deleteCover(w http.ResponseWriter, r *http.Request) {
	h.deleteMedia(w, r, KindCover)
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request, kind string) {
	_, maxBytes, _ := kindLimits(kind)
	raw, err := readUploadFile(w, r, maxBytes)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	out, err := h.svc.UploadImage(r.Context(), sessionUserID(r), httpx.ClientIP(r), kind, raw)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeMe(w, out)
}

func (h *Handler) deleteMedia(w http.ResponseWriter, r *http.Request, kind string) {
	out, err := h.svc.DeleteImage(r.Context(), sessionUserID(r), httpx.ClientIP(r), kind)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeMe(w, out)
}

func (h *Handler) getMedia(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	rc, ct, err := h.svc.OpenObject(r.Context(), key)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	defer rc.Close()
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if rs, ok := rc.(io.ReadSeeker); ok {
		http.ServeContent(w, r, "image.jpg", time.Time{}, rs)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}

func readUploadFile(w http.ResponseWriter, r *http.Request, maxBytes int) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes)+multipartSlack)
	if err := r.ParseMultipartForm(int64(maxBytes) + multipartSlack); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, ErrTooLarge
		}
		return nil, ValidationError{Msg: "multipart body is invalid"}
	}
	form := r.MultipartForm
	if form == nil {
		return nil, ValidationError{Msg: "file is required"}
	}
	for name := range form.File {
		if name != "file" {
			return nil, ValidationError{Msg: "unexpected file field"}
		}
	}
	files := form.File["file"]
	if len(files) != 1 {
		return nil, ValidationError{Msg: "file is required"}
	}
	f, err := files[0].Open()
	if err != nil {
		return nil, ValidationError{Msg: "file is required"}
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, ErrTooLarge
		}
		return nil, ValidationError{Msg: "file is required"}
	}
	if len(raw) > maxBytes {
		return nil, ErrTooLarge
	}
	return raw, nil
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
	case errors.Is(err, ErrInvalidImage):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_image", "Image is not a supported JPEG, PNG, or WebP")
	case errors.Is(err, ErrTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "File is too large")
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
