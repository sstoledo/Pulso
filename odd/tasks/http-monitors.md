# Pulso — HTTP monitor management
Base: `main` at `189b954`. Branch stack: `feat/monitor-persistence` → `feat/monitor-api` → `feat/monitor-dashboard` → `docs/http-monitors`, ultimately targeting `main` under the short-lived stacked-branch policy.
Test runners: Go unit/PostgreSQL integration tests, Vitest, type-check/lint/build, OpenAPI lint and authenticated CRUD runtime probes.

## Specs

S1. "El usuario podrá crear, listar, editar, pausar, reanudar y eliminar monitores." Each monitor has "Nombre.", "URL HTTP o HTTPS.", "Intervalo de chequeo: 30 segundos, 1 minuto o 5 minutos.", "Timeout configurable, con un máximo de 10 segundos.", "Rango de estados HTTP esperados, por defecto 200–399.", "Estado habilitado o pausado.", and "Opción para mostrarlo en la página pública." Acceptance: all fields round-trip through PostgreSQL and the API; defaults are 60 seconds, 5,000 ms, 200–399, enabled and private; names are 1–100 characters, timeout is 100–10,000 ms, URLs reject credentials/fragments and non-HTTP(S) schemes.

S2. Monitor states are "UNKNOWN: todavía no existen chequeos.", "UP: el servicio está disponible.", and "DOWN: el servicio alcanzó el umbral de fallos consecutivos." Acceptance: new monitors start UNKNOWN with no latency/check timestamp; this delivery does not fabricate UP/DOWN or execute checks. Pausing is independent of health status.

S3. "Todos los recursos pertenecen a una organización. Cada operación debe comprobar que el usuario pertenece a la organización correspondiente." Acceptance: every monitor query filters by organization, shared backend membership middleware protects every route, another organization's monitor returns 404, and an organization the user does not belong to returns 403.

S4. The dashboard includes "Resumen de servicios UP, DOWN y UNKNOWN.", "Tabla de monitores con estado, última latencia y último chequeo.", "Formulario de creación y edición con validaciones.", "Opciones para pausar, reanudar y eliminar monitores.", "Paginación.", "Estados de carga, vacío y error.", and "Actualización mediante polling cada 15 segundos." Acceptance: authenticated users can complete all monitor operations through the SPA with accessible forms and observable server state.

S5. "Las respuestas de error tendrán un formato consistente y no mostrarán información interna sensible." and "OpenAPI para documentación de la API." Acceptance: validation/not-found/forbidden errors use the existing safe envelope; monitor contracts and dashboard usage are documented; PostgreSQL integration runs in CI.

## Tasks

- [x] T1 (S1, S2) — inline on `feat/monitor-persistence` — monitor schema, sqlc queries, domain validation and unit tests — commit: recording after commit.
- [ ] T2 (S1-S3, S5) — inline on `feat/monitor-api` — shared organization middleware, CRUD/pause/resume/delete API and integration tests — commit: pending.
- [ ] T3 (S4) — inline on `feat/monitor-dashboard` — TanStack Query monitor management UI, forms, polling, pagination and tests — commit: pending.
- [ ] T4 (S5) — inline on `docs/http-monitors` — OpenAPI, README and verification documentation — commit: pending.
- [ ] T5 (S1-S5) — inline checks plus delegated independent verification for high-risk items 1 (schema) and 2 (tenant authorization) — full feature verification — evidence: pending.

## Log

L1. 2026-10-27 — "si, ahora si seguimos con la segunda entrega. por otro lado, para las prs, el body solo tiene que tener un summary y nada mas, si el pr lo requiere alguna cosa mas, pero los otros puntos que has estado poniendo no me llama la atencion para nada la verdad, lo veo como mucho ruido. es una regla mia que la debes aplicar si o si desde ahora en adelante"

L2. Permanent PR presentation rule: future PR bodies contain only a short Summary, plus content strictly required by that specific PR. No unsolicited diagrams, checklists or decorative sections. Permanent branch rule: delete local and remote branches after an independent PR or full stack merges, leaving only main.

L3. URL validation in this delivery is syntactic and rejects credentials. SSRF DNS/IP/redirect enforcement belongs to the check executor in Delivery 3; no monitor execution behavior is claimed here. Delete is a confirmed hard delete for the MVP, with dependent data protected by foreign-key cascades. Monitor lists use page/page_size pagination (default 1/10, maximum 50). RDD remains globally disabled; independent verification remains required for the named high-risk items.

L4. T1 evidence: validation tests were RED because `internal/monitors` had no implementation; a fragment-specific failure revealed that `url.ParseRequestURI` did not preserve the rejected fragment, so parsing uses `url.Parse` and the full suite is GREEN. Migration `00003` adds monitor configuration, UNKNOWN/UP/DOWN checks, scheduling fields, foreign keys and organization/scheduling indexes. sqlc generation, Go tests/vet and `git diff --check` passed.
