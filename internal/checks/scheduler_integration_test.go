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
)

func TestSchedulerClaimsOnceConcurrently(t *testing.T) {
	pool, _, _ := newSchedulerMonitor(t)
	scheduler := NewScheduler(pool, discardSchedulerLogger())
	ctx := context.Background()
	if rows, _ := scheduler.Schedule(ctx); rows != 1 {
		t.Fatalf("scheduled = %d", rows)
	}
	if rows, _ := scheduler.Schedule(ctx); rows != 0 {
		t.Fatalf("duplicate scheduled = %d", rows)
	}
	claimed := concurrentSchedulerClaims(t, scheduler, 8)
	if len(claimed) != 1 || claimed[0].Attempts != 1 {
		t.Fatalf("claimed = %#v", claimed)
	}
}

func TestSchedulerRecoversStaleJobs(t *testing.T) {
	pool, _, _ := newSchedulerMonitor(t)
	scheduler := NewScheduler(pool, discardSchedulerLogger())
	ctx := context.Background()
	scheduler.Schedule(ctx)
	claimed := concurrentSchedulerClaims(t, scheduler, 1)
	if _, err := pool.Exec(ctx, "UPDATE check_jobs SET locked_at = now() - interval '3 minutes' WHERE id = $1::uuid", claimed[0].ID.String()); err != nil {
		t.Fatal(err)
	}
	if rows, err := scheduler.Recover(ctx); err != nil || rows != 1 {
		t.Fatalf("Recover() rows = %d, err = %v", rows, err)
	}
	reclaimed := concurrentSchedulerClaims(t, scheduler, 1)
	if len(reclaimed) != 1 || reclaimed[0].ID != claimed[0].ID || reclaimed[0].Attempts != 2 {
		t.Fatalf("reclaimed = %#v", reclaimed)
	}
}

func TestPauseCancelsPendingJobs(t *testing.T) {
	pool, owner, monitor := newSchedulerMonitor(t)
	scheduler := NewScheduler(pool, discardSchedulerLogger())
	ctx := context.Background()
	scheduler.Schedule(ctx)
	if _, err := monitors.NewService(pool).SetEnabled(ctx, owner.Organizations[0].ID, monitor.ID, false); err != nil {
		t.Fatal(err)
	}
	var pending int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM check_jobs WHERE monitor_id = $1::uuid AND status = 'pending'", monitor.ID).Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if pending != 0 {
		t.Fatalf("pending after pause = %d", pending)
	}
	if claimed := concurrentSchedulerClaims(t, scheduler, 2); len(claimed) != 0 {
		t.Fatalf("claimed paused jobs = %d", len(claimed))
	}
}

func newSchedulerMonitor(t *testing.T) (*pgxpool.Pool, auth.Identity, monitors.Monitor) {
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
		Name: "Scheduler Owner", Email: fmt.Sprintf("scheduler-%d@example.com", time.Now().UnixNano()), Password: "password-12345",
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(server.Close)
	monitor, err := monitors.NewService(pool).Create(context.Background(), owner.Organizations[0].ID, monitors.CreateInput{Name: "Scheduler API", URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	return pool, owner, monitor
}

func concurrentSchedulerClaims(t *testing.T, scheduler *Scheduler, count int) []ClaimedJob {
	t.Helper()
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var claimed []ClaimedJob
	for index := 0; index < count; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			jobs, err := scheduler.Claim(context.Background(), 10)
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

func discardSchedulerLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
