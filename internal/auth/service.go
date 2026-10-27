package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/store/sqlc"
)

const SessionTTL = 7 * 24 * time.Hour

var (
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrForbidden              = errors.New("organization access forbidden")
)

type ValidationError struct {
	Fields map[string]string `json:"fields"`
}

func (e *ValidationError) Error() string {
	return "validation failed"
}

type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type Identity struct {
	User          User           `json:"user"`
	Organizations []Organization `json:"organizations"`
}

type Service struct {
	pool              *pgxpool.Pool
	queries           *sqlc.Queries
	dummyPasswordHash string
}

func NewService(pool *pgxpool.Pool) (*Service, error) {
	dummyHash, err := HashPassword("not-a-real-user-password")
	if err != nil {
		return nil, fmt.Errorf("initialize password verification: %w", err)
	}
	return &Service{pool: pool, queries: sqlc.New(pool), dummyPasswordHash: dummyHash}, nil
}

func ValidateRegistration(name, email, password string) (RegisterInput, error) {
	input := RegisterInput{
		Name:     strings.TrimSpace(name),
		Email:    strings.ToLower(strings.TrimSpace(email)),
		Password: password,
	}
	fields := make(map[string]string)
	if utf8.RuneCountInString(input.Name) < 1 || utf8.RuneCountInString(input.Name) > 80 {
		fields["name"] = "Enter between 1 and 80 characters."
	}
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email || len(input.Email) > 254 {
		fields["email"] = "Enter a valid email address."
	}
	passwordLength := utf8.RuneCountInString(password)
	if passwordLength < 12 || passwordLength > 128 {
		fields["password"] = "Enter between 12 and 128 characters."
	}
	if len(fields) > 0 {
		return RegisterInput{}, &ValidationError{Fields: fields}
	}
	return input, nil
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (Identity, string, error) {
	input, err := ValidateRegistration(input.Name, input.Email, input.Password)
	if err != nil {
		return Identity{}, "", err
	}
	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return Identity{}, "", fmt.Errorf("hash password: %w", err)
	}
	rawToken, tokenHash, err := NewSessionToken()
	if err != nil {
		return Identity{}, "", err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Identity{}, "", fmt.Errorf("begin registration: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := s.queries.WithTx(tx)

	user, err := queries.CreateUser(ctx, sqlc.CreateUserParams{
		Name: input.Name, Email: input.Email, PasswordHash: passwordHash,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Identity{}, "", ErrEmailAlreadyRegistered
		}
		return Identity{}, "", fmt.Errorf("create user: %w", err)
	}
	organizationName := input.Name + "'s organization"
	organization, err := queries.CreateOrganization(ctx, organizationName)
	if err != nil {
		return Identity{}, "", fmt.Errorf("create organization: %w", err)
	}
	if _, err := queries.CreateMembership(ctx, sqlc.CreateMembershipParams{
		OrganizationID: organization.ID, UserID: user.ID, Role: "owner",
	}); err != nil {
		return Identity{}, "", fmt.Errorf("create owner membership: %w", err)
	}
	expiresAt := time.Now().UTC().Add(SessionTTL)
	if _, err := queries.CreateSession(ctx, sqlc.CreateSessionParams{
		UserID: user.ID, TokenHash: tokenHash[:], ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		return Identity{}, "", fmt.Errorf("create session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		if isUniqueViolation(err) {
			return Identity{}, "", ErrEmailAlreadyRegistered
		}
		return Identity{}, "", fmt.Errorf("commit registration: %w", err)
	}

	identity := Identity{
		User: userFromModel(user),
		Organizations: []Organization{{
			ID: uuid.UUID(organization.ID.Bytes).String(), Name: organization.Name, Role: "owner",
		}},
	}
	return identity, rawToken, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (Identity, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.queries.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		_, _ = VerifyPassword(s.dummyPasswordHash, password)
		return Identity{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return Identity{}, "", fmt.Errorf("find user: %w", err)
	}
	valid, err := VerifyPassword(user.PasswordHash, password)
	if err != nil {
		return Identity{}, "", fmt.Errorf("verify password hash: %w", err)
	}
	if !valid {
		return Identity{}, "", ErrInvalidCredentials
	}

	rawToken, tokenHash, err := NewSessionToken()
	if err != nil {
		return Identity{}, "", err
	}
	if _, err := s.queries.CreateSession(ctx, sqlc.CreateSessionParams{
		UserID: user.ID, TokenHash: tokenHash[:],
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().UTC().Add(SessionTTL), Valid: true},
	}); err != nil {
		return Identity{}, "", fmt.Errorf("create session: %w", err)
	}
	identity, err := s.identityForUser(ctx, user)
	if err != nil {
		return Identity{}, "", err
	}
	return identity, rawToken, nil
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (Identity, error) {
	if rawToken == "" {
		return Identity{}, ErrUnauthorized
	}
	hash := HashSessionToken(rawToken)
	row, err := s.queries.GetSessionUser(ctx, hash[:])
	if errors.Is(err, pgx.ErrNoRows) {
		return Identity{}, ErrUnauthorized
	}
	if err != nil {
		return Identity{}, fmt.Errorf("find session: %w", err)
	}
	identity, err := s.identityForUser(ctx, sqlc.User{
		ID: row.ID, Name: row.Name, Email: row.Email, CreatedAt: row.CreatedAt,
	})
	if err != nil {
		return Identity{}, err
	}
	return identity, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return ErrUnauthorized
	}
	hash := HashSessionToken(rawToken)
	if _, err := s.queries.RevokeSession(ctx, hash[:]); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (s *Service) Organization(ctx context.Context, userID, organizationID string) (Organization, error) {
	userUUID, userErr := uuid.Parse(userID)
	organizationUUID, organizationErr := uuid.Parse(organizationID)
	if userErr != nil || organizationErr != nil {
		return Organization{}, ErrForbidden
	}
	row, err := s.queries.GetOrganizationMembership(ctx, sqlc.GetOrganizationMembershipParams{
		UserID:         pgtype.UUID{Bytes: userUUID, Valid: true},
		OrganizationID: pgtype.UUID{Bytes: organizationUUID, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, ErrForbidden
	}
	if err != nil {
		return Organization{}, fmt.Errorf("check organization membership: %w", err)
	}
	return Organization{ID: uuid.UUID(row.ID.Bytes).String(), Name: row.Name, Role: row.Role}, nil
}

func (s *Service) identityForUser(ctx context.Context, user sqlc.User) (Identity, error) {
	rows, err := s.queries.ListOrganizationsByUser(ctx, user.ID)
	if err != nil {
		return Identity{}, fmt.Errorf("list organizations: %w", err)
	}
	organizations := make([]Organization, 0, len(rows))
	for _, row := range rows {
		organizations = append(organizations, Organization{
			ID: uuid.UUID(row.ID.Bytes).String(), Name: row.Name, Role: row.Role,
		})
	}
	return Identity{User: userFromModel(user), Organizations: organizations}, nil
}

func userFromModel(user sqlc.User) User {
	return User{
		ID: uuid.UUID(user.ID.Bytes).String(), Name: user.Name, Email: user.Email, CreatedAt: user.CreatedAt.Time,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
