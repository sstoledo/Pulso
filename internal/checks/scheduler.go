package checks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/store/sqlc"
)

type ClaimedJob = sqlc.ClaimCheckJobsRow

type Scheduler struct {
	pool          *pgxpool.Pool
	queries       *sqlc.Queries
	logger        *slog.Logger
	BatchSize     int32
	PollInterval  time.Duration
	LeaseDuration time.Duration
}

func NewScheduler(pool *pgxpool.Pool, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		pool: pool, queries: sqlc.New(pool), logger: logger,
		BatchSize: 20, PollInterval: time.Second, LeaseDuration: 2 * time.Minute,
	}
}

func (s *Scheduler) Schedule(ctx context.Context) (int64, error) {
	rows, err := s.queries.ScheduleDueMonitors(ctx)
	if err != nil {
		return 0, fmt.Errorf("schedule due monitors: %w", err)
	}
	return rows, nil
}

func (s *Scheduler) Claim(ctx context.Context, limit int32) ([]ClaimedJob, error) {
	jobs, err := s.queries.ClaimCheckJobs(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("claim check jobs: %w", err)
	}
	return jobs, nil
}

func (s *Scheduler) Recover(ctx context.Context) (int64, error) {
	threshold := time.Now().UTC().Add(-s.LeaseDuration)
	rows, err := s.queries.RecoverStaleCheckJobs(ctx, pgtype.Timestamptz{Time: threshold, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("recover stale jobs: %w", err)
	}
	return rows, nil
}
