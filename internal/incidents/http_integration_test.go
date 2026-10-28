package incidents_test

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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/auth"
	"pulso/internal/incidents"
	"pulso/internal/monitors"
	"pulso/internal/store/sqlc"
)

func TestIncidentListAuthorizationFilterAndPagination(t *testing.T) {
	router, pool, owner, ownerToken, outsiderToken := newIncidentAPI(t)
	orgID := owner.Organizations[0].ID
	monitor, err := monitors.NewService(pool).Create(context.Background(), orgID, monitors.CreateInput{Name: "API", URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 11; index++ {
		incident, err := sqlc.New(pool).InsertIncident(context.Background(), sqlc.InsertIncidentParams{
			MonitorID: incidentUUID(t, monitor.ID), Reason: fmt.Sprintf("failure %d", index),
		})
		if err != nil {
			t.Fatal(err)
		}
		if index < 10 {
			if _, err := sqlc.New(pool).CloseIncident(context.Background(), incident.ID); err != nil {
				t.Fatal(err)
			}
		}
	}

	firstPage := incidentRequest[incidents.ListResult](t, router, orgID+"/incidents?page_size=10", ownerToken, http.StatusOK)
	if firstPage.Total != 11 || len(firstPage.Items) != 10 || firstPage.TotalPages != 2 || firstPage.Items[0].Status != "OPEN" {
		t.Fatalf("first page = %#v", firstPage)
	}
	secondPage := incidentRequest[incidents.ListResult](t, router, orgID+"/incidents?page=2&page_size=10", ownerToken, http.StatusOK)
	if len(secondPage.Items) != 1 {
		t.Fatalf("second page = %#v", secondPage)
	}
	filtered := incidentRequest[incidents.ListResult](t, router, orgID+"/incidents?monitor_id="+monitor.ID, ownerToken, http.StatusOK)
	if filtered.Total != 11 {
		t.Fatalf("filtered total = %d", filtered.Total)
	}
	otherMonitor := incidentRequest[incidents.ListResult](t, router, orgID+"/incidents?monitor_id="+uuid.NewString(), ownerToken, http.StatusOK)
	if otherMonitor.Total != 0 || len(otherMonitor.Items) != 0 {
		t.Fatalf("other monitor result = %#v", otherMonitor)
	}
	body := incidentRequest[map[string]any](t, router, orgID+"/incidents", outsiderToken, http.StatusForbidden)
	if strings.Contains(fmt.Sprint(body), "example.com") {
		t.Fatal("response leaked monitor URL")
	}
}

func newIncidentAPI(t *testing.T) (*chi.Mux, *pgxpool.Pool, auth.Identity, string, string) {
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
	router.Mount("/api/v1/orgs/{organizationID}", incidents.NewHTTPHandler(incidents.NewService(pool), authService, logger))
	owner, ownerToken := registerIncidentUser(t, authService, "owner")
	outsider, outsiderToken := registerIncidentUser(t, authService, "outsider")
	cleanupIncidentIdentity(t, pool, owner)
	cleanupIncidentIdentity(t, pool, outsider)
	return router, pool, owner, ownerToken, outsiderToken
}

func registerIncidentUser(t *testing.T, service *auth.Service, prefix string) (auth.Identity, string) {
	t.Helper()
	identity, token, err := service.Register(context.Background(), auth.RegisterInput{
		Name: prefix, Email: fmt.Sprintf("%s-%d@example.com", prefix, time.Now().UnixNano()), Password: "password-12345",
	})
	if err != nil {
		t.Fatal(err)
	}
	return identity, token
}

func cleanupIncidentIdentity(t *testing.T, pool *pgxpool.Pool, identity auth.Identity) {
	t.Helper()
	t.Cleanup(func() {
		for _, organization := range identity.Organizations {
			_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1::uuid", organization.ID)
		}
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1::uuid", identity.User.ID)
	})
}

func incidentRequest[T any](t *testing.T, router http.Handler, path, token string, wantStatus int) T {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/"+path, nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, wantStatus, response.Body.String())
	}
	var result T
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func incidentUUID(t *testing.T, value string) pgtype.UUID {
	t.Helper()
	parsed, err := uuid.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}
}
