package statuspages

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"pulso/internal/organizations"
)

var ErrPageNotFound = errors.New("status page not found")

type PublicPage struct {
	Title     string           `json:"title"`
	Monitors  []PublicMonitor  `json:"monitors"`
	Incidents []PublicIncident `json:"incidents"`
}

type PublicMonitor struct {
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	Availability24h *float64 `json:"availability_24h"`
}

type PublicIncident struct {
	MonitorName string     `json:"monitor_name"`
	OpenedAt    time.Time  `json:"opened_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	DurationMS  *int64     `json:"duration_ms"`
	Status      string     `json:"status"`
}

func (s *Service) Public(ctx context.Context, slug string) (PublicPage, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if !organizations.ValidStatusSlug(slug) {
		return PublicPage{}, ErrPageNotFound
	}
	page, err := s.queries.GetStatusPageBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicPage{}, ErrPageNotFound
	}
	if err != nil {
		return PublicPage{}, fmt.Errorf("get public status page: %w", err)
	}
	monitorRows, err := s.queries.ListPublicMonitorsByStatusPage(ctx, slug)
	if err != nil {
		return PublicPage{}, fmt.Errorf("list public monitors: %w", err)
	}
	incidentRows, err := s.queries.ListPublicIncidentsByStatusPage(ctx, slug)
	if err != nil {
		return PublicPage{}, fmt.Errorf("list public incidents: %w", err)
	}

	result := PublicPage{
		Title:     page.Title,
		Monitors:  make([]PublicMonitor, 0, len(monitorRows)),
		Incidents: make([]PublicIncident, 0, len(incidentRows)),
	}
	for _, row := range monitorRows {
		monitor := PublicMonitor{Name: row.Name, Status: row.Status}
		if row.TotalChecks > 0 {
			availability := float64(row.SuccessfulChecks) / float64(row.TotalChecks)
			monitor.Availability24h = &availability
		}
		result.Monitors = append(result.Monitors, monitor)
	}
	for _, row := range incidentRows {
		incident := PublicIncident{
			MonitorName: row.MonitorName,
			OpenedAt:    row.OpenedAt.Time,
			Status:      "OPEN",
		}
		if row.ResolvedAt.Valid {
			resolvedAt := row.ResolvedAt.Time
			incident.ResolvedAt = &resolvedAt
			incident.Status = "RESOLVED"
		}
		if row.DurationMs.Valid {
			duration := row.DurationMs.Int64
			incident.DurationMS = &duration
		}
		result.Incidents = append(result.Incidents, incident)
	}
	return result, nil
}
