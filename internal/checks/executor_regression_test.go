package checks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExecutorBlocksAdditionalInternalRanges(t *testing.T) {
	for _, target := range []string{"http://100.64.0.1", "http://198.18.0.1", "http://240.0.0.1", "http://[::ffff:100.64.0.1]"} {
		t.Run(target, func(t *testing.T) {
			_, err := NewExecutor(nil).destination(context.Background(), target)
			if !errors.Is(err, ErrDestinationBlocked) {
				t.Fatalf("destination error = %v, want ErrDestinationBlocked", err)
			}
		})
	}
}

func TestExecutorTimeoutAfterResponseHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()

	result := NewExecutor([]string{"127.0.0.1"}).Check(context.Background(), Target{
		URL: server.URL, TimeoutMS: 20, ExpectedStatusMin: 200, ExpectedStatusMax: 399,
	})
	if result.Success || result.ErrorSummary != "request timeout" {
		t.Fatalf("whole-check timeout after headers = %#v", result)
	}
}
