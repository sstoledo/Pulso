package incidents

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/store/sqlc"
)

type Incident struct {
	ID         string     `json:"id"`
	MonitorID  string     `json:"monitor_id"`
	OpenedAt   time.Time  `json:"opened_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
	DurationMS *int64     `json:"duration_ms"`
	Reason     string     `json:"reason"`
	Status     string     `json:"status"`
}

type ListResult struct {
	Items      []Incident `json:"items"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

type Service struct {
	queries *sqlc.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{queries: sqlc.New(pool)}
}

func (s *Service) List(ctx context.Context, organizationID, monitorID string, page, pageSize int) (ListResult, error) {
	if page < 1 || pageSize < 1 || pageSize > 50 {
		return ListResult{}, &ValidationError{Fields: map[string]string{"pagination": "Page must be at least 1 and page size between 1 and 50."}}
	}
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return ListResult{}, fmt.Errorf("parse organization ID: %w", err)
	}
	orgID := pgtype.UUID{Bytes: organizationUUID, Valid: true}
	if monitorID == "" {
		total, err := s.queries.CountIncidentsByOrg(ctx, orgID)
		if err != nil {
			return ListResult{}, fmt.Errorf("count incidents: %w", err)
		}
		rows, err := s.queries.ListIncidentsByOrg(ctx, sqlc.ListIncidentsByOrgParams{
			OrganizationID: orgID, Limit: int32(pageSize), Offset: int32((page - 1) * pageSize),
		})
		if err != nil {
			return ListResult{}, fmt.Errorf("list incidents: %w", err)
		}
		return incidentListResult(rowsToIncidents(rows), total, page, pageSize), nil
	}
	monitorUUID, err := uuid.Parse(monitorID)
	if err != nil {
		return ListResult{}, &ValidationError{Fields: map[string]string{"monitor_id": "Enter a valid monitor ID."}}
	}
	params := sqlc.CountIncidentsByMonitorAndOrgParams{OrganizationID: orgID, MonitorID: pgtype.UUID{Bytes: monitorUUID, Valid: true}}
	total, err := s.queries.CountIncidentsByMonitorAndOrg(ctx, params)
	if err != nil {
		return ListResult{}, fmt.Errorf("count monitor incidents: %w", err)
	}
	rows, err := s.queries.ListIncidentsByMonitorAndOrg(ctx, sqlc.ListIncidentsByMonitorAndOrgParams{
		OrganizationID: params.OrganizationID, MonitorID: params.MonitorID,
		Limit: int32(pageSize), Offset: int32((page - 1) * pageSize),
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("list monitor incidents: %w", err)
	}
	return incidentListResult(rowsToIncidents(rows), total, page, pageSize), nil
}

func rowsToIncidents(rows []sqlc.Incident) []Incident {
	items := make([]Incident, 0, len(rows))
	for _, row := range rows {
		item := Incident{
			ID: uuid.UUID(row.ID.Bytes).String(), MonitorID: uuid.UUID(row.MonitorID.Bytes).String(),
			OpenedAt: row.OpenedAt.Time, Reason: row.Reason, Status: "OPEN",
		}
		if row.ResolvedAt.Valid {
			value := row.ResolvedAt.Time
			item.ResolvedAt = &value
			item.Status = "RESOLVED"
		}
		if row.DurationMs.Valid {
			item.DurationMS = &row.DurationMs.Int64
		}
		items = append(items, item)
	}
	return items
}

func incidentListResult(items []Incident, total int64, page, pageSize int) ListResult {
	return ListResult{Items: items, Total: int(total), Page: page, PageSize: pageSize, TotalPages: (int(total) + pageSize - 1) / pageSize}
}
