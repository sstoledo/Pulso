-- name: InsertNotificationJob :one
INSERT INTO notification_jobs (incident_id, kind, idempotency_key)
VALUES ($1, $2, $3)
ON CONFLICT (idempotency_key) DO NOTHING
RETURNING *;

-- name: ClaimNotificationJobs :many
WITH candidates AS (
    SELECT j.id
    FROM notification_jobs AS j
    WHERE j.status = 'pending'
      AND j.next_attempt_at <= now()
    ORDER BY j.next_attempt_at, j.id
    FOR UPDATE OF j SKIP LOCKED
    LIMIT $1
)
UPDATE notification_jobs AS j
SET locked_at = now(), attempts = j.attempts + 1
FROM candidates, incidents AS i, monitors AS m, organizations AS o, memberships AS ms, users AS u
WHERE j.id = candidates.id
  AND i.id = j.incident_id
  AND m.id = i.monitor_id
  AND o.id = m.organization_id
  AND ms.organization_id = o.id
  AND ms.role = 'owner'
  AND u.id = ms.user_id
RETURNING j.id, j.incident_id, j.kind, j.attempts, j.max_attempts,
          i.monitor_id, i.opened_at, i.resolved_at, i.reason,
          m.name AS monitor_name, o.name AS organization_name, u.email AS recipient_email;

-- name: InsertNotificationAttempt :one
INSERT INTO notification_attempts (job_id, error)
VALUES ($1, $2)
RETURNING *;

-- name: MarkNotificationSent :execrows
UPDATE notification_jobs
SET status = 'sent', sent_at = now(), locked_at = NULL, last_error = NULL
WHERE id = $1;

-- name: ScheduleNotificationRetry :execrows
UPDATE notification_jobs
SET next_attempt_at = $2, last_error = $3, locked_at = NULL
WHERE id = $1
  AND status = 'pending';

-- name: MarkNotificationFailed :execrows
UPDATE notification_jobs
SET status = 'failed', last_error = $2, locked_at = NULL
WHERE id = $1
  AND status = 'pending';

-- name: RecoverStaleNotificationJobs :execrows
UPDATE notification_jobs
SET locked_at = NULL
WHERE status = 'pending'
  AND locked_at < $1;
