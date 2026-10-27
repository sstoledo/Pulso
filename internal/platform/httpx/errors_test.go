package httpx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"pulso/internal/platform/httpx"
)

func TestRouterErrorsUseConsistentEnvelope(t *testing.T) {
	router := httpx.NewRouter(readinessFunc(func(context.Context) error { return nil }), testLogger())

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing", nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
	want := "{\"error\":{\"code\":\"not_found\",\"message\":\"Resource not found.\"}}\n"
	if response.Body.String() != want {
		t.Fatalf("body = %q, want %q", response.Body.String(), want)
	}
}

func TestWriteErrorIncludesValidationDetails(t *testing.T) {
	response := httptest.NewRecorder()
	httpx.WriteError(response, http.StatusUnprocessableEntity, "validation_failed", "Validation failed.", map[string]string{"email": "Enter a valid email address."})

	want := "{\"error\":{\"code\":\"validation_failed\",\"message\":\"Validation failed.\",\"details\":{\"email\":\"Enter a valid email address.\"}}}\n"
	if response.Body.String() != want {
		t.Fatalf("body = %q, want %q", response.Body.String(), want)
	}
}
