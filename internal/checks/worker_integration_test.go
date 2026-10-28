package checks

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/auth"
	"pulso/internal/monitors"
	"pulso/internal/store/sqlc"
)

func TestWorkerConcurrentClaimAndIdempotentResult(t *testing.T) {
	pool, _, monitor := newWorkerMonitor(t, http.StatusOK)
	worker := NewWorker(pool, NewExecutor([]string{"127.0.0.1"}), discardWorkerLogger())
	ctx := context.Background()

	if rows, _ := worker.Schedule(ctx); rows != 1 {
		t.Fatalf("scheduled rows = %d", rows)
	}
	if rows, _ := worker.Schedule(ctx); rows != 0 {
		t.Fatalf("duplicate scheduled rows = %d", rows)
	}
	claimed := concurrentClaims(t, worker, 8)
	if len(claimed) != 1 {
		t.Fatalf("claimed jobs = %d, want 1", len(claimed))
	}

	var wg sync.WaitGroup
	for index := 0; index < 6; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := worker.Process(ctx, claimed[0]); err != nil {
				t.Errorf("Process() error = %v", err)
			}
		}()
	}
	wg.Wait()

	var results int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM check_results WHERE check_job_id = $1::uuid", claimed[0].ID.String()).Scan(&results); err != nil {
		t.Fatal(err)
	}
	if results != 1 {
		t.Fatalf("results = %d, want 1", results)
	}
	var status string
	if err := pool.QueryRow(ctx, "SELECT status FROM monitors WHERE id = $1::uuid", monitor.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "UP" {
		t.Fatalf("monitor status = %s", status)
	}
}

func TestWorkerFailureAndRecoveryTransitions(t *testing.T) {
	pool, _, monitor := newWorkerMonitor(t, http.StatusInternalServerError)
	worker := NewWorker(pool, NewExecutor([]string{"127.0.0.1"}), discardWorkerLogger())
	ctx := context.Background()

	worker.ScheduleAndProcess(ctx)
	assertMonitorState(t, pool, monitor.ID, "UNKNOWN", 1)
	if _, err := pool.Exec(ctx, "UPDATE monitors SET next_check_at = now() - interval '1 second' WHERE id = $1::uuid", monitor.ID); err != nil {
		t.Fatal(err)
	}
	worker.ScheduleAndProcess(ctx)
	assertMonitorState(t, pool, monitor.ID, "DOWN", 2)

	if _, err := pool.Exec(ctx, "UPDATE monitors SET url = $1, next_check_at = now() - interval '1 second' WHERE id = $2::uuid", healthyServer(t).URL, monitor.ID); err != nil {
		t.Fatal(err)
	}
	worker.ScheduleAndProcess(ctx)
	assertMonitorState(t, pool, monitor.ID, "UP", 0)
}

func newWorkerMonitor(t *testing.T, status int) (*pgxpool.Pool, auth.Identity, monitors.Monitor) {
	t.Helper()
	databaseURL := os.Getenv("PULSO_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("PULSO_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	authService, err := auth.NewService(pool)
	if err != nil {
		t.Fatal(err)
	}
	owner, _, err := authService.Register(context.Background(), auth.RegisterInput{
		Name: "Worker Owner", Email: fmt.Sprintf("worker-%d@example.com", time.Now().UnixNano()), Password: "password-12345",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, organization := range owner.Organizations {
			_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1::uuid", organization.ID)
		}
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1::uuid", owner.User.ID)
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
	t.Cleanup(server.Close)
	monitor, err := monitors.NewService(pool).Create(context.Background(), owner.Organizations[0].ID, monitors.CreateInput{Name: "Test API", URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	return pool, owner, monitor
}

func concurrentClaims(t *testing.T, worker *Worker, count int) []sqlc.ClaimCheckJobsRow {
	t.Helper()
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var claimed []sqlc.ClaimCheckJobsRow
	for index := 0; index < count; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			jobs, err := worker.Claim(context.Background(), 10)
			if err != nil {
				t.Errorf("Claim() error = %v", err)
				return
			}
			mutex.Lock()
			claimed = append(claimed, jobs...)
			mutex.Unlock()
		}()
	}
	wg.Wait()
	return claimed
}

func assertMonitorState(t *testing.T, pool *pgxpool.Pool, monitorID, wantStatus string, wantFailures int) {
	t.Helper()
	var status string
	var failures int
	if err := pool.QueryRow(context.Background(), "SELECT status, consecutive_failures FROM monitors WHERE id = $1::uuid", monitorID).Scan(&status, &failures); err != nil {
		t.Fatal(err)
	}
	if status != wantStatus || failures != wantFailures {
		t.Fatalf("state = %s/%d, want %s/%d", status, failures, wantStatus, wantFailures)
	}
}

func healthyServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(server.Close)
	return server
}

func discardWorkerLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
