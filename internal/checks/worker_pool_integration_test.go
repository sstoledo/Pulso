package checks

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"pulso/internal/monitors"
)

func TestWorkerPoolBoundsConcurrency(t *testing.T) {
	pool, owner, _ := newWorkerMonitor(t, http.StatusOK)
	ctx := context.Background()
	service := monitors.NewService(pool)
	var current, maximum, finished int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		active := atomic.AddInt32(&current, 1)
		for {
			previous := atomic.LoadInt32(&maximum)
			if active <= previous || atomic.CompareAndSwapInt32(&maximum, previous, active) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&current, -1)
		atomic.AddInt32(&finished, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	for index := 0; index < 9; index++ {
		if _, err := service.Create(ctx, owner.Organizations[0].ID, monitors.CreateInput{Name: fmt.Sprintf("API %d", index), URL: server.URL}); err != nil {
			t.Fatal(err)
		}
	}
	worker := NewWorker(pool, NewExecutor([]string{"127.0.0.1"}), discardWorkerLogger())
	worker.Concurrency = 2
	worker.PollInterval = 5 * time.Millisecond
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- worker.Run(runCtx) }()
	for atomic.LoadInt32(&finished) < 9 {
		select {
		case <-time.After(3 * time.Second):
			t.Fatalf("completed checks = %d, want 9", atomic.LoadInt32(&finished))
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if maximum > 2 {
		t.Fatalf("maximum concurrency = %d", maximum)
	}
}
