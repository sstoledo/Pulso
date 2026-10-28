# Pulso — Metrics and monitor detail
Base: `main` at `ed74635`. Branch stack: `feat/metrics-api` → `feat/check-history` → `feat/monitor-detail` → `feat/check-history-ui` → `feat/latency-chart` → `docs/metrics-dashboard` → `fix/metrics-precision-pagination`, ultimately targeting `main` under the short-lived stacked-branch and Summary-only PR policies.
Test runners: Go metric/PostgreSQL tests, Vitest with mocked ECharts, type-check/lint/build, OpenAPI lint and authenticated metric runtime probes.

## Specs

S1. The detail view includes "Estado actual.", "Disponibilidad de las últimas 24 horas.", "Latencia promedio y p95.", "Gráfico de latencia.", "Historial de chequeos.", and "Historial de incidentes." Acceptance: all sections use current server data, organization/monitor authorization, loading/empty/error states and polling every 15 seconds.

S2. Availability is successful checks divided by all completed checks in 24 hours. Acceptance: expected-status successes count, unexpected HTTP statuses and network failures do not; no data returns null, never 100%.
S3. Average and p95 latency are calculated only over checks with an HTTP response, including unexpected statuses. Acceptance: DNS/connection/timeout errors without an HTTP status are excluded from latency aggregates but included in availability; p95 uses PostgreSQL percentile_cont(0.95).
S4. History is paginated and tenant-safe. Acceptance: check results return checked_at, success, nullable HTTP status, nonnegative latency and safe error summary; monitor incidents use the existing safe endpoint and monitor_id filter; another organization's monitor returns 404.
S5. The latency chart reflects the same 24-hour series as the API and is responsive/accessibly labeled. Acceptance: ECharts receives ordered timestamps, latency and success values, no-response checks are not treated as zero latency, and no fabricated series appears when there is no data.

## Tasks

- [x] T1a (S2, S3) — inline on `feat/metrics-api` — 24-hour aggregates and ordered series with integration tests — commits: `a2691d3`, `2afab2c`.
- [x] T1b (S4) — inline on `feat/check-history` — paginated tenant-safe check history — commit: `d78b6ba`.
- [x] T2a (S1, S4) — inline on `feat/monitor-detail` — detail route, summary, polling and UI tests — commit: `cd24ec8`.
- [x] T2b (S1, S4) — inline on `feat/check-history-ui` — check and incident history sections with pagination — commits: `cc78928`, `2afab2c`.
- [x] T3 (S1, S5) — inline on `feat/latency-chart` — responsive ECharts integration and empty/no-response handling — commit: `5c4d0d8`.
- [x] T4 (S1-S5) — inline on `docs/metrics-dashboard` — OpenAPI, README and end-to-end metric demo — commit: recording after commit.
- [x] T5 (S1-S5) — inline checks plus delegated independent verification for high-risk item 2 (tenant authorization) — full feature verification — evidence: independent PASS after one bounded precision/pagination correction.

## Log

L1. 2026-10-28 — "genial, sigamos con esto entonces"

L2. Delivery 4 PRs #58-#66 were merged and local/remote branches cleaned before this feature. Every deliverable ends with pushed Summary-only PRs and short-lived branch cleanup.
L3. This delivery does not add public status pages, SSE or WebSockets. Metrics are computed on demand for the current 24-hour window; no TimescaleDB or historical compaction is introduced. No new monitor execution behavior is claimed beyond what existing results already record.

L4. T1 evidence: no-data metrics return null availability/average/p95 and an empty series. Five results with expected success, unexpected HTTP status and a no-response failure produce availability 0.6, average 250 ms and percentile_cont p95 385 ms. Check history paginates five rows correctly and foreign monitor access returns 404. SQL null handling uses responses=0 to distinguish API null from real zero latency.

L5. T2 evidence: the detail route renders current status, 60.0% availability, 250 ms average, 385 ms p95, check history and incident history with pagination. Empty data shows em dash and no fabricated 100%, errors are visible and monitor/metrics/check/incident queries poll every 15 seconds. Full `make check` passed with 23 frontend tests.

L6. T3 evidence: ECharts receives ordered timestamps, actual latency and success values, including a no-response failure that is not converted to zero. Empty data skips chart initialization; resize and dispose are tested. Full `make check` passed with 26 frontend tests. Component-based ECharts imports reduced the main bundle from 1.28 MB to 656 KB; Vite still reports a non-blocking chunk-size warning.

L7. T4 evidence: OpenAPI documents metrics and check history with nullable aggregates and pagination. Redocly validates with the same four known recommended warnings. README defines availability, average/p95 and history behavior. Runtime demo with two successes and two failures returned availability 0.5, non-null latency aggregates, four series rows, four check-history rows and one resolved incident.

L8. Independent verification returned FAIL for S3 and S4: integer conversion truncated p95 100.95 to 100, and incident history always requested page 1 without navigation. Reopen T1a/T2b for one bounded correction preserving fractional percentile_cont results and independent incident pagination. Scratch `/tmp/opencode/pulso-metrics-verify-SUmymK`, Compose project `pulso-metrics-verify-sumymk`, and images await parent cleanup.

L9. Correction evidence: p95 now remains float64 through the API and OpenAPI, preserving percentile_cont(0.95)=100.95. Incident history has independent previous/next state and navigation, with a regression proving page 2 is requested. Focused tests, full `make check` and `git diff --check` passed. Initial verifier scratch resources were cleaned.

L10. Final verification: independent recheck at `2afab2c` returned PASS for S3 and S4. Fractional p95 is returned exactly as 100.95 and incident Next navigation requests/renders page 2. Parent spot check reran the focused Go precision test successfully. Prior independent MET verdicts stand for S1, S2 and S5. Scratch resources were cleaned. Delivery 5 is ready for Summary-only stacked PR submission with no size exceptions.
