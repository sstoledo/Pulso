package auth_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"pulso/internal/auth"
)

func TestHTTPAuthenticationAndAuthorizationLifecycle(t *testing.T) {
	service, pool := newIntegrationService(t)
	handler := auth.NewHTTPHandler(service, false, "http://localhost:5173", discardLogger())
	router := chi.NewRouter()
	router.Mount("/api/v1", handler)

	identity, sessionCookie, body := registerHTTP(t, router, "Ada Lovelace", uniqueEmail("http"), "password-12345")
	cleanupIdentity(t, pool, identity)
	if strings.Contains(body, "password") || strings.Contains(body, "token") {
		t.Fatalf("registration response exposed credentials: %s", body)
	}
	if sessionCookie.Name != auth.SessionCookieName || !sessionCookie.HttpOnly || sessionCookie.Secure || sessionCookie.Path != "/" || sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie = %#v", sessionCookie)
	}

	response := authenticatedRequest(t, router, http.MethodGet, "/api/v1/auth/me", nil, sessionCookie)
	if response.Code != http.StatusOK {
		t.Fatalf("me status = %d, body = %s", response.Code, response.Body.String())
	}
	response = authenticatedRequest(t, router, http.MethodGet, "/api/v1/orgs/"+identity.Organizations[0].ID, nil, sessionCookie)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"role":"owner"`) {
		t.Fatalf("organization response = %d %s", response.Code, response.Body.String())
	}

	outsider, outsiderCookie, _ := registerHTTP(t, router, "Outsider", uniqueEmail("outsider-http"), "password-12345")
	cleanupIdentity(t, pool, outsider)
	response = authenticatedRequest(t, router, http.MethodGet, "/api/v1/orgs/"+identity.Organizations[0].ID, nil, outsiderCookie)
	assertErrorResponse(t, response, http.StatusForbidden, "organization_forbidden")

	response = authenticatedRequest(t, router, http.MethodPost, "/api/v1/auth/logout", nil, sessionCookie)
	if response.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, body = %s", response.Code, response.Body.String())
	}
	response = authenticatedRequest(t, router, http.MethodGet, "/api/v1/auth/me", nil, sessionCookie)
	assertErrorResponse(t, response, http.StatusUnauthorized, "unauthorized")

	loginPayload := []byte(`{"email":"` + identity.User.Email + `","password":"password-12345"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginPayload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost:5173")
	loginResponse := httptest.NewRecorder()
	router.ServeHTTP(loginResponse, request)
	if loginResponse.Code != http.StatusOK || len(loginResponse.Result().Cookies()) != 1 {
		t.Fatalf("login response = %d %s", loginResponse.Code, loginResponse.Body.String())
	}
	response = authenticatedRequest(t, router, http.MethodGet, "/api/v1/auth/me", nil, loginResponse.Result().Cookies()[0])
	if response.Code != http.StatusOK {
		t.Fatalf("me after login = %d %s", response.Code, response.Body.String())
	}
}

func TestHTTPSecureCookieOutsideDevelopment(t *testing.T) {
	service, pool := newIntegrationService(t)
	handler := auth.NewHTTPHandler(service, true, "https://status.example", discardLogger())
	router := chi.NewRouter()
	router.Mount("/api/v1", handler)

	payload := []byte(`{"name":"Secure User","email":"` + uniqueEmail("secure") + `","password":"password-12345"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://status.example")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", response.Code, response.Body.String())
	}
	var identity auth.Identity
	if err := json.Unmarshal(response.Body.Bytes(), &identity); err != nil {
		t.Fatal(err)
	}
	cleanupIdentity(t, pool, identity)
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("secure cookies = %#v", cookies)
	}
}

func TestHTTPRejectsCrossOriginMutation(t *testing.T) {
	service, _ := newIntegrationService(t)
	handler := auth.NewHTTPHandler(service, false, "http://localhost:5173", discardLogger())
	router := chi.NewRouter()
	router.Mount("/api/v1", handler)

	payload := []byte(`{"name":"Mallory","email":"mallory@example.com","password":"password-12345"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusForbidden, "origin_forbidden")
}

func registerHTTP(t *testing.T, router http.Handler, name, email, password string) (auth.Identity, *http.Cookie, string) {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"name": name, "email": email, "password": password})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost:5173")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", response.Code, response.Body.String())
	}
	var identity auth.Identity
	if err := json.Unmarshal(response.Body.Bytes(), &identity); err != nil {
		t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %#v", cookies)
	}
	return identity, cookies[0], response.Body.String()
}

func authenticatedRequest(t *testing.T, router http.Handler, method, path string, body io.Reader, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, body)
	request.AddCookie(cookie)
	request.Header.Set("Origin", "http://localhost:5173")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func assertErrorResponse(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body.String())
	}
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.Code != code {
		t.Fatalf("error code = %q, want %q; body = %s", envelope.Error.Code, code, response.Body.String())
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
