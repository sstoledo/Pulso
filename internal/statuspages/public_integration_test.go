package statuspages_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/auth"
	"pulso/internal/monitors"
	"pulso/internal/statuspages"
)

func TestPublicStatusPageExposesOnlyPublicData(t *testing.T) {
	router, pool, owner := newPublicStatusAPI(t)
	orgID := owner.Organizations[0].ID
	ctx := context.Background()

	monitorService := monitors.NewService(pool)
	public, err := monitorService.Create(ctx, orgID, monitors.CreateInput{
		Name: "Public API", URL: "https://secret-internal.example.com/health", ShowPublic: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	hidden, err := monitorService.Create(ctx, orgID, monitors.CreateInput{
		Name: "Hidden API", URL: "https://hidden-internal.example.com/health", ShowPublic: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := monitorService.Create(ctx, orgID, monitors.CreateInput{
		Name: "No Data API", URL: "https://no-data-internal.example.com/health", ShowPublic: true,
	}); err != nil {
		t.Fatal(err)
	}

	base := time.Now().Add(-time.Hour)
	insertPublicResult(t, pool, public.ID, base, true, 100)
	insertPublicResult(t, pool, public.ID, base.Add(time.Minute), false, 300)
	insertPublicResult(t, pool, hidden.ID, base, false, 500)

	insertPublicIncident(t, pool, public.ID, base, false, "upstream timeout")
	for index := 0; index < 5; index++ {
		insertPublicIncident(t, pool, public.ID, base.Add(time.Duration(index+1)*time.Hour), true, "recovered")
	}
	insertPublicIncident(t, pool, hidden.ID, base.Add(10*time.Hour), false, "hidden monitor failure")

	if _, err := statuspages.NewService(pool).Update(ctx, orgID, statuspages.UpdateInput{
		Slug: "acme", Title: "Acme Status", IsPublished: true,
	}); err != nil {
		t.Fatal(err)
	}

	page := publicStatusRequest[statuspages.PublicPage](t, router, "/api/v1/public/status/acme", http.StatusOK)
	if page.Title != "Acme Status" || len(page.Monitors) != 2 {
		t.Fatalf("page = %#v", page)
	}
	if page.Monitors[0].Name != "No Data API" || page.Monitors[0].Availability24h != nil || page.Monitors[0].Status != "UNKNOWN" {
		t.Fatalf("no-data monitor = %#v", page.Monitors[0])
	}
	if page.Monitors[1].Name != "Public API" || page.Monitors[1].Availability24h == nil || *page.Monitors[1].Availability24h != 0.5 {
		t.Fatalf("public monitor = %#v", page.Monitors[1])
	}
	if len(page.Incidents) != 5 {
		t.Fatalf("incidents = %d, want 5", len(page.Incidents))
	}
	if page.Incidents[0].Status != "RESOLVED" || page.Incidents[0].ResolvedAt == nil || page.Incidents[0].DurationMS == nil {
		t.Fatalf("newest incident = %#v", page.Incidents[0])
	}
	if !page.Incidents[0].OpenedAt.After(page.Incidents[1].OpenedAt) {
		t.Fatalf("incidents are not newest first: %#v", page.Incidents)
	}
	for _, incident := range page.Incidents {
		if incident.MonitorName == "Hidden API" {
			t.Fatalf("hidden monitor incident leaked: %#v", incident)
		}
	}

	raw := publicStatusRaw(t, router, "/api/v1/public/status/acme", http.StatusOK)
	for _, leaked := range []string{"secret-internal", "hidden-internal", "no-data-internal", public.ID, hidden.ID, orgID, "upstream timeout", "show_public", "interval_seconds"} {
		if strings.Contains(raw, leaked) {
			t.Fatalf("public response leaked %q: %s", leaked, raw)
		}
	}

	unpublished := statuspages.NewService(pool)
	if _, err := unpublished.Update(ctx, orgID, statuspages.UpdateInput{Slug: "acme", Title: "Acme Status", IsPublished: false}); err != nil {
		t.Fatal(err)
	}
	hiddenBody := publicStatusRaw(t, router, "/api/v1/public/status/acme", http.StatusNotFound)
	unknownBody := publicStatusRaw(t, router, "/api/v1/public/status/unknown-page", http.StatusNotFound)
	if hiddenBody != unknownBody {
		t.Fatalf("unpublished body %s differs from unknown body %s", hiddenBody, unknownBody)
	}
	publicStatusRaw(t, router, "/api/v1/public/status/Invalid_Slug", http.StatusNotFound)
}

func newPublicStatusAPI(t *testing.T) (*chi.Mux, *pgxpool.Pool, auth.Identity) {
	t.Helper()
	databaseURL := os.Getenv("PULSO_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("PULSO_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	authService, err := auth.NewService(pool)
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := chi.NewRouter()
	router.Mount("/api/v1/public", statuspages.NewPublicHTTPHandler(statuspages.NewService(pool), logger))

	owner, _, err := authService.Register(context.Background(), auth.RegisterInput{
		Name: "owner", Email: fmt.Sprintf("public-owner-%d@example.com", time.Now().UnixNano()), Password: "password-12345",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, organization := range owner.Organizations {
			_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1::uuid", organization.ID)
		}
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1::uuid", owner.User.ID)
	})
	return router, pool, owner
}

func insertPublicResult(t *testing.T, pool *pgxpool.Pool, monitorID string, checkedAt time.Time, success bool, latency int) {
	t.Helper()
	ctx := context.Background()
	var jobID string
	if err := pool.QueryRow(ctx, "INSERT INTO check_jobs (monitor_id, scheduled_for, status) VALUES ($1::uuid, $2, 'completed') RETURNING id", monitorID, checkedAt).Scan(&jobID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO check_results (check_job_id, monitor_id, checked_at, success, http_status, latency_ms)
		VALUES ($1::uuid, $2::uuid, $3, $4, 200, $5)`, jobID, monitorID, checkedAt, success, latency); err != nil {
		t.Fatal(err)
	}
}

func insertPublicIncident(t *testing.T, pool *pgxpool.Pool, monitorID string, openedAt time.Time, resolved bool, reason string) {
	t.Helper()
	var resolvedAt any
	var duration any
	if resolved {
		resolvedAt = openedAt.Add(5 * time.Minute)
		duration = int64(300000)
	}
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO incidents (monitor_id, opened_at, resolved_at, duration_ms, reason)
		 VALUES ($1::uuid, $2, $3, $4, $5)`, monitorID, openedAt, resolvedAt, duration, reason); err != nil {
		t.Fatal(err)
	}
}

func publicStatusRequest[T any](t *testing.T, router http.Handler, path string, wantStatus int) T {
	t.Helper()
	body := publicStatusRaw(t, router, path, wantStatus)
	var result T
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func publicStatusRaw(t *testing.T, router http.Handler, path string, wantStatus int) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != wantStatus {
		t.Fatalf("GET %s: status = %d, want %d; body = %s", path, response.Code, wantStatus, response.Body.String())
	}
	return response.Body.String()
}
