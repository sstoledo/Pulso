package monitors_test

import (
	"bytes"
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
)

func TestMonitorHTTPCrudAndTenancy(t *testing.T) {
	router, pool, owner, ownerToken, outsider, outsiderToken := newMonitorAPI(t)
	orgID := owner.Organizations[0].ID

	created := monitorRequest[monitors.Monitor](t, router, http.MethodPost, "/api/v1/orgs/"+orgID+"/monitors", ownerToken, map[string]any{
		"name": "Public API", "url": "https://example.com/health",
	}, http.StatusCreated)
	if created.Status != "UNKNOWN" || !created.Enabled || created.ShowPublic || created.IntervalSeconds != 60 || created.TimeoutMS != 5000 || created.ExpectedStatusMin != 200 || created.ExpectedStatusMax != 399 || created.LastCheckAt != nil || created.LastLatencyMS != nil {
		t.Fatalf("created monitor = %#v", created)
	}

	summary := monitorRequest[monitors.Summary](t, router, http.MethodGet, "/api/v1/orgs/"+orgID+"/dashboard/summary", ownerToken, nil, http.StatusOK)
	if summary.Unknown != 1 || summary.Up != 0 || summary.Down != 0 {
		t.Fatalf("summary = %#v", summary)
	}

	list := monitorRequest[monitors.ListResult](t, router, http.MethodGet, "/api/v1/orgs/"+orgID+"/monitors", ownerToken, nil, http.StatusOK)
	if list.Total != 1 || len(list.Items) != 1 || list.TotalPages != 1 {
		t.Fatalf("list = %#v", list)
	}

	updated := monitorRequest[monitors.Monitor](t, router, http.MethodPatch, "/api/v1/orgs/"+orgID+"/monitors/"+created.ID, ownerToken, map[string]any{
		"name": "Updated API", "timeout_ms": 250, "show_public": true,
	}, http.StatusOK)
	if updated.Name != "Updated API" || updated.TimeoutMS != 250 || !updated.ShowPublic || updated.Status != "UNKNOWN" {
		t.Fatalf("updated monitor = %#v", updated)
	}

	paused := monitorRequest[monitors.Monitor](t, router, http.MethodPost, "/api/v1/orgs/"+orgID+"/monitors/"+created.ID+"/pause", ownerToken, nil, http.StatusOK)
	if paused.Enabled || paused.Status != "UNKNOWN" {
		t.Fatalf("paused monitor = %#v", paused)
	}
	var nextCheckValid bool
	if err := pool.QueryRow(context.Background(), "SELECT next_check_at IS NOT NULL FROM monitors WHERE id = $1::uuid", created.ID).Scan(&nextCheckValid); err != nil {
		t.Fatal(err)
	}
	if nextCheckValid {
		t.Fatal("paused monitor still has a scheduled check")
	}
	resumed := monitorRequest[monitors.Monitor](t, router, http.MethodPost, "/api/v1/orgs/"+orgID+"/monitors/"+created.ID+"/resume", ownerToken, nil, http.StatusOK)
	if !resumed.Enabled {
		t.Fatal("resume did not enable the monitor")
	}

	outsiderOrgID := outsider.Organizations[0].ID
	monitorRequest[map[string]any](t, router, http.MethodGet, "/api/v1/orgs/"+outsiderOrgID+"/monitors/"+created.ID, outsiderToken, nil, http.StatusNotFound)
	monitorRequest[map[string]any](t, router, http.MethodGet, "/api/v1/orgs/"+orgID+"/monitors", outsiderToken, nil, http.StatusForbidden)
	monitorRequest[map[string]any](t, router, http.MethodDelete, "/api/v1/orgs/"+outsiderOrgID+"/monitors/"+created.ID, outsiderToken, nil, http.StatusNotFound)
	monitorRequest[monitors.Monitor](t, router, http.MethodGet, "/api/v1/orgs/"+orgID+"/monitors/"+created.ID, ownerToken, nil, http.StatusOK)

	monitorRequest[map[string]any](t, router, http.MethodDelete, "/api/v1/orgs/"+orgID+"/monitors/"+created.ID, ownerToken, nil, http.StatusNoContent)
	monitorRequest[map[string]any](t, router, http.MethodGet, "/api/v1/orgs/"+orgID+"/monitors/"+created.ID, ownerToken, nil, http.StatusNotFound)
}

