package monitors

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulso/internal/store/sqlc"
)

type Metrics struct {
	Availability24h  *float64      `json:"availability_24h"`
	AverageLatencyMS *float64      `json:"average_latency_ms"`
	P95LatencyMS     *float64      `json:"p95_latency_ms"`
	Series           []MetricPoint `json:"series"`
}

type MetricPoint struct {
	CheckedAt  time.Time `json:"checked_at"`
	LatencyMS  int       `json:"latency_ms"`
	Success    bool      `json:"success"`
	HTTPStatus *int      `json:"http_status"`
}

type MetricsService struct {
	queries *sqlc.Queries
}

func NewMetricsService(pool *pgxpool.Pool) *MetricsService {
	return &MetricsService{queries: sqlc.New(pool)}
}

func (s *MetricsService) Get(ctx context.Context, organizationID, monitorID string) (Metrics, error) {
	params, err := metricParams(organizationID, monitorID)
	if err != nil {
		return Metrics{}, err
	}
	row, err := s.queries.GetMonitorMetrics24h(ctx, params)
	if err != nil {
		return Metrics{}, fmt.Errorf("get monitor metrics: %w", err)
	}
	seriesRows, err := s.queries.ListMonitorMetricSeries24h(ctx, sqlc.ListMonitorMetricSeries24hParams{MonitorID: params.MonitorID, OrganizationID: params.OrganizationID})
	if err != nil {
		return Metrics{}, fmt.Errorf("list monitor metric series: %w", err)
	}
	metrics := Metrics{Series: make([]MetricPoint, 0, len(seriesRows))}
	if row.Total > 0 {
		availability := float64(row.Successes) / float64(row.Total)
		metrics.Availability24h = &availability
	}
	if row.Responses > 0 {
		average := row.AverageLatencyMs
		p95 := row.P95LatencyMs
		metrics.AverageLatencyMS = &average
		metrics.P95LatencyMS = &p95
	}
	for _, point := range seriesRows {
		item := MetricPoint{CheckedAt: point.CheckedAt.Time, LatencyMS: int(point.LatencyMs), Success: point.Success}
		if point.HttpStatus.Valid {
			status := int(point.HttpStatus.Int32)
			item.HTTPStatus = &status
		}
		metrics.Series = append(metrics.Series, item)
	}
	return metrics, nil
}

type CheckResult struct {
	ID           string    `json:"id"`
	CheckedAt    time.Time `json:"checked_at"`
	Success      bool      `json:"success"`
	HTTPStatus   *int      `json:"http_status"`
	LatencyMS    int       `json:"latency_ms"`
	ErrorSummary *string   `json:"error_summary"`
}

type CheckResultList struct {
	Items      []CheckResult `json:"items"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

func (s *MetricsService) ListChecks(ctx context.Context, organizationID, monitorID string, page, pageSize int) (CheckResultList, error) {
	if page < 1 || pageSize < 1 || pageSize > 50 {
		return CheckResultList{}, &ValidationError{Fields: map[string]string{"pagination": "Page must be at least 1 and page size between 1 and 50."}}
	}
	params, err := metricParams(organizationID, monitorID)
	if err != nil {
		return CheckResultList{}, err
	}
	total, err := s.queries.CountCheckResultsByMonitorAndOrg(ctx, sqlc.CountCheckResultsByMonitorAndOrgParams{MonitorID: params.MonitorID, OrganizationID: params.OrganizationID})
	if err != nil {
		return CheckResultList{}, fmt.Errorf("count check results: %w", err)
	}
	rows, err := s.queries.ListCheckResultsByMonitorAndOrg(ctx, sqlc.ListCheckResultsByMonitorAndOrgParams{
		MonitorID: params.MonitorID, OrganizationID: params.OrganizationID,
		Limit: int32(pageSize), Offset: int32((page - 1) * pageSize),
	})
	if err != nil {
		return CheckResultList{}, fmt.Errorf("list check results: %w", err)
	}
	items := make([]CheckResult, 0, len(rows))
	for _, row := range rows {
		item := CheckResult{
			ID: uuid.UUID(row.ID.Bytes).String(), CheckedAt: row.CheckedAt.Time,
			Success: row.Success, LatencyMS: int(row.LatencyMs),
		}
		if row.HttpStatus.Valid {
			status := int(row.HttpStatus.Int32)
			item.HTTPStatus = &status
		}
		if row.ErrorSummary.Valid {
			item.ErrorSummary = &row.ErrorSummary.String
		}
		items = append(items, item)
	}
	return CheckResultList{
		Items: items, Total: int(total), Page: page, PageSize: pageSize,
		TotalPages: (int(total) + pageSize - 1) / pageSize,
	}, nil
}

func metricParams(organizationID, monitorID string) (sqlc.GetMonitorMetrics24hParams, error) {
	organizationUUID, organizationErr := uuid.Parse(organizationID)
	monitorUUID, monitorErr := uuid.Parse(monitorID)
	if organizationErr != nil || monitorErr != nil {
		return sqlc.GetMonitorMetrics24hParams{}, ErrNotFound
	}
	return sqlc.GetMonitorMetrics24hParams{
		MonitorID:      pgtype.UUID{Bytes: monitorUUID, Valid: true},
		OrganizationID: pgtype.UUID{Bytes: organizationUUID, Valid: true},
	}, nil
}
