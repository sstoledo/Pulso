# Pulso — Incidents and notifications
Base: `main` at `35fe670`. Branch stack: `feat/incident-persistence` → `feat/incident-transitions` → `feat/notification-worker` → `feat/incident-api` → `docs/incident-alerts`, ultimately targeting `main` under the short-lived stacked-branch and Summary-only PR policies.
Test runners: Go unit/PostgreSQL concurrency tests, fake SMTP tests, Mailpit/API runtime probes, Vitest/build and CI.

## Specs

S1. "Un incidente se abre después de dos chequeos fallidos consecutivos.", "No se abre otro incidente mientras exista uno abierto para el mismo monitor.", "Un chequeo exitoso posterior cierra el incidente.", "Se registra la fecha de apertura, recuperación, duración y motivo resumido.", and "PostgreSQL debe impedir más de un incidente abierto por monitor." Acceptance: first failure creates no incident, second opens exactly one, further failures preserve it, success closes it with duration and safe reason, and concurrent probes cannot create duplicate open incidents.

S2. "Las transiciones se realizan mediante transacciones y bloqueos adecuados." Acceptance: result insertion, monitor state, incident transition, outbox creation and job completion commit or roll back together; monitor and open-incident locks plus a partial unique index protect the invariants.

S3. "Correo al abrirse un incidente.", "Correo cuando el servicio se recupera.", "Claves de idempotencia para evitar alertas duplicadas.", "Registro de intentos y errores.", and "Reintentos con backoff y una cantidad máxima de intentos." Acceptance: OPENED and RECOVERED jobs are unique per incident, due jobs are claimed with SKIP LOCKED, SMTP happens outside database transactions, attempts and errors persist, retries use capped exponential backoff and stop after five attempts, and Mailpit receives both messages without duplicates during normal retries.

S4. "Consulta de incidentes." and backend organization isolation. Acceptance: authenticated members can list monitor/organization incidents with pagination, safe fields and consistent errors; non-members receive 403 and foreign monitor filters return empty results rather than another organization's data.

S5. The main demonstration is "registrar un usuario, crear un monitor, simular una caída, comprobar la apertura del incidente y la alerta, restaurar el servicio y verificar el cierre del incidente." Acceptance: the full Compose flow produces one open incident and OPENED email, then one resolved incident with duration and one RECOVERED email, with no duplicate incidents or notification jobs.

## Tasks

- [x] T1a (S1, S2) — inline on `feat/incident-persistence` — incident schema, transition queries and unit rules — commit: `6947773`.
- [x] T1b (S3) — inline on `feat/notification-persistence` — outbox, idempotency, claim/retry and attempt-history schema — commit: recording after commit.
- [x] T2 (S1, S2) — inline on `feat/incident-transitions` — worker transactional incident/open-close integration and concurrency tests — commit: recording after commit.
- [x] T3a (S3) — inline on `feat/notification-worker` — outbox dispatcher, idempotency, attempt logging, backoff and fake-mailer tests — commit: `34ff145`.
- [x] T3b (S3) — inline on `feat/smtp-mailer` — SMTP client, worker runtime configuration and Mailpit integration — commit: recording after commit.
- [x] T4 (S4) — inline on `feat/incident-api` — paginated incident queries and organization/member API tests — commit: recording after commit.
- [ ] T5 (S5) — inline on `docs/incident-alerts` — OpenAPI, README and full Mailpit demo — commit: pending.
- [ ] T6 (S1-S5) — inline checks plus delegated independent verification for high-risk items 1 (schema/transactions), 2 (tenant/credentials), and 4 (concurrency/idempotency) — full feature verification — evidence: pending.

## Log

L1. 2026-10-28 — "sigue"

L2. Delivery 3 PRs #39-#48 were merged and local/remote branches cleaned before this feature. Every deliverable ends with pushed Summary-only PRs; branches are deleted after the full stack merges.

L3. Notification delivery is at-least-once across process crashes because SMTP has no distributed transaction with PostgreSQL. Idempotent outbox keys prevent duplicate jobs and normal retry duplicates; crash-after-send is documented rather than claiming exactly-once delivery. Notification backoff is capped at 60 seconds and uses at most five attempts. No incident/notification behavior is added to public status pages in this delivery.

L4. T1 evidence: transition tests were RED before implementation and now cover no-open on first failure, open on second, no duplicate open, close on recovery and no close without an incident. Migrations `00006` and `00007` were split into incident and notification persistence PRs under 400 lines each. The partial unique open-incident index, idempotency-key uniqueness, outbox claiming/retry fields and attempt history are generated and tested.

L5. T2 evidence: worker integration now asserts zero incidents after the first failure, one open incident and one OPENED job after the second, and one resolved incident with duration plus a second RECOVERED job after recovery. Fresh migrations 6 and 7 applied; all executor, scheduler and worker tests passed.

L6. T3a evidence: fake-mailer integration passes one successful send with no duplicate dispatch, two failures followed by success on the third attempt, and terminal failure after exactly five attempts. Attempt history, status, retry counts and idempotent dispatch are asserted against real PostgreSQL.

L7. T3b evidence: added a deadline-aware SMTP client and runtime configuration for host, port and sender. The worker now runs check and notification loops concurrently with shared cancellation. Go tests, vet and Compose validation passed; actual Mailpit delivery remains the full-demo verification step.

L8. T4 evidence: authenticated incident listing returns safe fields, OPEN/RESOLVED status, pagination and monitor filters without URLs or emails. Non-members receive 403 and unknown monitor filters return empty results. The partial unique index rejected the fixture's 11 simultaneous open incidents, so history is created through valid close transitions.
