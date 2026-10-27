package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type organizationContextKey struct{}

func RequireAuthentication(service *Service, logger *slog.Logger) func(http.Handler) http.Handler {
	handler := &HTTPHandler{service: service, logger: logger}
	return handler.requireAuthentication
}

func RequireOrganization(service *Service, logger *slog.Logger) func(http.Handler) http.Handler {
	handler := &HTTPHandler{service: service, logger: logger}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity := IdentityFromContext(r.Context())
			organization, err := service.Organization(r.Context(), identity.User.ID, chi.URLParam(r, "organizationID"))
			if err != nil {
				handler.writeError(w, r, err)
				return
			}
			ctx := context.WithValue(r.Context(), organizationContextKey{}, organization)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireSameOrigin(allowedOrigin string) func(http.Handler) http.Handler {
	handler := &HTTPHandler{allowedOrigin: allowedOrigin}
	return handler.requireSameOrigin
}

func IdentityFromContext(ctx context.Context) Identity {
	return sessionFromContext(ctx).identity
}

func OrganizationFromContext(ctx context.Context) Organization {
	return ctx.Value(organizationContextKey{}).(Organization)
}
