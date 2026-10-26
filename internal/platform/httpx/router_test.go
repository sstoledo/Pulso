package httpx_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pulso/internal/platform/httpx"
)

type readinessFunc func(context.Context) error

func (f readinessFunc) Check(ctx context.Context) error {
	return f(ctx)
}

func TestHealthzDoesNotCheckDependencies(t *testing.T) {
	called := false
	router := httpx.NewRouter(readinessFunc(func(context.Context) error {
		called = true
		return errors.New("database unavailable")
	}), testLogger())

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if called {
		t.Fatal("healthz must not check dependencies")
	}
	if response.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("body = %q", response.Body.String())
	}
}

func TestReadyzReportsDatabaseReadiness(t *testing.T) {
	tests := []struct {
		name       string
		checkError error
		wantStatus int
		wantBody   string
	}{
		{name: "ready", wantStatus: http.StatusOK, wantBody: "{\"status\":\"ok\"}\n"},
		{name: "database unavailable", checkError: errors.New("postgres secret details"), wantStatus: http.StatusServiceUnavailable, wantBody: "{\"status\":\"unavailable\"}\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := httpx.NewRouter(readinessFunc(func(context.Context) error {
				return tt.checkError
			}), testLogger())

			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if response.Body.String() != tt.wantBody {
				t.Fatalf("body = %q, want %q", response.Body.String(), tt.wantBody)
			}
			if strings.Contains(response.Body.String(), "secret") {
				t.Fatal("readiness response leaked internal error details")
			}
			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q", got)
			}
		})
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
