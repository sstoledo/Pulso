package monitors

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"pulso/internal/auth"
	"pulso/internal/platform/httpx"
)

type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

type createRequest struct {
	Name              string `json:"name"`
	URL               string `json:"url"`
	IntervalSeconds   *int   `json:"interval_seconds"`
	TimeoutMS         *int   `json:"timeout_ms"`
	ExpectedStatusMin *int   `json:"expected_status_min"`
	ExpectedStatusMax *int   `json:"expected_status_max"`
	ShowPublic        *bool  `json:"show_public"`
}

func (input createRequest) toCreateInput() (CreateInput, error) {
	result := defaultCreateInput()
	result.Name = input.Name
	result.URL = input.URL
	if input.IntervalSeconds != nil {
		result.IntervalSeconds = *input.IntervalSeconds
	}
	if input.TimeoutMS != nil {
		result.TimeoutMS = *input.TimeoutMS
	}
	if input.ExpectedStatusMin != nil {
		result.ExpectedStatusMin = *input.ExpectedStatusMin
	}
	if input.ExpectedStatusMax != nil {
		result.ExpectedStatusMax = *input.ExpectedStatusMax
	}
	if input.ShowPublic != nil {
		result.ShowPublic = *input.ShowPublic
	}
	result.Name = strings.TrimSpace(result.Name)
	result.URL = strings.TrimSpace(result.URL)
	if err := validateConfiguration(result.Name, result.URL, result.IntervalSeconds, result.TimeoutMS, result.ExpectedStatusMin, result.ExpectedStatusMax); err != nil {
		return CreateInput{}, err
	}
	return result, nil
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
	readChain := []func(http.Handler) http.Handler{auth.RequireAuthentication(authService, logger), auth.RequireOrganization(authService, logger)}
	writeChain := []func(http.Handler) http.Handler{auth.RequireSameOrigin(allowedOrigin), auth.RequireAuthentication(authService, logger), auth.RequireOrganization(authService, logger)}
	router.With(readChain...).Get("/monitors", handler.list)
	router.With(readChain...).Get("/dashboard/summary", handler.summary)
	router.With(readChain...).Get("/monitors/{monitorID}", handler.get)
	router.With(writeChain...).Post("/monitors", handler.create)
	router.With(writeChain...).Patch("/monitors/{monitorID}", handler.update)
	router.With(writeChain...).Delete("/monitors/{monitorID}", handler.delete)
	router.With(writeChain...).Post("/monitors/{monitorID}/pause", handler.pause)
	router.With(writeChain...).Post("/monitors/{monitorID}/resume", handler.resume)
	return router
}

func (h *HTTPHandler) create(w http.ResponseWriter, r *http.Request) {
	var request createRequest
	if !httpx.DecodeJSON(w, r, &request) {
		return
	}
	input, err := request.toCreateInput()
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	monitor, err := h.service.Create(r.Context(), organizationID(r), input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, monitor)
}

func (h *HTTPHandler) list(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 10)
	result, err := h.service.List(r.Context(), organizationID(r), page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *HTTPHandler) summary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.Summary(r.Context(), organizationID(r))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, summary)
}

func (h *HTTPHandler) get(w http.ResponseWriter, r *http.Request) {
	monitor, err := h.service.Get(r.Context(), organizationID(r), chi.URLParam(r, "monitorID"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, monitor)
}

func (h *HTTPHandler) update(w http.ResponseWriter, r *http.Request) {
	var input UpdateInput
	if !httpx.DecodeJSON(w, r, &input) {
		return
	}
	monitor, err := h.service.Update(r.Context(), organizationID(r), chi.URLParam(r, "monitorID"), input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, monitor)
}

func (h *HTTPHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), organizationID(r), chi.URLParam(r, "monitorID")); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) pause(w http.ResponseWriter, r *http.Request) {
	h.setEnabled(w, r, false)
}

func (h *HTTPHandler) resume(w http.ResponseWriter, r *http.Request) {
	h.setEnabled(w, r, true)
}

func (h *HTTPHandler) setEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	monitor, err := h.service.SetEnabled(r.Context(), organizationID(r), chi.URLParam(r, "monitorID"), enabled)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, monitor)
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validation *ValidationError
	switch {
	case errors.As(err, &validation):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "Validation failed.", validation.Fields)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "monitor_not_found", "Monitor not found.")
	default:
		h.logger.ErrorContext(r.Context(), "monitor request failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
	}
}

func organizationID(r *http.Request) string {
	return auth.OrganizationFromContext(r.Context()).ID
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
