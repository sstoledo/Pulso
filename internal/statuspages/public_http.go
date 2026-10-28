package statuspages

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"pulso/internal/platform/httpx"
)

type PublicHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewPublicHTTPHandler(service *Service, logger *slog.Logger) *chi.Mux {
	handler := &PublicHandler{service: service, logger: logger}
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age=15")
			next.ServeHTTP(w, r)
		})
	})
	router.Get("/status/{slug}", handler.get)
	return router
}

func (h *PublicHandler) get(w http.ResponseWriter, r *http.Request) {
	page, err := h.service.Public(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, ErrPageNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "status_page_not_found", "Status page not found.")
		return
	}
	if err != nil {
		h.logger.ErrorContext(r.Context(), "public status page request failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}
