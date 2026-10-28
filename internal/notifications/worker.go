package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/store/sqlc"
)

type Message struct {
	To      string
	Subject string
	Text    string
}

type Mailer interface {
	Send(context.Context, Message) error
}

type Worker struct {
	pool          *pgxpool.Pool
	queries       *sqlc.Queries
	mailer        Mailer
	logger        *slog.Logger
	BatchSize     int32
	PollInterval  time.Duration
	LeaseDuration time.Duration
	Backoff       func(int) time.Duration
}

func NewWorker(pool *pgxpool.Pool, mailer Mailer, logger *slog.Logger) *Worker {
	return &Worker{
		pool: pool, queries: sqlc.New(pool), mailer: mailer, logger: logger,
		BatchSize: 20, PollInterval: time.Second, LeaseDuration: 2 * time.Minute, Backoff: ExponentialBackoff,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.PollInterval)
	defer ticker.Stop()
	for {
		if err := w.Dispatch(ctx); err != nil && ctx.Err() == nil {
			w.logger.Error("dispatch notifications", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (w *Worker) Dispatch(ctx context.Context) error {
	staleBefore := time.Now().UTC().Add(-w.LeaseDuration)
	if _, err := w.queries.RecoverStaleNotificationJobs(ctx, pgtype.Timestamptz{Time: staleBefore, Valid: true}); err != nil {
		return fmt.Errorf("recover stale notification jobs: %w", err)
	}
	jobs, err := w.queries.ClaimNotificationJobs(ctx, sqlc.ClaimNotificationJobsParams{
		Limit: w.BatchSize, StaleBefore: pgtype.Timestamptz{Time: staleBefore, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("claim notification jobs: %w", err)
	}
	for _, job := range jobs {
		if err := w.process(ctx, job); err != nil {
			return fmt.Errorf("process notification %s: %w", job.ID.String(), err)
		}
	}
	return nil
}

func (w *Worker) process(ctx context.Context, job sqlc.ClaimNotificationJobsRow) error {
	message := Message{To: job.RecipientEmail, Subject: subject(job), Text: text(job)}
	sendErr := w.mailer.Send(ctx, message)
	if sendErr == nil {
		if _, err := w.queries.InsertNotificationAttempt(ctx, sqlc.InsertNotificationAttemptParams{JobID: job.ID}); err != nil {
			return fmt.Errorf("record successful attempt: %w", err)
		}
		if _, err := w.queries.MarkNotificationSent(ctx, job.ID); err != nil {
			return fmt.Errorf("mark notification sent: %w", err)
		}
		return nil
	}

	failure := pgtype.Text{String: "smtp send failed", Valid: true}
	if _, err := w.queries.InsertNotificationAttempt(ctx, sqlc.InsertNotificationAttemptParams{JobID: job.ID, Error: failure}); err != nil {
		return fmt.Errorf("record failed attempt: %w", err)
	}
	if job.Attempts >= job.MaxAttempts {
		if _, err := w.queries.MarkNotificationFailed(ctx, sqlc.MarkNotificationFailedParams{ID: job.ID, LastError: failure}); err != nil {
			return fmt.Errorf("mark notification failed: %w", err)
		}
		return nil
	}
	nextAttempt := time.Now().UTC().Add(w.Backoff(int(job.Attempts)))
	if _, err := w.queries.ScheduleNotificationRetry(ctx, sqlc.ScheduleNotificationRetryParams{
		ID: job.ID, NextAttemptAt: pgtype.Timestamptz{Time: nextAttempt, Valid: true}, LastError: failure,
	}); err != nil {
		return fmt.Errorf("schedule notification retry: %w", err)
	}
	return nil
}

func ExponentialBackoff(attempt int) time.Duration {
	seconds := 1 << max(0, attempt-1)
	if seconds > 60 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

func subject(job sqlc.ClaimNotificationJobsRow) string {
	if job.Kind == "RECOVERED" {
		return fmt.Sprintf("[Pulso] %s recovered", job.MonitorName)
	}
	return fmt.Sprintf("[Pulso] %s is DOWN", job.MonitorName)
}

func text(job sqlc.ClaimNotificationJobsRow) string {
	if job.Kind == "RECOVERED" {
		return fmt.Sprintf("Monitor %q recovered at %s.\nOrganization: %s\n", job.MonitorName, job.ResolvedAt.Time.Format(time.RFC3339), job.OrganizationName)
	}
	return fmt.Sprintf("Monitor %q is DOWN.\nReason: %s\nOpened at: %s\nOrganization: %s\n", job.MonitorName, job.Reason, job.OpenedAt.Time.Format(time.RFC3339), job.OrganizationName)
}
