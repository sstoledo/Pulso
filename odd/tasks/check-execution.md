# Pulso — Check execution
Base: `main` at `244e4e9`. Split branch stack: `feat/check-job-queue` → `feat/check-result-store` → `feat/http-executor` → `feat/ssrf-transport` → `feat/check-scheduler` → `feat/check-worker-v2` → `docs/check-execution-v2`, ultimately targeting `main` under the short-lived stacked-branch and Summary-only PR policies.
Test runners: Go unit/httptest/PostgreSQL concurrency tests, frontend regression checks, Docker Compose worker/demo-target runtime probes and CI.

## Specs

S1. "Ejecución automática según el intervalo de cada monitor.", "Límite de chequeos concurrentes.", "Timeout y cancelación por solicitud.", and "Registro de fecha, resultado, código HTTP, latencia y error resumido." Acceptance: due monitors are scheduled, a bounded goroutine pool executes them, each whole check has its monitor timeout and cancellation, and every completed job persists exactly one result with outcome, nullable HTTP status, nonnegative latency and a safe summary.

S2. "PostgreSQL coordinará los trabajos pendientes. Los workers utilizarán FOR UPDATE SKIP LOCKED para reclamar trabajos sin ejecutarlos por duplicado. También existirán restricciones para evitar trabajos duplicados para el mismo monitor y horario." Acceptance: unique `(monitor_id, scheduled_for)` and one-active-job constraints hold, claiming uses short transactions with FOR UPDATE SKIP LOCKED, stale running jobs recover, result insertion is idempotent, and concurrent integration probes cannot produce duplicate results.

S3. "Validación de redirects antes de seguirlos.", "Protección contra SSRF, bloqueando destinos loopback, privados, link-local e internos.", and "No se almacenarán credenciales ni respuestas completas de los servicios monitoreados." Acceptance: initial and redirect destinations resolve through the same SSRF policy; private/loopback/link-local/unspecified/multicast and IPv4-mapped variants are blocked by default; DNS is resolved once per request and the validated IP is used for the connection without breaking TLS hostname verification; response bodies are discarded; a development-only exact-host exception supports the local demo target.

S4. Health states and monitor configuration drive execution: UNKNOWN before checks, UP after success, DOWN after two consecutive failures; first failure preserves the prior confirmed state. Acceptance: success resets consecutive failures, failure increments them, the second consecutive failure reaches DOWN, paused jobs are not claimed, pausing cancels pending jobs, and next scheduling respects monitor intervals without overlapping checks for one monitor.

S5. "Pruebas del worker con servidores HTTP de prueba.", "Pruebas de integración con PostgreSQL.", "Pruebas para impedir incidentes y trabajos duplicados.", and "Datos y endpoints locales para simular servicios sanos y caídos." Acceptance: httptest covers status/timeout/redirect/SSRF behavior, PostgreSQL tests cover claiming/idempotency/state transitions, Compose includes a controllable demo target, and existing API/dashboard behavior shows persisted latest check and latency.

## Tasks

- [x] T1a (S1, S2) — inline on `feat/check-job-queue` — check_jobs schema and scheduling/claim/recovery queries — commit: `c971b7c`.
- [x] T1b (S1, S2) — inline on `feat/check-result-store` — check_results schema and idempotent result/state queries — commit: `83d6d6d`.
- [x] T2a (S1) — inline on `feat/http-executor` — HTTP outcomes, timeout, cancellation and safe summaries — commit: `f107748`.
- [x] T2b (S3) — inline on `feat/ssrf-transport` — SSRF destination validation and IP-pinned transport — commit: `5827920`.
- [x] T2c (S3) — inline on `feat/ssrf-redirects` — manually revalidated redirects and regression coverage — commit: `f8b94f4`.
- [x] T3a (S1-S5) — inline on `feat/check-scheduler` — scheduler, claiming, stale recovery and pause behavior — commit: recording after commit.
- [x] T3b (S1-S5) — inline on `feat/check-result-worker` — transactional results, idempotency and state transitions — commit: `faf536e`.
- [x] T3c (S1-S5) — inline on `feat/check-worker-v2` — bounded worker pool, runtime configuration and concurrency coverage — commit: recording after commit.
- [ ] T4 (S5) — inline on `docs/check-execution-v2` — controllable demo target, README/operations and verification documentation — commit: pending.
- [ ] T5 (S1-S5) — inline checks plus delegated independent verification for high-risk items 1, 2, 4 and 5 — restructured full-feature verification — evidence: pending.

