package monitors

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/store/sqlc"
)

var ErrNotFound = errors.New("monitor not found")

type Monitor struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	URL               string     `json:"url"`
	IntervalSeconds   int        `json:"interval_seconds"`
	TimeoutMS         int        `json:"timeout_ms"`
	ExpectedStatusMin int        `json:"expected_status_min"`
	ExpectedStatusMax int        `json:"expected_status_max"`
	Enabled           bool       `json:"enabled"`
	ShowPublic        bool       `json:"show_public"`
	Status            string     `json:"status"`
	LastCheckAt       *time.Time `json:"last_check_at"`
	LastLatencyMS     *int       `json:"last_latency_ms"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type UpdateInput struct {
	Name              *string `json:"name"`
	URL               *string `json:"url"`
	IntervalSeconds   *int    `json:"interval_seconds"`
	TimeoutMS         *int    `json:"timeout_ms"`
	ExpectedStatusMin *int    `json:"expected_status_min"`
	ExpectedStatusMax *int    `json:"expected_status_max"`
	ShowPublic        *bool   `json:"show_public"`
}

type ListResult struct {
	Items      []Monitor `json:"items"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalPages int       `json:"total_pages"`
}

type Service struct {
	queries *sqlc.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{queries: sqlc.New(pool)}
}

func (s *Service) Create(ctx context.Context, organizationID string, input CreateInput) (Monitor, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return Monitor{}, fmt.Errorf("parse organization ID: %w", err)
	}
	input, err = NormalizeCreate(input)
	if err != nil {
		return Monitor{}, err
	}
	monitor, err := s.queries.CreateMonitor(ctx, sqlc.CreateMonitorParams{
		OrganizationID: pgtype.UUID{Bytes: organizationUUID, Valid: true}, Name: input.Name, Url: input.URL,
		IntervalSeconds: int32(input.IntervalSeconds), TimeoutMs: int32(input.TimeoutMS),
		ExpectedStatusMin: int32(input.ExpectedStatusMin), ExpectedStatusMax: int32(input.ExpectedStatusMax),
		ShowPublic: input.ShowPublic,
	})
	if err != nil {
		return Monitor{}, fmt.Errorf("create monitor: %w", err)
	}
	return monitorFromModel(monitor), nil
}

func (s *Service) List(ctx context.Context, organizationID string, page, pageSize int) (ListResult, error) {
	fields := make(map[string]string)
	if page < 1 {
		fields["page"] = "Page must be at least 1."
	}
	if pageSize < 1 || pageSize > 50 {
		fields["page_size"] = "Page size must be between 1 and 50."
	}
	if len(fields) > 0 {
		return ListResult{}, &ValidationError{Fields: fields}
	}
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return ListResult{}, fmt.Errorf("parse organization ID: %w", err)
	}
	orgID := pgtype.UUID{Bytes: organizationUUID, Valid: true}
	total, err := s.queries.CountMonitorsByOrg(ctx, orgID)
	if err != nil {
		return ListResult{}, fmt.Errorf("count monitors: %w", err)
	}
	rows, err := s.queries.ListMonitorsByOrg(ctx, sqlc.ListMonitorsByOrgParams{
		OrganizationID: orgID, Limit: int32(pageSize), Offset: int32((page - 1) * pageSize),
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("list monitors: %w", err)
	}
	items := make([]Monitor, 0, len(rows))
	for _, row := range rows {
		items = append(items, monitorFromModel(row))
	}
	return ListResult{
		Items: items, Total: int(total), Page: page, PageSize: pageSize,
		TotalPages: (int(total) + pageSize - 1) / pageSize,
	}, nil
}

func (s *Service) Get(ctx context.Context, organizationID, monitorID string) (Monitor, error) {
	params, err := idAndOrgParams(organizationID, monitorID)
	if err != nil {
		return Monitor{}, err
	}
	monitor, err := s.queries.GetMonitorByIDAndOrg(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return Monitor{}, ErrNotFound
	}
	if err != nil {
		return Monitor{}, fmt.Errorf("get monitor: %w", err)
	}
	return monitorFromModel(monitor), nil
}

