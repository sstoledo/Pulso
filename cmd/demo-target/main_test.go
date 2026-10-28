package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDemoTargetHealthCanBeControlled(t *testing.T) {
	handler := newDemoHandler()

	healthy := httptest.NewRecorder()
	handler.ServeHTTP(healthy, httptest.NewRequest(http.MethodGet, "/health", nil))
	if healthy.Code != http.StatusOK {
		t.Fatalf("initial health = %d", healthy.Code)
	}

	control := httptest.NewRecorder()
	handler.ServeHTTP(control, httptest.NewRequest(http.MethodPost, "/control", strings.NewReader(`{"healthy":false}`)))
	if control.Code != http.StatusOK {
		t.Fatalf("control = %d", control.Code)
	}
	down := httptest.NewRecorder()
	handler.ServeHTTP(down, httptest.NewRequest(http.MethodGet, "/health", nil))
	if down.Code != http.StatusInternalServerError {
		t.Fatalf("down health = %d", down.Code)
	}
}
