package checks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/store/sqlc"
)

type CheckExecutor interface {
	Check(context.Context, Target) Result
}

type Worker struct {
	*Scheduler
	executor CheckExecutor
	logger   *slog.Logger
}

func NewWorker(pool *pgxpool.Pool, executor CheckExecutor, logger *slog.Logger) *Worker {
	return &Worker{Scheduler: NewScheduler(pool, logger), executor: executor, logger: logger}
}

func (w *Worker) ScheduleAndProcess(ctx context.Context) error {
	if _, err := w.Schedule(ctx); err != nil {
		return err
	}
	jobs, err := w.Claim(ctx, w.BatchSize)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if err := w.Process(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) Process(ctx context.Context, job ClaimedJob) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	result := w.executor.Check(ctx, Target{
		URL: job.Url, TimeoutMS: int(job.TimeoutMs),
		ExpectedStatusMin: int(job.ExpectedStatusMin), ExpectedStatusMax: int(job.ExpectedStatusMax),
	})
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin check result: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := w.queries.WithTx(tx)
	monitor, err := queries.LockMonitorByID(ctx, job.MonitorID)
	if err != nil {
		return fmt.Errorf("lock monitor: %w", err)
	}
	checkedAt := time.Now().UTC()
	httpStatus := pgtype.Int4{}
	if result.HTTPStatus != nil {
		httpStatus = pgtype.Int4{Int32: int32(*result.HTTPStatus), Valid: true}
	}
	_, err = queries.InsertCheckResult(ctx, sqlc.InsertCheckResultParams{
		CheckJobID: job.ID, MonitorID: job.MonitorID,
		CheckedAt: pgtype.Timestamptz{Time: checkedAt, Valid: true}, Success: result.Success,
		HttpStatus: httpStatus, LatencyMs: int32(result.LatencyMS),
		ErrorSummary: pgtype.Text{String: result.ErrorSummary, Valid: result.ErrorSummary != ""},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("insert check result: %w", err)
	}
	status, failures := ApplyResult(monitor.Status, int(monitor.ConsecutiveFailures), result.Success)
	if _, err := queries.UpdateMonitorAfterCheck(ctx, sqlc.UpdateMonitorAfterCheckParams{
		ID: monitor.ID, Status: status, ConsecutiveFailures: int32(failures),
		LastCheckAt:   pgtype.Timestamptz{Time: checkedAt, Valid: true},
		LastLatencyMs: pgtype.Int4{Int32: int32(result.LatencyMS), Valid: true},
		NextCheckAt:   pgtype.Timestamptz{Time: job.ScheduledFor.Time.Add(time.Duration(monitor.IntervalSeconds) * time.Second), Valid: true},
	}); err != nil {
		return fmt.Errorf("update monitor after check: %w", err)
	}
	if _, err := queries.CompleteCheckJob(ctx, job.ID); err != nil {
		return fmt.Errorf("complete check job: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit check result: %w", err)
	}
	return nil
}

func ApplyResult(currentStatus string, consecutiveFailures int, success bool) (string, int) {
	if success {
		return "UP", 0
	}
	consecutiveFailures++
	if consecutiveFailures >= 2 {
		return "DOWN", consecutiveFailures
	}
	return currentStatus, consecutiveFailures
}
