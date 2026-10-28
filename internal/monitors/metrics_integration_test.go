package monitors_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/monitors"
)

func TestMonitorMetricsAndCheckHistory(t *testing.T) {
	router, pool, owner, ownerToken, outsider, outsiderToken := newMonitorAPI(t)
	orgID := owner.Organizations[0].ID
	monitor, err := monitors.NewService(pool).Create(context.Background(), orgID, monitors.CreateInput{Name: "Metrics API", URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}

	empty := monitorRequest[monitors.Metrics](t, router, http.MethodGet, "/api/v1/orgs/"+orgID+"/monitors/"+monitor.ID+"/metrics", ownerToken, nil, http.StatusOK)
	if empty.Availability24h != nil || empty.AverageLatencyMS != nil || empty.P95LatencyMS != nil || len(empty.Series) != 0 {
		t.Fatalf("empty metrics = %#v", empty)
	}

	base := time.Now().Add(-time.Hour)
	insertMetricResult(t, pool, monitor.ID, base, true, 200, 100)
	insertMetricResult(t, pool, monitor.ID, base.Add(time.Minute), true, 200, 200)
	insertMetricResult(t, pool, monitor.ID, base.Add(2*time.Minute), false, 500, 300)
	insertMetricResult(t, pool, monitor.ID, base.Add(3*time.Minute), true, 400, 400)
	insertMetricResult(t, pool, monitor.ID, base.Add(4*time.Minute), false, 0, 50)

	metrics := monitorRequest[monitors.Metrics](t, router, http.MethodGet, "/api/v1/orgs/"+orgID+"/monitors/"+monitor.ID+"/metrics", ownerToken, nil, http.StatusOK)
	if metrics.Availability24h == nil || *metrics.Availability24h != 0.6 {
		t.Fatalf("availability = %#v", metrics.Availability24h)
	}
	if metrics.AverageLatencyMS == nil || *metrics.AverageLatencyMS != 250 || metrics.P95LatencyMS == nil || *metrics.P95LatencyMS != 385 {
		t.Fatalf("latency aggregates: average=%v p95=%v", *metrics.AverageLatencyMS, *metrics.P95LatencyMS)
	}
	if len(metrics.Series) != 5 || metrics.Series[4].HTTPStatus != nil {
		t.Fatalf("series = %#v", metrics.Series)
	}

	monitorRequest[map[string]any](t, router, http.MethodGet, "/api/v1/orgs/"+outsider.Organizations[0].ID+"/monitors/"+monitor.ID+"/metrics", outsiderToken, nil, http.StatusNotFound)
}

func insertMetricResult(t *testing.T, pool *pgxpool.Pool, monitorID string, checkedAt time.Time, success bool, httpStatus, latency int) {
	t.Helper()
	ctx := context.Background()
	var jobID string
	if err := pool.QueryRow(ctx, "INSERT INTO check_jobs (monitor_id, scheduled_for, status) VALUES ($1::uuid, $2, 'completed') RETURNING id", monitorID, checkedAt).Scan(&jobID); err != nil {
		t.Fatal(err)
	}
	var status any
	if httpStatus > 0 {
		status = httpStatus
	}
	errorSummary := any(nil)
	if !success {
		errorSummary = "request failed"
	}
	if _, err := pool.Exec(ctx, `INSERT INTO check_results (check_job_id, monitor_id, checked_at, success, http_status, latency_ms, error_summary)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)`, jobID, monitorID, checkedAt, success, status, latency, errorSummary); err != nil {
		t.Fatal(err)
	}
}
