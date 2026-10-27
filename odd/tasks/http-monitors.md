# Pulso — HTTP monitor management
Base: `main` at `189b954`. Branch stack: `feat/monitor-persistence` → `feat/monitor-api` → `feat/monitor-dashboard` → `docs/http-monitors` → `fix/monitor-validation`, ultimately targeting `main` under the short-lived stacked-branch policy.
Test runners: Go unit/PostgreSQL integration tests, Vitest, type-check/lint/build, OpenAPI lint and authenticated CRUD runtime probes.

## Specs

S1. "El usuario podrá crear, listar, editar, pausar, reanudar y eliminar monitores." Each monitor has "Nombre.", "URL HTTP o HTTPS.", "Intervalo de chequeo: 30 segundos, 1 minuto o 5 minutos.", "Timeout configurable, con un máximo de 10 segundos.", "Rango de estados HTTP esperados, por defecto 200–399.", "Estado habilitado o pausado.", and "Opción para mostrarlo en la página pública." Acceptance: all fields round-trip through PostgreSQL and the API; defaults are 60 seconds, 5,000 ms, 200–399, enabled and private; names are 1–100 characters, timeout is 100–10,000 ms, URLs reject credentials/fragments and non-HTTP(S) schemes.

S2. Monitor states are "UNKNOWN: todavía no existen chequeos.", "UP: el servicio está disponible.", and "DOWN: el servicio alcanzó el umbral de fallos consecutivos." Acceptance: new monitors start UNKNOWN with no latency/check timestamp; this delivery does not fabricate UP/DOWN or execute checks. Pausing is independent of health status.

S3. "Todos los recursos pertenecen a una organización. Cada operación debe comprobar que el usuario pertenece a la organización correspondiente." Acceptance: every monitor query filters by organization, shared backend membership middleware protects every route, another organization's monitor returns 404, and an organization the user does not belong to returns 403.

S4. The dashboard includes "Resumen de servicios UP, DOWN y UNKNOWN.", "Tabla de monitores con estado, última latencia y último chequeo.", "Formulario de creación y edición con validaciones.", "Opciones para pausar, reanudar y eliminar monitores.", "Paginación.", "Estados de carga, vacío y error.", and "Actualización mediante polling cada 15 segundos." Acceptance: authenticated users can complete all monitor operations through the SPA with accessible forms and observable server state.

S5. "Las respuestas de error tendrán un formato consistente y no mostrarán información interna sensible." and "OpenAPI para documentación de la API." Acceptance: validation/not-found/forbidden errors use the existing safe envelope; monitor contracts and dashboard usage are documented; PostgreSQL integration runs in CI.

## Tasks

- [x] T1 (S1, S2) — inline on `feat/monitor-persistence` — monitor schema, sqlc queries, domain validation and unit tests — commits: `2ab36f4`, `26dbe76`.
- [x] T2 (S1-S3, S5) — inline on `feat/monitor-api` — shared organization middleware, CRUD/pause/resume/delete API and integration tests — commits: `7dc05b3`, `26dbe76`.
- [x] T3 (S4) — inline on `feat/monitor-dashboard` — TanStack Query monitor management UI, forms, polling, pagination and tests — commits: `28f1d75`, `26dbe76`.
- [x] T4 (S5) — inline on `docs/http-monitors` — OpenAPI, README and verification documentation — commit: `64c994e`.
- [x] T5 (S1-S5) — inline checks plus delegated independent verification for high-risk items 1 (schema) and 2 (tenant authorization) — full feature verification — evidence: independent PASS after one bounded validation correction.

## Log

L1. 2026-10-27 — "si, ahora si seguimos con la segunda entrega. por otro lado, para las prs, el body solo tiene que tener un summary y nada mas, si el pr lo requiere alguna cosa mas, pero los otros puntos que has estado poniendo no me llama la atencion para nada la verdad, lo veo como mucho ruido. es una regla mia que la debes aplicar si o si desde ahora en adelante"

