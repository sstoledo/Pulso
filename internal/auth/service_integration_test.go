package auth_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/auth"
	"pulso/internal/organizations"
)

func TestServiceIdentityLifecycle(t *testing.T) {
	service, pool := newIntegrationService(t)
	ctx := context.Background()
	email := uniqueEmail("owner")
	input := auth.RegisterInput{Name: "Ada Lovelace", Email: email, Password: "password-12345"}

	identity, rawToken, err := service.Register(ctx, input)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	cleanupIdentity(t, pool, identity)
	if len(identity.Organizations) != 1 || identity.Organizations[0].Role != "owner" {
		t.Fatalf("organizations = %#v", identity.Organizations)
	}
	var slug string
	var published bool
	if err := pool.QueryRow(ctx, "SELECT slug, is_published FROM status_pages WHERE organization_id = $1::uuid", identity.Organizations[0].ID).Scan(&slug, &published); err != nil {
		t.Fatalf("status page was not created: %v", err)
	}
	if !organizations.ValidStatusSlug(slug) || published {
		t.Fatalf("status page slug = %q, published = %v", slug, published)
	}
	if rawToken == "" || strings.Contains(rawToken, ".") {
		t.Fatalf("unexpected opaque token %q", rawToken)
	}

	var passwordHash string
	if err := pool.QueryRow(ctx, "SELECT password_hash FROM users WHERE email = $1", email).Scan(&passwordHash); err != nil {
		t.Fatal(err)
	}
	if passwordHash == input.Password || !strings.HasPrefix(passwordHash, "$argon2id$") {
		t.Fatalf("stored password hash = %q", passwordHash)
	}
	var storedTokenHash []byte
	if err := pool.QueryRow(ctx, "SELECT token_hash FROM sessions WHERE user_id = $1::uuid", identity.User.ID).Scan(&storedTokenHash); err != nil {
		t.Fatal(err)
	}
	wantHash := auth.HashSessionToken(rawToken)
	if string(storedTokenHash) != string(wantHash[:]) || string(storedTokenHash) == rawToken {
		t.Fatal("session token was not stored as its SHA-256 hash")
	}

	authenticated, err := service.Authenticate(ctx, rawToken)
	if err != nil || authenticated.User.ID != identity.User.ID {
		t.Fatalf("Authenticate() identity = %#v, err = %v", authenticated, err)
	}
	organization, err := service.Organization(ctx, identity.User.ID, identity.Organizations[0].ID)
	if err != nil || organization.Role != "owner" {
		t.Fatalf("Organization() = %#v, err = %v", organization, err)
	}
	if _, err := pool.Exec(ctx, "UPDATE sessions SET created_at = now() - interval '2 days', expires_at = now() - interval '1 day' WHERE user_id = $1::uuid", identity.User.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(ctx, rawToken); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("expired Authenticate() error = %v", err)
	}

	var organizationsBefore int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM organizations").Scan(&organizationsBefore); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Register(ctx, input); !errors.Is(err, auth.ErrEmailAlreadyRegistered) {
		t.Fatalf("duplicate Register() error = %v", err)
	}
	var organizationsAfter int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM organizations").Scan(&organizationsAfter); err != nil {
		t.Fatal(err)
	}
	if organizationsAfter != organizationsBefore {
		t.Fatal("duplicate registration left an organization behind")
	}

	if _, _, err := service.Login(ctx, email, "wrong-password"); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("wrong-password Login() error = %v", err)
	}
	_, loginToken, err := service.Login(ctx, strings.ToUpper(email), input.Password)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err := service.Logout(ctx, loginToken); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := service.Authenticate(ctx, loginToken); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("revoked Authenticate() error = %v", err)
	}
}

func TestServiceRejectsCrossOrganizationAccess(t *testing.T) {
	service, pool := newIntegrationService(t)
	ctx := context.Background()
	owner, _, err := service.Register(ctx, auth.RegisterInput{Name: "Owner", Email: uniqueEmail("owner"), Password: "password-12345"})
	if err != nil {
		t.Fatal(err)
	}
	cleanupIdentity(t, pool, owner)
	outsider, _, err := service.Register(ctx, auth.RegisterInput{Name: "Outsider", Email: uniqueEmail("outsider"), Password: "password-12345"})
	if err != nil {
		t.Fatal(err)
	}
	cleanupIdentity(t, pool, outsider)

	if _, err := service.Organization(ctx, outsider.User.ID, owner.Organizations[0].ID); !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("cross-organization error = %v, want ErrForbidden", err)
	}
}

func newIntegrationService(t *testing.T) (*auth.Service, *pgxpool.Pool) {
	t.Helper()
	databaseURL := os.Getenv("PULSO_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("PULSO_TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	service, err := auth.NewService(pool)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service, pool
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@example.com", prefix, time.Now().UnixNano())
}

func cleanupIdentity(t *testing.T, pool *pgxpool.Pool, identity auth.Identity) {
	t.Helper()
	t.Cleanup(func() {
		for _, organization := range identity.Organizations {
			if _, err := pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1::uuid", organization.ID); err != nil {
				t.Errorf("clean organization: %v", err)
			}
		}
		if _, err := pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1::uuid", identity.User.ID); err != nil {
			t.Errorf("clean user: %v", err)
		}
	})
}
