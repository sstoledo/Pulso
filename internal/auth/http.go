package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"pulso/internal/platform/httpx"
)

const SessionCookieName = "pulso_session"

type HTTPHandler struct {
	service       *Service
	cookieSecure  bool
	allowedOrigin string
	logger        *slog.Logger
}

type sessionContext struct {
	identity Identity
	rawToken string
}

type sessionContextKey struct{}

func NewHTTPHandler(service *Service, cookieSecure bool, allowedOrigin string, logger *slog.Logger) *chi.Mux {
	handler := &HTTPHandler{service: service, cookieSecure: cookieSecure, allowedOrigin: allowedOrigin, logger: logger}
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			next.ServeHTTP(w, r)
		})
	})
	router.With(handler.requireSameOrigin).Post("/auth/register", handler.register)
	router.With(handler.requireSameOrigin).Post("/auth/login", handler.login)
	router.With(handler.requireSameOrigin, handler.requireAuthentication).Post("/auth/logout", handler.logout)
	router.With(handler.requireAuthentication).Get("/auth/me", handler.me)
	router.With(RequireAuthentication(service, logger), RequireOrganization(service, logger)).Get("/orgs/{organizationID}", handler.organization)
	return router
}

func (h *HTTPHandler) register(w http.ResponseWriter, r *http.Request) {
	var input RegisterInput
	if !httpx.DecodeJSON(w, r, &input) {
		return
	}
	identity, rawToken, err := h.service.Register(r.Context(), input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.setSessionCookie(w, rawToken)
	httpx.WriteJSON(w, http.StatusCreated, identity)
}

func (h *HTTPHandler) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !httpx.DecodeJSON(w, r, &input) {
		return
	}
	identity, rawToken, err := h.service.Login(r.Context(), input.Email, input.Password)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.setSessionCookie(w, rawToken)
	httpx.WriteJSON(w, http.StatusOK, identity)
}

func (h *HTTPHandler) logout(w http.ResponseWriter, r *http.Request) {
	session := sessionFromContext(r.Context())
	if err := h.service.Logout(r.Context(), session.rawToken); err != nil {
		h.writeError(w, r, err)
		return
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) me(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, sessionFromContext(r.Context()).identity)
}

func (h *HTTPHandler) organization(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, OrganizationFromContext(r.Context()))
}

func (h *HTTPHandler) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "Authentication required.")
			return
		}
		identity, err := h.service.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), sessionContextKey{}, sessionContext{identity: identity, rawToken: cookie.Value})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *HTTPHandler) requireSameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if (origin != "" && origin != h.allowedOrigin) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			httpx.WriteError(w, http.StatusForbidden, "origin_forbidden", "Request origin is not allowed.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validation *ValidationError
	switch {
	case errors.As(err, &validation):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "Validation failed.", validation.Fields)
	case errors.Is(err, ErrEmailAlreadyRegistered):
		httpx.WriteError(w, http.StatusConflict, "email_already_registered", "An account with this email already exists.")
	case errors.Is(err, ErrInvalidCredentials):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password.")
	case errors.Is(err, ErrUnauthorized):
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "Authentication required.")
	case errors.Is(err, ErrForbidden):
		httpx.WriteError(w, http.StatusForbidden, "organization_forbidden", "You do not belong to this organization.")
	default:
		h.logger.ErrorContext(r.Context(), "authentication request failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
	}
}

func (h *HTTPHandler) setSessionCookie(w http.ResponseWriter, rawToken string) {
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookieName, Value: rawToken, Path: "/",
		Expires: time.Now().UTC().Add(SessionTTL), MaxAge: int(SessionTTL.Seconds()),
		HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *HTTPHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookieName, Value: "", Path: "/", Expires: time.Unix(1, 0), MaxAge: -1,
		HttpOnly: true, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func sessionFromContext(ctx context.Context) sessionContext {
	return ctx.Value(sessionContextKey{}).(sessionContext)
}
