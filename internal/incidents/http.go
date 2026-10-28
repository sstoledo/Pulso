package incidents

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"pulso/internal/auth"
	"pulso/internal/platform/httpx"
)

type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewHTTPHandler(service *Service, authService *auth.Service, logger *slog.Logger) *chi.Mux {
	handler := &HTTPHandler{service: service, logger: logger}
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			next.ServeHTTP(w, r)
		})
	})
	router.With(auth.RequireAuthentication(authService, logger), auth.RequireOrganization(authService, logger)).Get("/incidents", handler.list)
	return router
}

func (h *HTTPHandler) list(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 10)
	result, err := h.service.List(r.Context(), auth.OrganizationFromContext(r.Context()).ID, r.URL.Query().Get("monitor_id"), page, pageSize)
	if err != nil {
		var validation *ValidationError
		if errors.As(err, &validation) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "Validation failed.", validation.Fields)
			return
		}
		h.logger.ErrorContext(r.Context(), "list incidents failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}
