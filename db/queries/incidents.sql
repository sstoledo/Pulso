-- name: InsertIncident :one
INSERT INTO incidents (monitor_id, reason)
VALUES ($1, $2)
RETURNING *;

-- name: GetOpenIncidentByMonitorForUpdate :one
SELECT *
FROM incidents
WHERE monitor_id = $1
  AND resolved_at IS NULL
FOR UPDATE;

-- name: CloseIncident :one
UPDATE incidents
SET resolved_at = now(),
    duration_ms = floor(extract(epoch FROM (now() - opened_at)) * 1000)::bigint
WHERE id = $1
  AND resolved_at IS NULL
RETURNING *;

-- name: CountIncidentsByOrg :one
SELECT count(*)
FROM incidents AS i
JOIN monitors AS m ON m.id = i.monitor_id
WHERE m.organization_id = $1;

-- name: ListIncidentsByOrg :many
SELECT i.*
FROM incidents AS i
JOIN monitors AS m ON m.id = i.monitor_id
WHERE m.organization_id = $1
ORDER BY i.opened_at DESC, i.id DESC
LIMIT $2 OFFSET $3;

-- name: CountIncidentsByMonitorAndOrg :one
SELECT count(*)
FROM incidents AS i
JOIN monitors AS m ON m.id = i.monitor_id
WHERE m.organization_id = $1
  AND i.monitor_id = $2;

-- name: ListIncidentsByMonitorAndOrg :many
SELECT i.*
FROM incidents AS i
JOIN monitors AS m ON m.id = i.monitor_id
WHERE m.organization_id = $1
  AND i.monitor_id = $2
ORDER BY i.opened_at DESC, i.id DESC
LIMIT $3 OFFSET $4;