L2. Permanent PR presentation rule: future PR bodies contain only a short Summary, plus content strictly required by that specific PR. No unsolicited diagrams, checklists or decorative sections. Permanent branch rule: delete local and remote branches after an independent PR or full stack merges, leaving only main.

L3. URL validation in this delivery is syntactic and rejects credentials. SSRF DNS/IP/redirect enforcement belongs to the check executor in Delivery 3; no monitor execution behavior is claimed here. Delete is a confirmed hard delete for the MVP, with dependent data protected by foreign-key cascades. Monitor lists use page/page_size pagination (default 1/10, maximum 50). RDD remains globally disabled; independent verification remains required for the named high-risk items.

L4. T1 evidence: validation tests were RED because `internal/monitors` had no implementation; a fragment-specific failure revealed that `url.ParseRequestURI` did not preserve the rejected fragment, so parsing uses `url.Parse` and the full suite is GREEN. Migration `00003` adds monitor configuration, UNKNOWN/UP/DOWN checks, scheduling fields, foreign keys and organization/scheduling indexes. sqlc generation, Go tests/vet and `git diff --check` passed.

L5. T2 evidence: public CRUD/tenancy tests were RED before the handler existed. Migration 3 applied to real PostgreSQL. `make test-integration` passed auth and monitor suites, including defaults, UNKNOWN state, list/get/update, pause clearing next_check_at, resume scheduling, delete, validation, pagination, 403 for non-members and 404 for foreign monitor IDs within a member organization. Integration-enabled `make check` passed. A test initially mixed the 403/404 boundaries; it was corrected to exercise each through the appropriate organization without changing production behavior.

L6. T3 evidence: dashboard tests were RED because the monitor API did not exist; the real view now passes empty/create, validation, edit/pause/delete, error, 15-second polling and pagination cases. Full `make check` passed with 7 frontend test files and 18 tests. Added a database-backed dashboard summary endpoint so totals are not derived from partial pages. A Compose/Vite smoke passed register, create UNKNOWN with defaults, summary UNKNOWN=1, pause, resume, edit and delete 204. Test identities were cleaned and Compose stopped without removing volumes.

L7. T4 evidence: OpenAPI now documents monitor list/create/get/update/delete, pause/resume and dashboard summary contracts with pagination, defaults, state and error responses. Redocly reports the document valid with the same four known non-blocking recommended warnings (license, localhost server, and health 4xx suggestions). README documents dashboard use, validation, pagination, tenancy, pause/resume/delete and the explicit boundary that worker SSRF/execution belongs to Delivery 3. `make generate-check`, `make check`, `make compose-config`, and `git diff --check` passed.

L8. Independent item-1/item-2 verification at `64c994e` returned FAIL for S1 and S4: explicit zero values incorrectly became defaults and an empty `#` fragment was accepted by backend and frontend. S2, S3 and S5 passed. Reopen only validation-linked T1/T2/T3 for one bounded correction. HTTP input must distinguish omitted numeric fields (defaults) from explicit zero (422), and every rejection must preserve monitor count. Scratch `/tmp/opencode/pulso-http-monitors.x4Gl91`, Compose project `pulsoverify64c994e`, and its images await parent cleanup.

L9. Correction evidence: empty-fragment tests were RED in Go and Vue; the new explicit-zero API cases also covered monitor-count invariants. A pointer-based HTTP create DTO now applies defaults only to omitted fields while explicit zero reaches validation and returns 422. Raw `#` detection closes both backend and frontend empty-fragment bypasses. Focused and real PostgreSQL integration tests passed; integration-enabled `make check` passed with 19 frontend tests. Initial verifier scratch resources were cleaned.

L10. Final verification: the independent bounded recheck at `26dbe76` returned PASS for S1 and S4. All four explicit-zero fields and the empty fragment returned 422 with monitor count unchanged; omitted optional fields retained every default and returned 201 UNKNOWN. Parent spot check reran the Go fragment regression successfully. Prior independent MET verdicts stand for S2, S3 and S5. Scratch directories, isolated Compose resources and recheck images were cleaned. Delivery 2 is complete; push, PR creation, merge and branch deletion remain user decisions under the concise-PR-body and short-lived-branch policies.
