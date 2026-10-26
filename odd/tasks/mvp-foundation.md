# Pulso — MVP foundation
Branch stack: `chore/foundation-go` → `chore/foundation-web` → `chore/foundation-infra` → `docs/foundation-ci`, ultimately targeting `main`. Delivery: `ask-on-risk`, `stacked-to-main`; retain every branch until the whole stack is merged.
Test runners: Go test/vet/build, Vue Vitest/ESLint/type-check/build, Docker Compose config and runtime smoke checks.

## Specs

S1. "API REST con net/http y Chi." Use Go with structured "slog para logs estructurados." Establish separate API and worker processes with configuration and graceful shutdown. Acceptance: both binaries compile and their focused tests pass.

S2. "Vue 3 con Composition API.", "TypeScript.", "Vite.", "Vue Router para navegación y rutas protegidas.", "Pinia para autenticación y estado global de la aplicación.", "TanStack Query para Vue para consultas, caché y actualización de datos remotos.", "Tailwind CSS para estilos." Establish the SPA foundation without implementing business features. Acceptance: the application renders, its focused test passes, and type-check/lint/build succeed.

S3. "Docker Compose." and "Contenedores o servicios para frontend, API, worker, PostgreSQL y Mailpit." Include PostgreSQL migrations and sqlc generation. Acceptance: configuration validates, migrations and generation run, and the composed services pass a runtime smoke check when Docker is available.

S4. "Variables de entorno y archivo .env.example sin secretos." Acceptance: all required local settings are documented with non-secret example values and services fail clearly on invalid configuration.

S5. "Endpoints healthz y readyz." Acceptance: health reports process liveness without dependencies; readiness verifies PostgreSQL and returns an appropriate non-2xx status when unavailable.

S6. "CI básico para formato, lint, compilación y pruebas." Include OpenAPI documentation for the initial health endpoints and README instructions for installation, execution, generation, and tests. Acceptance: local quality commands and CI commands agree; documentation contains no secrets.

## Tasks

- [x] T1 (S1, S4, S5) — inline — Go API/worker foundation, configuration, health/readiness, tests, OpenAPI — commit: `c5a0c88`.
- [x] T2 (S2) — inline — Vue SPA foundation and focused smoke test — commit: `0ce6580`.
- [x] T3 (S1, S3, S4) — inline — PostgreSQL migration/sqlc and Docker Compose services — commit: recording after commit.
- [ ] T4 (S6) — inline — README, quality commands and CI — commit: pending.
- [ ] T5 (S1-S6) — inline checks plus delegated independent verification for high-risk item 5 (delivery/environment) — full feature verification — commit: evidence pending.

## Log

L1. 2026-10-26 — "Genial, empecemos"

L2. The accepted plan defines Delivery 0 as repository setup, configuration, migrations, sqlc, Compose, API/worker/web, PostgreSQL, Mailpit, health checks, and CI. Forecast: approximately 1,200–1,800 authored changed lines. Work units keep behavior, tests, and docs together; item 5 receives an independent feature-end verifier.

L3. 2026-10-26 — "Pr apilados a main, pero ojo cada feature./module/change tieen que ser en una rama separada, cada rama sera eliminada despues de ser mergeado el PR a menos que este en un apilamiento de PRs porque alli tenemos que esperar a que todo se  mergee, pero me entiendes a lo que me refiero"

Decision: one branch per feature/module/change. Independent branches are deleted after their PR merges. Stacked branches are all retained until the entire stack merges, then deleted together. No push, PR, merge, or branch deletion without user authorization.

L4. T1 evidence: observed RED because config and httpx had no implementation; GREEN from `go test ./internal/platform/config ./internal/platform/httpx`; `go test ./...`, `go vet ./...`, and both binary builds passed. Runtime smoke returned `healthz` 200/`{"status":"ok"}`, `readyz` 503/`{"status":"unavailable"}` with PostgreSQL unavailable, and graceful shutdown succeeded. RDD is globally disabled; no native review started.

L5. T2 evidence: observed RED because `App.vue` did not exist; GREEN from `npm test` (1 test). `npm run type-check`, `npm run lint`, `npm run build`, and `npm run format:check` passed. Prettier initially conflicted with Vue formatting rules; `eslint-config-prettier` now disables only overlapping stylistic lint rules while semantic lint remains enabled.

L6. T3 evidence: `go tool sqlc generate` and `docker compose config` were RED before their configuration existed. Generation, Go tests/vet/build, and Compose validation passed afterward. The first runtime attempt built all images but could not bind host port 5432; the default published PostgreSQL port changed to 55432 without investigating or altering the existing host service. A complete rerun passed migration version 1, organizations table presence, API healthz/readyz 200, frontend 200 with the Pulso title, and Mailpit 200. `docker compose down` completed without removing volumes.
