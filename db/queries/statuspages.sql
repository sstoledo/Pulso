-- name: CreateStatusPage :one
INSERT INTO status_pages (organization_id, slug, title)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetStatusPageByOrg :one
SELECT *
FROM status_pages
WHERE organization_id = $1;

-- name: GetStatusPageBySlug :one
SELECT *
FROM status_pages
WHERE slug = $1
  AND is_published;

-- name: UpdateStatusPage :one
UPDATE status_pages
SET slug = $2, title = $3, is_published = $4, updated_at = now()
WHERE organization_id = $1
RETURNING *;

-- name: ListPublicMonitorsByStatusPage :many
SELECT m.name,
       m.status,
       count(r.id) AS total_checks,
       count(r.id) FILTER (WHERE r.success) AS successful_checks
FROM status_pages AS p
JOIN monitors AS m ON m.organization_id = p.organization_id
LEFT JOIN check_results AS r ON r.monitor_id = m.id AND r.checked_at >= now() - interval '24 hours'
WHERE p.slug = $1
  AND p.is_published
  AND m.show_public
GROUP BY m.id, m.name, m.status
ORDER BY m.name, m.id;

-- name: ListPublicIncidentsByStatusPage :many
SELECT i.opened_at, i.resolved_at, i.duration_ms, m.name AS monitor_name
FROM status_pages AS p
JOIN monitors AS m ON m.organization_id = p.organization_id
JOIN incidents AS i ON i.monitor_id = m.id
WHERE p.slug = $1
  AND p.is_published
  AND m.show_public
ORDER BY i.opened_at DESC, i.id DESC
LIMIT 5;
