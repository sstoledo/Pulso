# Pulso — Identity and tenancy
Base: `2ddd166`. Branch stack: `feat/identity-persistence` → `feat/identity-api` → `feat/web-authentication` → `docs/identity-tenancy` → `fix/identity-password-length`, ultimately targeting `main` under the retained `stacked-to-main` policy.
Test runners: Go unit/integration tests with PostgreSQL, Vitest, type-check/lint/build, and authenticated API/SPA runtime probes.

## Specs

S1. "Registro con nombre, correo y contraseña.", "Creación automática de una organización al registrarse.", and "Usuario creador con rol owner." Acceptance: registration atomically creates the user, organization, and owner membership; normalized email uniqueness and input validation are enforced; no password or hash is returned.

S2. "Inicio y cierre de sesión.", "Contraseñas protegidas con Argon2id.", "Sesiones mediante tokens opacos almacenados hasheados en PostgreSQL.", "Cookies HttpOnly, Secure y con una configuración SameSite adecuada.", and "Endpoint para obtener el usuario autenticado." Acceptance: Argon2id verifies correct passwords and rejects incorrect ones; only random opaque token hashes are persisted; login, logout, and `/me` work; cookies are HttpOnly, SameSite=Lax, Secure outside development, and scoped to `/`; revocation and expiry reject requests.

S3. "Todos los recursos pertenecen a una organización. Cada operación debe comprobar que el usuario pertenece a la organización correspondiente. No será suficiente con ocultar información en el frontend." Acceptance: backend membership middleware protects organization routes; members can read their organization and role; non-members receive 403; another user's session cannot access the organization.

S4. The frontend includes "Registro e inicio de sesión.", "Rutas protegidas.", and "Manejo de sesión expirada." Acceptance: register/login forms validate and submit; session restoration completes before protected navigation; unauthenticated access redirects to login with a safe return path; logout and any API 401 clear authentication; the monitors route is accessible after authentication.

S5. "Las respuestas de error tendrán un formato consistente y no mostrarán información interna sensible." and "OpenAPI para documentación de la API." Acceptance: API errors use `{ "error": { "code", "message", "details?" } }`, credentials and database details never appear in responses, auth/organization contracts are documented, and README/CI cover the new integration checks.

## Tasks

- [x] T1 (S1, S2) — inline on `feat/identity-persistence` — identity schema, sqlc queries, Argon2id and opaque-token primitives with unit tests — commit: `573d01c`.
- [x] T2 (S1-S3, S5) — inline on `feat/identity-api` — transactional registration, session endpoints, membership authorization, consistent errors and PostgreSQL/API tests — commits: `6eee6ed`, `3517093`.
- [x] T3 (S4) — inline on `feat/web-authentication` — Pinia session lifecycle, router guards, register/login/logout UI and tests — commit: `3af4c6f`.
- [x] T4 (S5) — inline on `docs/identity-tenancy` — OpenAPI, README, Compose/CI integration wiring — commit: `767e497`.
- [x] T5 (S1-S5) — inline checks plus delegated independent verification for high-risk items 1 (schema), 2 (authentication/authorization) and 5 (CI/environment) — full feature verification — evidence: independent PASS after one bounded Unicode-password correction.

## Log

L1. 2026-10-26 — "continua"

L2. Delivery strategy and branch lifecycle are inherited from the user's approved policy in `odd/tasks/mvp-foundation.md`: one branch per feature/module/change, PRs stacked to main, and no stacked branch deletion until the whole stack merges. Forecast: approximately 1,800–2,600 authored lines across reviewable units. Passwords use Argon2id with versioned parameters; raw session tokens exist only in the client cookie and process memory. MVP session lifetime is seven days; password length is 12–128 characters; registration names are 1–80 characters. These are implementation defaults within the requested security scope.

L3. RDD remains globally disabled, so no native review will be started. Independent verification remains mandatory for the named high-risk items. Push, PR creation, merge, and branch deletion remain user decisions.

L4. T1 evidence: password/token tests were RED because `internal/auth` had no implementation. After implementation, `go test ./internal/auth` passed correct/incorrect password verification, unique salts, malformed hashes, 32-byte opaque tokens, deterministic SHA-256 hashes, and uniqueness. Migration `00002` adds users, memberships and hashed sessions with foreign keys/uniqueness/checks; sqlc generation and the full Go test/vet commands passed. Runtime migration and transaction behavior remain T2 integration scope.

L5. T2 evidence: service, HTTP and error-envelope tests were RED before implementation. Migration 2 applied to real PostgreSQL. `PULSO_TEST_DATABASE_URL=... go test -count=1 ./internal/auth` passed registration atomicity, owner membership, normalized login, hashed storage, expiry/revocation, cross-organization rejection, origin checks, secure/development cookies, login/logout/me and consistent errors. `make check` and `git diff --check` passed. A test-helper defect initially deleted identities before assertions; it was corrected to register cleanup with `t.Cleanup`, after which the unmodified production transaction passed. OpenAPI auth and organization contracts accompany the API change.

L6. T3 evidence: store, router and login tests were RED before implementation; subsequent registration and API-client tests cover the same public rules. The full `make check` passed with 6 frontend test files and 12 tests, including session restoration/expiry, safe return paths, registration/login validation and 401 publication. A full Compose build passed register 201 through the Vite proxy, owner `/me` and organization responses, cross-organization 403, logout 204 and post-logout `/me` 401. Test identities were deleted and Compose stopped without removing volumes.

L7. T4 evidence: CI now starts PostgreSQL 17, applies migrations, and exposes the test database to `make check`. `make test-integration`, integration-enabled `make check`, `make generate-check`, and `make compose-config` passed. README documents registration, session/cookie policy, production origin/Secure settings, executable curl examples, and integration commands. Redocly initially failed missing servers/security declarations; after correction the document is valid. Four non-blocking recommended-rule warnings remain: no license declared, a localhost server URL, and no artificial 4xx responses for liveness/readiness.

L8. Independent item-1/item-2/item-5 verification at `767e497` returned FAIL for S1 only: a valid 128-character Unicode password was rejected with 422 because backend `len` counted UTF-8 bytes. S2-S5 passed, including credential secrecy, invalid-credential parity, authorization, frontend flow, OpenAPI and state preservation. Reopen only T2 for one bounded correction. Align frontend UTF-16 `.length` with the same Unicode-code-point rule. Scratch `/tmp/opencode/pulso-identity-verify.c8Pxea`, Compose project `pulso-identity-verify-c8pxea`, and its images await parent cleanup.

L9. Correction evidence: backend `é` boundaries and frontend 128-emoji registration were RED before the fix. Go now uses `utf8.RuneCountInString` and Vue uses `Array.from`, both counting Unicode code points. Focused tests are GREEN, real PostgreSQL integration passed, and integration-enabled `make check` passed with 13 frontend tests. An initial migration rerun raced PostgreSQL startup; waiting for `pg_isready` resolved the environment timing without a code change. Scratch resources from the independent verifier were cleaned.

L10. Final verification: the independent bounded recheck at `3517093` returned PASS for S1. The original public HTTP case now returns 201 for exactly 128 `é` characters and the documented 422 envelope for 129. Backend and frontend use the same Unicode-code-point boundary; no S2-S5 reruns were needed after their prior MET verdicts. Parent spot check reran the focused Go boundary test successfully. Scratch directories, isolated Compose resources, and recheck images were cleaned. Delivery 1 is complete; push, PR creation, merge, and branch deletion remain user decisions.
