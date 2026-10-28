-- name: ScheduleDueMonitors :execrows
INSERT INTO check_jobs (monitor_id, scheduled_for)
SELECT id, next_check_at
FROM monitors
WHERE enabled
  AND next_check_at IS NOT NULL
  AND next_check_at <= now()
ON CONFLICT DO NOTHING;

-- name: ClaimCheckJobs :many
WITH candidates AS (
    SELECT j.id
    FROM check_jobs AS j
    JOIN monitors AS m ON m.id = j.monitor_id
    WHERE j.status = 'pending'
      AND j.available_at <= now()
      AND m.enabled
    ORDER BY j.scheduled_for, j.id
    FOR UPDATE OF j SKIP LOCKED
    LIMIT $1
)
UPDATE check_jobs AS j
SET status = 'running', locked_at = now(), attempts = j.attempts + 1
FROM candidates, monitors AS m
WHERE j.id = candidates.id
  AND m.id = j.monitor_id
RETURNING j.id, j.monitor_id, j.scheduled_for, j.attempts,
          m.name, m.url, m.interval_seconds, m.timeout_ms,
          m.expected_status_min, m.expected_status_max;

-- name: RecoverStaleCheckJobs :execrows
UPDATE check_jobs
SET status = 'pending', locked_at = NULL
WHERE status = 'running'
  AND locked_at < $1;

-- name: DeletePendingCheckJobsForMonitor :exec
DELETE FROM check_jobs
WHERE monitor_id = $1
  AND status = 'pending';

-- name: InsertCheckResult :one
INSERT INTO check_results (
    check_job_id, monitor_id, checked_at, success, http_status, latency_ms, error_summary
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (check_job_id) DO NOTHING
RETURNING *;

-- name: CompleteCheckJob :execrows
UPDATE check_jobs
SET status = 'completed', completed_at = now()
WHERE id = $1
  AND status = 'running';

-- name: LockMonitorByID :one
SELECT *
FROM monitors
WHERE id = $1
FOR UPDATE;

-- name: UpdateMonitorAfterCheck :one
UPDATE monitors
SET status = $2,
    consecutive_failures = $3,
    last_check_at = $4,
    last_latency_ms = $5,
    next_check_at = CASE WHEN enabled THEN sqlc.arg(next_check_at)::timestamptz ELSE NULL END,
    updated_at = now()
WHERE id = $1
RETURNING *;
