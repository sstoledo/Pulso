package notifications

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/auth"
	"pulso/internal/monitors"
	"pulso/internal/store/sqlc"
)

type fakeMailer struct {
	calls []Message
	fail  int
}

func (f *fakeMailer) Send(_ context.Context, message Message) error {
	f.calls = append(f.calls, message)
	if f.fail != 0 {
		if f.fail < 0 || len(f.calls) <= f.fail {
			return errors.New("fake SMTP failure")
		}
	}
	return nil
}

func TestNotificationSuccessIsIdempotent(t *testing.T) {
	pool, _, mailer := newNotificationFixture(t)
	worker := NewWorker(pool, mailer, discardNotificationLogger())

	if err := worker.Dispatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := worker.Dispatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(mailer.calls) != 1 {
		t.Fatalf("mail calls = %d, want 1", len(mailer.calls))
	}
	assertNotificationState(t, pool, "sent", 1, 1)
}

func TestNotificationRetriesUntilSuccess(t *testing.T) {
	pool, _, mailer := newNotificationFixture(t)
	mailer.fail = 2
	worker := NewWorker(pool, mailer, discardNotificationLogger())
	worker.Backoff = func(int) time.Duration { return 0 }

	for attempt := 0; attempt < 3; attempt++ {
		if err := worker.Dispatch(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(mailer.calls) != 3 {
		t.Fatalf("mail calls = %d, want 3", len(mailer.calls))
	}
	assertNotificationState(t, pool, "sent", 3, 3)
}

func TestNotificationStopsAfterMaximumAttempts(t *testing.T) {
	pool, _, mailer := newNotificationFixture(t)
	mailer.fail = -1
	worker := NewWorker(pool, mailer, discardNotificationLogger())
	worker.Backoff = func(int) time.Duration { return 0 }

	for attempt := 0; attempt < 6; attempt++ {
		if err := worker.Dispatch(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(mailer.calls) != 5 {
		t.Fatalf("mail calls = %d, want 5", len(mailer.calls))
	}
	assertNotificationState(t, pool, "failed", 5, 5)
}

type blockingMailer struct {
	started chan struct{}
	release chan struct{}
}

func (m *blockingMailer) Send(context.Context, Message) error {
	m.started <- struct{}{}
	<-m.release
	return nil
}

func TestNotificationConcurrentDispatchClaimsOnce(t *testing.T) {
	pool, _, _ := newNotificationFixture(t)
	mailer := &blockingMailer{started: make(chan struct{}, 2), release: make(chan struct{})}
	logger := discardNotificationLogger()
	first := NewWorker(pool, mailer, logger)
	second := NewWorker(pool, mailer, logger)
	errs := make(chan error, 2)

	go func() { errs <- first.Dispatch(context.Background()) }()
	select {
	case <-mailer.started:
	case <-time.After(3 * time.Second):
		t.Fatal("first send did not start")
	}
	go func() { errs <- second.Dispatch(context.Background()) }()
	select {
	case <-mailer.started:
		t.Error("second worker sent a leased notification")
	case <-time.After(250 * time.Millisecond):
	}
	close(mailer.release)
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	assertNotificationState(t, pool, "sent", 1, 1)
}

func newNotificationFixture(t *testing.T) (*pgxpool.Pool, auth.Identity, *fakeMailer) {
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
		Name: "Notification Owner", Email: fmt.Sprintf("notification-%d@example.com", time.Now().UnixNano()), Password: "password-12345",
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
	monitor, err := monitors.NewService(pool).Create(context.Background(), owner.Organizations[0].ID, monitors.CreateInput{Name: "API", URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	incident, err := sqlc.New(pool).InsertIncident(context.Background(), sqlc.InsertIncidentParams{MonitorID: uuidFromString(t, monitor.ID), Reason: "unexpected HTTP status"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlc.New(pool).InsertNotificationJob(context.Background(), sqlc.InsertNotificationJobParams{
		IncidentID: incident.ID, Kind: "OPENED", IdempotencyKey: "incident:" + incident.ID.String() + ":opened",
	}); err != nil {
		t.Fatal(err)
	}
	return pool, owner, &fakeMailer{}
}

func uuidFromString(t *testing.T, value string) pgtype.UUID {
	t.Helper()
	parsed, err := uuid.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}
}

func assertNotificationState(t *testing.T, pool *pgxpool.Pool, status string, attempts, history int) {
	t.Helper()
	var gotStatus string
	var gotAttempts, gotHistory int
	if err := pool.QueryRow(context.Background(), "SELECT status, attempts FROM notification_jobs LIMIT 1").Scan(&gotStatus, &gotAttempts); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM notification_attempts").Scan(&gotHistory); err != nil {
		t.Fatal(err)
	}
	if gotStatus != status || gotAttempts != attempts || gotHistory != history {
		t.Fatalf("state = %s/%d/%d, want %s/%d/%d", gotStatus, gotAttempts, gotHistory, status, attempts, history)
	}
}

func discardNotificationLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
