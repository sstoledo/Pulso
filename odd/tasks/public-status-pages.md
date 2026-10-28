# Pulso — Public status pages
Base: `main` at `e5e4fa8`. Branch stack: `feat/status-page-schema` → `feat/status-page-config` → `feat/public-status-api` → `feat/public-status-ui` → `docs/public-status`, ultimately targeting `main` under the short-lived stacked-branch and Summary-only PR policies.
Test runners: Go unit/PostgreSQL integration, public/private API tests, Vitest, OpenAPI lint and anonymous runtime probes.

## Specs

S1. "Cada organización tendrá un slug público único." and "La página estará disponible en /status/:slug." Acceptance: registration creates a unique status page with a lowercase slug and default title; members can read/update slug, title and publication state; invalid or duplicate slugs are rejected safely.

S2. "No requerirá autenticación." and "Mostrará únicamente los monitores marcados como públicos." Acceptance: the public endpoint requires no cookie, published pages expose only show_public monitors, paused monitors follow their health status, and unpublished/unknown slugs return the same 404 without revealing existence.

S3. "Incluirá nombre, estado actual, disponibilidad de 24 horas e incidentes recientes." Acceptance: each public monitor includes name, UNKNOWN/UP/DOWN status and nullable availability; at most five recent incidents include monitor name, opened/resolved timestamps, duration and OPEN/RESOLVED status, ordered newest first.

S4. "No expondrá URLs, correos, identificadores privados ni configuración interna." Acceptance: public DTOs contain no IDs, monitor URL, email, interval, timeout, expected status, next check, credentials, SQL errors or membership data. Private configuration remains authenticated and organization-scoped.

S5. The SPA public route renders organization/page title, public monitors and incidents with loading/empty/error states. Acceptance: anonymous visitors can use /status/:slug without being redirected to login; private navigation is unchanged, and hidden monitors never appear even through direct API access.

## Tasks

- [x] T1a (S1, S4) — inline on `feat/status-page-schema` — status page schema, unique slugs and registration creation — commit: `bdfeece`.
- [x] T1b (S1, S4) — inline on `feat/status-page-config` — authenticated organization-scoped configuration API — commit: recording after commit.
- [x] T2 (S2-S4) — inline on `feat/public-status-api` — anonymous public DTO, monitor availability and recent incidents with isolation tests — commit: recording after commit.
- [x] T3 (S2, S5) — inline on `feat/public-status-ui` — anonymous /status/:slug view with loading/empty/error states — commit: recording after commit.
- [ ] T4 (S1-S5) — inline on `docs/public-status` — OpenAPI, README and anonymous full-stack demo — commit: pending.
- [ ] T5 (S1-S5) — inline checks plus delegated independent verification for high-risk item 2 (public/private data boundary) — full feature verification — evidence: pending.

## Log

L1. 2026-10-28 — "continua" and "ya los mergee"

L2. Delivery 5 PRs #74-#80 were merged and branches cleaned before this feature. Every deliverable ends with pushed Summary-only PRs and short-lived cleanup after merge.

L3. Public availability uses the same definition as private metrics: successful checks divided by all completed checks in 24 hours and null with no data. Public incident reasons are omitted to avoid leaking internal details. Slugs are 3–64 lowercase alphanumeric/hyphen characters and must not start/end with a hyphen. Public status is not published by default; owners must explicitly publish the page.

L4. T1a evidence: slug generation normalizes organization names and appends the organization UUID prefix, while validation enforces lowercase length and hyphen boundaries. Migration `00008` adds one status page per organization with a unique slug and publication flag. Registration now creates it atomically with user, organization, owner membership and session; integration confirms a valid unpublished slug.

L5. T1b evidence: authenticated GET/PATCH configuration trims and lowercases input, rejects invalid slugs and empty titles with 422, rejects a slug already used by another organization with 409, and returns 403/401 for foreign organizations and anonymous callers. `make check` with a live test database passed, including the new `internal/statuspages` integration suite.

L6. T2 evidence: `GET /api/v1/public/status/{slug}` needs no session and returns title, public monitors with nullable 24-hour availability and at most five newest incidents with monitor name, timestamps, duration and OPEN/RESOLVED. Hidden monitors, their incidents and their check data never appear. The public body contains no monitor URL, identifier, organization identifier, incident reason or monitor configuration; unpublished and unknown slugs return an identical 404. Removing the `show_public` filter produced the expected RED failure before restoring it.

L7. T3 evidence: `/status/:slug` renders title, service cards with status and availability, and the recent-incident table with loading, empty, not-found and generic error states. The route carries no `requiresAuth`, so anonymous visitors are not redirected to login, while private routes keep their guard. Vitest covers authenticated-free rendering, empty states without fabricated availability, 404 handling and generic failure; `make check` passed with 31 frontend tests.