## Log

L1. 2026-10-27 — "Genial, seguimos con la siguiente entrega"

L2. Delivery 2 PRs #24-#28 were merged before this feature. Local and remote branches were cleaned, leaving only main at 244e4e9. Every deliverable must end with pushed branches and Summary-only PRs for user review; branches are deleted after the full stack merges.

L3. This delivery intentionally does not open incidents or send notifications; those behaviors belong to Delivery 4. Check-job completion means an outcome was recorded, not that the target was healthy. Execution is at-least-once across process crashes; unique result rows and transactional completion prevent duplicate state effects. Redirects are followed manually up to five hops with a fresh SSRF-validated connection per hop. RDD remains globally disabled.

L4. T1 evidence: migration `00004` and typed queries add unique scheduled jobs, one active job per monitor, SKIP LOCKED claiming, stale recovery, idempotent result insertion and transactional state-update primitives. A composite FK ties each result to the correct job and monitor. sqlc generation, Go tests/vet and `git diff --check` passed. Runtime claiming, concurrency and transition verification remain T3 scope.

L5. T2 evidence: executor tests were RED before implementation and now pass expected/unexpected/custom statuses, timeout, cancellation, allowed/blocked redirects, loopback/private/link-local/unspecified/IPv4-mapped blocking, safe summaries and TLS with a custom SNI certificate. Each hop creates a fresh proxy-disabled transport to the resolved validated IP with the original hostname for TLS verification. Response bodies are discarded and never stored. A test-only certificate fixture was corrected rather than weakening production TLS verification.

L6. T3 evidence: migration 4 applied to real PostgreSQL. Worker tests pass concurrent claims with one result, idempotent concurrent processing, first/second failure and recovery transitions, stale-job recovery, pause cancellation and a bounded pool of two. A 60-second test investigation proved a fixture counted nine instrumented targets as ten, not a production wait; after correction the full checks suite completes in about one second. Integration packages run with `-p 1` because scheduler/count invariants share one database. Worker configuration adds concurrency, batch size, poll interval and exact private-host exceptions. `make test-integration` and integration-enabled `make check` passed.

L7. T4 evidence: the demo-target health/control contract was RED before implementation and passes afterward. Compose now starts a healthy internal demo target before the worker and uses an exact development-only private-host exception. The full runtime demo passed initial UP (1 result), first failure preserving UP (2), second failure reaching DOWN (3), and recovery to UP (4), with non-null latest latency/check timestamp. README documents worker operations, SSRF defaults, configuration and healthy/down controls. `make generate-check`, `make check`, `make compose-config`, and `git diff --check` passed.

L8. Independent item-1/item-2/item-4/item-5 verification at `d8167bb` returned FAIL for S1 and S3: response-body disposal ignored whole-check timeout errors, CGNAT/benchmark/reserved ranges passed, and production accepted private-host exceptions. S2, S4 and S5 passed. Reopen T2 for one bounded correction with three regression classes. Production configuration with private exceptions must fail explicitly rather than silently ignoring an option. Scratch `/tmp/opencode/pulso-check-verify.p0UMti`, Compose project `pulsoverify-p0umti`, and images await parent cleanup.

L9. Correction evidence: all three regression classes were RED before the fix. Response-body disposal now propagates errors into safe failed results; additional special-use CIDRs and mapped variants are denied; production private-host configuration fails explicitly at load time. Focused tests, real PostgreSQL integration and integration-enabled `make check` passed. Initial verifier scratch resources were cleaned.

L10. Final verification: the independent bounded recheck at `2020f73` returned PASS for S1 and S3. A flushed 200 with a stalled body fails with `request timeout`; all additional internal/mapped ranges are denied; production private-host configuration fails explicitly with the documented error. Parent spot check reran the focused executor regressions successfully. Prior independent MET verdicts stand for S2, S4 and S5. Scratch resources were cleaned. Delivery 3 is complete and must now be submitted as Summary-only stacked PRs.

L11. 2026-10-28 — User selected "Dividir antes de abrir" for the three oversized PRs. Replace the unpublished five-branch stack with seven cohesive branches: check job queue, check result store, HTTP executor, SSRF transport, scheduler, worker, and demo/docs. Preserve the original verified behavior and regression coverage, then rerun independent verification on the restructured tip before submission. The original branches remain unpublished backups during restructuring.
