package statuspages

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"pulso/internal/auth"
	"pulso/internal/platform/httpx"
)

type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewHTTPHandler(service *Service, authService *auth.Service, allowedOrigin string, logger *slog.Logger) *chi.Mux {
	handler := &HTTPHandler{service: service, logger: logger}
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			next.ServeHTTP(w, r)
		})
	})
	router.With(auth.RequireAuthentication(authService, logger), auth.RequireOrganization(authService, logger)).Get("/", handler.get)
	router.With(auth.RequireSameOrigin(allowedOrigin), auth.RequireAuthentication(authService, logger), auth.RequireOrganization(authService, logger)).Patch("/", handler.update)
	return router
}

func (h *HTTPHandler) get(w http.ResponseWriter, r *http.Request) {
	config, err := h.service.Get(r.Context(), auth.OrganizationFromContext(r.Context()).ID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, config)
}

func (h *HTTPHandler) update(w http.ResponseWriter, r *http.Request) {
	var input UpdateInput
	if !httpx.DecodeJSON(w, r, &input) {
		return
	}
	config, err := h.service.Update(r.Context(), auth.OrganizationFromContext(r.Context()).ID, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, config)
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validation *ValidationError
	switch {
	case errors.As(err, &validation):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "Validation failed.", validation.Fields)
	case errors.Is(err, ErrSlugConflict):
		httpx.WriteError(w, http.StatusConflict, "slug_already_exists", "This status page slug is already in use.")
	default:
		h.logger.ErrorContext(r.Context(), "status page request failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
	}
}
