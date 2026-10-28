package checks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExecutorHTTPOutcomes(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		min, max int
		want     bool
	}{
		{name: "expected", status: 204, min: 200, max: 399, want: true},
		{name: "unexpected", status: 500, min: 200, max: 399, want: false},
		{name: "custom", status: 500, min: 500, max: 500, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tt.status) }))
			defer server.Close()
			result := NewExecutor().Check(context.Background(), Target{URL: server.URL, TimeoutMS: 1000, ExpectedStatusMin: tt.min, ExpectedStatusMax: tt.max})
			if result.Success != tt.want || result.HTTPStatus == nil || *result.HTTPStatus != tt.status || result.LatencyMS < 0 {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestExecutorTimeoutAndCancellation(t *testing.T) {
	t.Run("whole check including body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		}))
		defer server.Close()
		result := NewExecutor().Check(context.Background(), Target{URL: server.URL, TimeoutMS: 20, ExpectedStatusMin: 200, ExpectedStatusMax: 399})
		if result.Success || result.ErrorSummary != "request timeout" {
			t.Fatalf("result = %#v", result)
		}
	})

	t.Run("caller cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result := NewExecutor().Check(ctx, Target{URL: "http://example.com", TimeoutMS: 1000, ExpectedStatusMin: 200, ExpectedStatusMax: 399})
		if result.Success || result.ErrorSummary != "request canceled" {
			t.Fatalf("result = %#v", result)
		}
	})
}
