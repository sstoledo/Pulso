package statuspages

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/organizations"
	"pulso/internal/store/sqlc"
)

var ErrSlugConflict = errors.New("status page slug already exists")

type Config struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	IsPublished bool   `json:"is_published"`
}

type UpdateInput struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	IsPublished bool   `json:"is_published"`
}

type Service struct {
	queries *sqlc.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{queries: sqlc.New(pool)}
}

func (s *Service) Get(ctx context.Context, organizationID string) (Config, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return Config{}, fmt.Errorf("parse organization ID: %w", err)
	}
	page, err := s.queries.GetStatusPageByOrg(ctx, pgtype.UUID{Bytes: organizationUUID, Valid: true})
	if err != nil {
		return Config{}, fmt.Errorf("get status page: %w", err)
	}
	return Config{Slug: page.Slug, Title: page.Title, IsPublished: page.IsPublished}, nil
}

func (s *Service) Update(ctx context.Context, organizationID string, input UpdateInput) (Config, error) {
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	input.Title = strings.TrimSpace(input.Title)
	if !organizations.ValidStatusSlug(input.Slug) {
		return Config{}, &ValidationError{Fields: map[string]string{"slug": "Use 3–64 lowercase letters, numbers or hyphens, without leading/trailing hyphens."}}
	}
	if len(input.Title) == 0 || len(input.Title) > 100 {
		return Config{}, &ValidationError{Fields: map[string]string{"title": "Enter between 1 and 100 characters."}}
	}
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return Config{}, fmt.Errorf("parse organization ID: %w", err)
	}
	page, err := s.queries.UpdateStatusPage(ctx, sqlc.UpdateStatusPageParams{
		OrganizationID: pgtype.UUID{Bytes: organizationUUID, Valid: true},
		Slug:           input.Slug, Title: input.Title, IsPublished: input.IsPublished,
	})
	if isUniqueViolation(err) {
		return Config{}, ErrSlugConflict
	}
	if err != nil {
		return Config{}, fmt.Errorf("update status page: %w", err)
	}
	return Config{Slug: page.Slug, Title: page.Title, IsPublished: page.IsPublished}, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