func TestMonitorHTTPValidationAndPagination(t *testing.T) {
	router, _, owner, ownerToken, _, _ := newMonitorAPI(t)
	orgID := owner.Organizations[0].ID

	monitorRequest[map[string]any](t, router, http.MethodPost, "/api/v1/orgs/"+orgID+"/monitors", ownerToken, map[string]any{
		"name": "Bad", "url": "https://user:secret@example.com",
	}, http.StatusUnprocessableEntity)
	for index := 0; index < 11; index++ {
		monitorRequest[monitors.Monitor](t, router, http.MethodPost, "/api/v1/orgs/"+orgID+"/monitors", ownerToken, map[string]any{
			"name": "API " + time.Duration(index).String(), "url": "https://example.com/" + time.Duration(index).String(),
		}, http.StatusCreated)
	}
	secondPage := monitorRequest[monitors.ListResult](t, router, http.MethodGet, "/api/v1/orgs/"+orgID+"/monitors?page=2&page_size=10", ownerToken, nil, http.StatusOK)
	if secondPage.Total != 11 || len(secondPage.Items) != 1 || secondPage.TotalPages != 2 || secondPage.Page != 2 {
		t.Fatalf("second page = %#v", secondPage)
	}
}

func TestMonitorHTTPCreateRejectsExplicitZeroValues(t *testing.T) {
	router, pool, owner, ownerToken, _, _ := newMonitorAPI(t)
	orgID := owner.Organizations[0].ID
	for _, field := range []string{"interval_seconds", "timeout_ms", "expected_status_min", "expected_status_max"} {
		t.Run(field, func(t *testing.T) {
			var before, after int
			if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM monitors WHERE organization_id = $1::uuid", orgID).Scan(&before); err != nil {
				t.Fatal(err)
			}
			payload := map[string]any{"name": "Invalid", "url": "https://example.com"}
			payload[field] = 0
			monitorRequest[map[string]any](t, router, http.MethodPost, "/api/v1/orgs/"+orgID+"/monitors", ownerToken, payload, http.StatusUnprocessableEntity)
			if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM monitors WHERE organization_id = $1::uuid", orgID).Scan(&after); err != nil {
				t.Fatal(err)
			}
			if after != before {
				t.Fatalf("monitor count changed from %d to %d", before, after)
			}
		})
	}
}

func newMonitorAPI(t *testing.T) (*chi.Mux, *pgxpool.Pool, auth.Identity, string, auth.Identity, string) {
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
	router.Mount("/api/v1", auth.NewHTTPHandler(authService, false, "http://localhost:5173", logger))
	router.Mount("/api/v1/orgs/{organizationID}", monitors.NewHTTPHandler(monitors.NewService(pool), authService, "http://localhost:5173", logger))

	owner, ownerToken := registerMonitorUser(t, authService, "owner")
	outsider, outsiderToken := registerMonitorUser(t, authService, "outsider")
	cleanupMonitorIdentity(t, pool, owner)
	cleanupMonitorIdentity(t, pool, outsider)
	return router, pool, owner, ownerToken, outsider, outsiderToken
}

func registerMonitorUser(t *testing.T, service *auth.Service, prefix string) (auth.Identity, string) {
	t.Helper()
	identity, token, err := service.Register(context.Background(), auth.RegisterInput{
		Name: prefix, Email: fmt.Sprintf("%s-%d@example.com", prefix, time.Now().UnixNano()), Password: "password-12345",
	})
	if err != nil {
		t.Fatal(err)
	}
	return identity, token
}

func cleanupMonitorIdentity(t *testing.T, pool *pgxpool.Pool, identity auth.Identity) {
	t.Helper()
	t.Cleanup(func() {
		for _, organization := range identity.Organizations {
			_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1::uuid", organization.ID)
		}
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1::uuid", identity.User.ID)
	})
}

func monitorRequest[T any](t *testing.T, router http.Handler, method, path, token string, input any, wantStatus int) T {
	t.Helper()
	var body io.Reader
	if input != nil {
		payload, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(payload)
	}
	request := httptest.NewRequest(method, path, body)
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Origin", "http://localhost:5173")
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != wantStatus {
		t.Fatalf("%s %s: status = %d, want %d; body = %s", method, path, response.Code, wantStatus, response.Body.String())
	}
	var result T
	if response.Body.Len() > 0 {
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(response.Body.String(), "database") || strings.Contains(response.Body.String(), "sql") {
			t.Fatalf("response leaked internal details: %s", response.Body.String())
		}
	}
	return result
}
