-- name: GetMonitorMetrics24h :one
SELECT count(*) AS total,
       count(*) FILTER (WHERE r.success) AS successes,
       count(*) FILTER (WHERE r.http_status IS NOT NULL) AS responses,
       cast(coalesce(avg(r.latency_ms) FILTER (WHERE r.http_status IS NOT NULL), 0) as float8) AS average_latency_ms,
       cast(coalesce(percentile_cont(0.95) WITHIN GROUP (ORDER BY r.latency_ms) FILTER (WHERE r.http_status IS NOT NULL), 0) as float8) AS p95_latency_ms
FROM check_results AS r
JOIN monitors AS m ON m.id = r.monitor_id
WHERE r.monitor_id = $1
  AND m.organization_id = $2
  AND r.checked_at >= now() - interval '24 hours';

-- name: ListMonitorMetricSeries24h :many
SELECT r.checked_at, r.latency_ms, r.success, r.http_status
FROM check_results AS r
JOIN monitors AS m ON m.id = r.monitor_id
WHERE r.monitor_id = $1
  AND m.organization_id = $2
  AND r.checked_at >= now() - interval '24 hours'
ORDER BY r.checked_at, r.id;
