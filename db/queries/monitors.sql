-- name: CreateMonitor :one
INSERT INTO monitors (
    organization_id, name, url, interval_seconds, timeout_ms,
    expected_status_min, expected_status_max, show_public, next_check_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
RETURNING *;

-- name: GetMonitorByIDAndOrg :one
SELECT *
FROM monitors
WHERE id = $1
  AND organization_id = $2;

-- name: CountMonitorsByOrg :one
SELECT count(*)
FROM monitors
WHERE organization_id = $1;

-- name: CountMonitorStatusByOrg :many
SELECT status, count(*) AS count
FROM monitors
WHERE organization_id = $1
GROUP BY status;

-- name: ListMonitorsByOrg :many
SELECT *
FROM monitors
WHERE organization_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: UpdateMonitor :one
UPDATE monitors
SET name = $3,
    url = $4,
    interval_seconds = $5,
    timeout_ms = $6,
    expected_status_min = $7,
    expected_status_max = $8,
    show_public = $9,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: SetMonitorEnabled :one
UPDATE monitors
SET enabled = $3,
    next_check_at = CASE WHEN $3 THEN now() ELSE NULL END,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: DeleteMonitorByIDAndOrg :execrows
DELETE FROM monitors
WHERE id = $1
  AND organization_id = $2;