func (s *Service) Update(ctx context.Context, organizationID, monitorID string, input UpdateInput) (Monitor, error) {
	current, err := s.Get(ctx, organizationID, monitorID)
	if err != nil {
		return Monitor{}, err
	}
	if input.Name != nil {
		current.Name = *input.Name
	}
	if input.URL != nil {
		current.URL = *input.URL
	}
	if input.IntervalSeconds != nil {
		current.IntervalSeconds = *input.IntervalSeconds
	}
	if input.TimeoutMS != nil {
		current.TimeoutMS = *input.TimeoutMS
	}
	if input.ExpectedStatusMin != nil {
		current.ExpectedStatusMin = *input.ExpectedStatusMin
	}
	if input.ExpectedStatusMax != nil {
		current.ExpectedStatusMax = *input.ExpectedStatusMax
	}
	if input.ShowPublic != nil {
		current.ShowPublic = *input.ShowPublic
	}
	current.Name = strings.TrimSpace(current.Name)
	current.URL = strings.TrimSpace(current.URL)
	if err := validateConfiguration(current.Name, current.URL, current.IntervalSeconds, current.TimeoutMS, current.ExpectedStatusMin, current.ExpectedStatusMax); err != nil {
		return Monitor{}, err
	}
	params, _ := idAndOrgParams(organizationID, monitorID)
	monitor, err := s.queries.UpdateMonitor(ctx, sqlc.UpdateMonitorParams{
		ID: params.ID, OrganizationID: params.OrganizationID, Name: current.Name, Url: current.URL,
		IntervalSeconds: int32(current.IntervalSeconds), TimeoutMs: int32(current.TimeoutMS),
		ExpectedStatusMin: int32(current.ExpectedStatusMin), ExpectedStatusMax: int32(current.ExpectedStatusMax),
		ShowPublic: current.ShowPublic,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Monitor{}, ErrNotFound
	}
	if err != nil {
		return Monitor{}, fmt.Errorf("update monitor: %w", err)
	}
	return monitorFromModel(monitor), nil
}

func (s *Service) SetEnabled(ctx context.Context, organizationID, monitorID string, enabled bool) (Monitor, error) {
	params, err := idAndOrgParams(organizationID, monitorID)
	if err != nil {
		return Monitor{}, err
	}
	monitor, err := s.queries.SetMonitorEnabled(ctx, sqlc.SetMonitorEnabledParams{
		ID: params.ID, OrganizationID: params.OrganizationID, Enabled: enabled,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Monitor{}, ErrNotFound
	}
	if err != nil {
		return Monitor{}, fmt.Errorf("set monitor enabled: %w", err)
	}
	return monitorFromModel(monitor), nil
}

func (s *Service) Delete(ctx context.Context, organizationID, monitorID string) error {
	params, err := idAndOrgParams(organizationID, monitorID)
	if err != nil {
		return err
	}
	rows, err := s.queries.DeleteMonitorByIDAndOrg(ctx, sqlc.DeleteMonitorByIDAndOrgParams{ID: params.ID, OrganizationID: params.OrganizationID})
	if err != nil {
		return fmt.Errorf("delete monitor: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func idAndOrgParams(organizationID, monitorID string) (sqlc.GetMonitorByIDAndOrgParams, error) {
	organizationUUID, organizationErr := uuid.Parse(organizationID)
	monitorUUID, monitorErr := uuid.Parse(monitorID)
	if organizationErr != nil || monitorErr != nil {
		return sqlc.GetMonitorByIDAndOrgParams{}, ErrNotFound
	}
	return sqlc.GetMonitorByIDAndOrgParams{
		ID: pgtype.UUID{Bytes: monitorUUID, Valid: true}, OrganizationID: pgtype.UUID{Bytes: organizationUUID, Valid: true},
	}, nil
}

func monitorFromModel(monitor sqlc.Monitor) Monitor {
	result := Monitor{
		ID: uuid.UUID(monitor.ID.Bytes).String(), Name: monitor.Name, URL: monitor.Url,
		IntervalSeconds: int(monitor.IntervalSeconds), TimeoutMS: int(monitor.TimeoutMs),
		ExpectedStatusMin: int(monitor.ExpectedStatusMin), ExpectedStatusMax: int(monitor.ExpectedStatusMax),
		Enabled: monitor.Enabled, ShowPublic: monitor.ShowPublic, Status: monitor.Status,
		CreatedAt: monitor.CreatedAt.Time, UpdatedAt: monitor.UpdatedAt.Time,
	}
	if monitor.LastCheckAt.Valid {
		value := monitor.LastCheckAt.Time
		result.LastCheckAt = &value
	}
	if monitor.LastLatencyMs.Valid {
		value := int(monitor.LastLatencyMs.Int32)
		result.LastLatencyMS = &value
	}
	return result
}
