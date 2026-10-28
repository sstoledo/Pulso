package statuspages_test

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
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/auth"
	"pulso/internal/statuspages"
)

func TestStatusPageConfigLifecycle(t *testing.T) {
	router, pool, owner, ownerToken, outsider, outsiderToken := newStatusPageAPI(t)
	orgID := owner.Organizations[0].ID

	initial := statusPageRequest[statuspages.Config](t, router, http.MethodGet, "/api/v1/orgs/"+orgID+"/status-page", ownerToken, nil, http.StatusOK)
	if initial.IsPublished || initial.Title == "" || initial.Slug == "" {
		t.Fatalf("initial config = %#v", initial)
	}

	updated := statusPageRequest[statuspages.Config](t, router, http.MethodPatch, "/api/v1/orgs/"+orgID+"/status-page", ownerToken, map[string]any{
		"slug": "  Acme-Status  ", "title": "  Acme Status  ", "is_published": true,
	}, http.StatusOK)
	if updated.Slug != "acme-status" || updated.Title != "Acme Status" || !updated.IsPublished {
		t.Fatalf("updated config = %#v", updated)
	}

	statusPageRequest[map[string]any](t, router, http.MethodPatch, "/api/v1/orgs/"+orgID+"/status-page", ownerToken, map[string]any{
		"slug": "no", "title": "Acme", "is_published": true,
	}, http.StatusUnprocessableEntity)
	statusPageRequest[map[string]any](t, router, http.MethodPatch, "/api/v1/orgs/"+orgID+"/status-page", ownerToken, map[string]any{
		"slug": "-acme", "title": "Acme", "is_published": true,
	}, http.StatusUnprocessableEntity)
	statusPageRequest[map[string]any](t, router, http.MethodPatch, "/api/v1/orgs/"+orgID+"/status-page", ownerToken, map[string]any{
		"slug": "acme", "title": "", "is_published": true,
	}, http.StatusUnprocessableEntity)

	outsiderOrgID := outsider.Organizations[0].ID
	statusPageRequest[map[string]any](t, router, http.MethodPatch, "/api/v1/orgs/"+outsiderOrgID+"/status-page", outsiderToken, map[string]any{
		"slug": "acme-status", "title": "Other", "is_published": true,
	}, http.StatusConflict)
	statusPageRequest[map[string]any](t, router, http.MethodGet, "/api/v1/orgs/"+orgID+"/status-page", outsiderToken, nil, http.StatusForbidden)
	statusPageRequest[map[string]any](t, router, http.MethodPatch, "/api/v1/orgs/"+orgID+"/status-page", outsiderToken, map[string]any{
		"slug": "hijacked", "title": "Hijacked", "is_published": true,
	}, http.StatusForbidden)
	statusPageRequest[map[string]any](t, router, http.MethodGet, "/api/v1/orgs/"+orgID+"/status-page", "", nil, http.StatusUnauthorized)

	var slug string
	if err := pool.QueryRow(context.Background(), "SELECT slug FROM status_pages WHERE organization_id = $1::uuid", orgID).Scan(&slug); err != nil {
		t.Fatal(err)
	}
	if slug != "acme-status" {
		t.Fatalf("stored slug = %q", slug)
	}
}

func newStatusPageAPI(t *testing.T) (*chi.Mux, *pgxpool.Pool, auth.Identity, string, auth.Identity, string) {
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
	organizationRouter := chi.NewRouter()
	organizationRouter.Mount("/status-page", statuspages.NewHTTPHandler(statuspages.NewService(pool), authService, "http://localhost:5173", logger))
	router.Mount("/api/v1/orgs/{organizationID}", organizationRouter)

	owner, ownerToken := registerStatusPageUser(t, authService, "owner")
	outsider, outsiderToken := registerStatusPageUser(t, authService, "outsider")
	for _, identity := range []auth.Identity{owner, outsider} {
		identity := identity
		t.Cleanup(func() {
			for _, organization := range identity.Organizations {
				_, _ = pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1::uuid", organization.ID)
			}
			_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1::uuid", identity.User.ID)
		})
	}
	return router, pool, owner, ownerToken, outsider, outsiderToken
}

func registerStatusPageUser(t *testing.T, service *auth.Service, prefix string) (auth.Identity, string) {
	t.Helper()
	identity, token, err := service.Register(context.Background(), auth.RegisterInput{
		Name: prefix, Email: fmt.Sprintf("%s-%d@example.com", prefix, time.Now().UnixNano()), Password: "password-12345",
	})
	if err != nil {
		t.Fatal(err)
	}
	return identity, token
}

func statusPageRequest[T any](t *testing.T, router http.Handler, method, path, token string, input any, wantStatus int) T {
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
	if token != "" {
		request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
	}
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
	}
	return result
}
