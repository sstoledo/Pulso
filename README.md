# Pulso

Pulso is a multi-organization platform for monitoring APIs, websites, and services. The MVP will provide availability and latency metrics, incident tracking, email alerts, a private dashboard, and public status pages.

The current deliveries establish the executable foundation and identity flow: Go API and worker processes, PostgreSQL migrations, sqlc generation, a Vue SPA, Mailpit, Docker Compose, and CI quality checks. Users can register with an automatically created owner organization, log in, restore and revoke sessions, and access organization-scoped routes through backend membership checks. HTTP monitor management is available: authenticated users can create, list, edit, pause, resume and delete organization-scoped monitors through the API or dashboard. The worker now schedules and executes HTTP checks concurrently, records results and latency, and moves monitors between UNKNOWN, UP and DOWN. The worker now opens incidents after two consecutive failures, closes them after recovery, and sends idempotent email alerts with retries and attempt history. Monitor detail now includes 24-hour availability, average/p95 latency, a latency chart, and paginated check and incident histories. Public status pages are implemented in the next delivery.

## Architecture

- `cmd/api`: HTTP API using Chi and `net/http`.
- `cmd/worker`: background process foundation with graceful shutdown.
- `cmd/migrate`: embedded Goose database migrations.
- `internal`: business and platform modules.
- `db/migrations`: PostgreSQL schema migrations.
- `db/queries`: sqlc source queries.
- `web`: Vue 3, TypeScript, Vite, Pinia, TanStack Query, and Tailwind CSS SPA.
- `openapi`: HTTP API contracts.
- `compose.yaml`: API, worker, frontend, PostgreSQL, migration, and Mailpit services.

## Requirements

- Go 1.27 or newer.
- Node.js 22 and npm.
- Docker with Docker Compose for the local stack.
- Make for the shared quality commands.

## Quick start

1. Create the local environment file:

   ```sh
   cp .env.example .env
   ```

   The example contains development-only values and no production secrets.

2. Build and start the stack:

   ```sh
   docker compose up --build -d
   ```

3. Inspect service state and migration output:

   ```sh
   docker compose ps
   docker compose logs migrate
   ```

4. Open the local services:

   | Service | URL |
   | --- | --- |
   | Frontend | http://localhost:5173 |
   | API liveness | http://localhost:8080/healthz |
   | API readiness | http://localhost:8080/readyz |
   | Mailpit | http://localhost:8025 |
   | PostgreSQL | localhost:55432 |

5. Stop the services without deleting database data:

   ```sh
   docker compose down
   ```

`healthz` checks only API liveness. `readyz` pings PostgreSQL and returns HTTP 503 when the database is unavailable.

## Authentication and tenancy

- Registration requires a 1–80 character name, a valid email, and a 12–128 character password.
- Passwords are hashed with Argon2id. Email addresses are normalized to lowercase.
- Registration atomically creates the user, an organization, and its `owner` membership.
- Sessions expire after seven days. Their opaque tokens are stored only as SHA-256 hashes in PostgreSQL.
- The `pulso_session` cookie is HttpOnly, SameSite=Lax, scoped to `/`, and revoked by logout.
- Organization endpoints always verify backend membership; frontend visibility is not an authorization boundary.
- API errors use an `error` envelope with a stable `code`, safe `message`, and optional validation `details`.

The SPA restores `/api/v1/auth/me` before protected navigation. Unauthenticated users are redirected to login with a local return path. Logout and any API 401 clear the Pinia authentication state. No session token is stored in `localStorage`.

For a deployed environment, set `PULSO_COOKIE_SECURE=true` and configure `PULSO_WEB_ORIGIN` to the exact allowed browser origin. Local development defaults to `http://localhost:5173` and a non-Secure cookie so HTTP localhost works.

Try the API through the Vite proxy after starting the stack:

```sh
email="ada-$(date +%s)@example.com"
curl -c /tmp/pulso.cookies \
  -H 'Content-Type: application/json' \
  -H 'Origin: http://localhost:5173' \
  -d "{\"name\":\"Ada Lovelace\",\"email\":\"$email\",\"password\":\"password-12345\"}" \
  http://localhost:5173/api/v1/auth/register

curl -b /tmp/pulso.cookies http://localhost:5173/api/v1/auth/me

curl -b /tmp/pulso.cookies \
  -H 'Origin: http://localhost:5173' \
  -X POST http://localhost:5173/api/v1/auth/logout
```

The complete request and response contracts are in `openapi/openapi.yaml`.

## Monitor management

Open **http://localhost:5173/monitors** after logging in. The dashboard polls monitor data and UP/DOWN/UNKNOWN totals every 15 seconds.

Each monitor supports:

- A 1–100 character name and an HTTP(S) URL without credentials or fragments.
- Check intervals of 30, 60, or 300 seconds (default 60).
- Timeouts from 100 to 10,000 ms (default 5,000).
- An expected HTTP status range (default 200–399).
- Enabled/paused scheduling and optional public status-page visibility.

New monitors start in `UNKNOWN` state with no check timestamp or latency. Pause clears the next scheduled check; resume schedules it again. Delete is permanent and requires confirmation in the SPA. Monitor lists use `page` and `page_size` (maximum 50), while the summary endpoint always aggregates the whole organization.

Monitor routes are nested under `/api/v1/orgs/{organizationID}` and enforce both backend membership and organization filtering. A non-member organization returns 403; a foreign monitor ID within a member organization returns 404.

## Check execution

The worker creates PostgreSQL-backed jobs for due monitors and claims them with `FOR UPDATE SKIP LOCKED`. Unique scheduling constraints allow one active job per monitor, and a unique result per job makes result effects idempotent. Stale running jobs return to the pending queue.

The pool defaults to 10 concurrent checks and batches of 20. Configure it with `PULSO_WORKER_CONCURRENCY`, `PULSO_WORKER_BATCH_SIZE`, and `PULSO_WORKER_POLL_INTERVAL`. Every check has its own timeout and cancellation context.

Each result stores checked time, success, nullable HTTP status, latency and a short safe error summary. Response bodies are discarded. A successful status within the expected range moves the monitor to UP; the second consecutive failure moves it to DOWN. One failure preserves the previous confirmed state. Pause cancels pending jobs and resume schedules a new one.

The HTTP executor validates every redirect manually, resolves each destination once, connects to the validated IP, preserves the TLS hostname, and blocks loopback, private, link-local, unspecified, multicast and IPv4-mapped internal destinations by default. `PULSO_CHECK_ALLOW_PRIVATE_HOSTS` is an exact-host exception intended only for the local demo. Leave it empty in deployed environments.

### Healthy/down demo

The Compose `demo-target` service listens on `localhost:8081` and internally at `http://demo-target:8080`. Create a monitor with this URL:

```text
http://demo-target:8080/health
```

Use a 30-second interval to observe the worker quickly. Control the target without restarting it:

```sh
# Force failures
curl -H 'Content-Type: application/json' \
  -d '{"healthy":false}' \
  http://localhost:8081/control

# Restore health
curl -H 'Content-Type: application/json' \
  -d '{"healthy":true}' \
  http://localhost:8081/control
```

After two failed checks, the dashboard shows DOWN. After the next healthy check, it shows UP again, with the latest latency and check timestamp. Incidents and email alerts are intentionally introduced in the next delivery.

## Metrics and monitor detail

Open a monitor from the dashboard to inspect current status, availability, average/p95 latency, a 24-hour latency chart, check history and incident history. Data refreshes every 15 seconds.

Metric definitions:

- Availability is successful checks divided by all completed checks in the last 24 hours. No data returns `null`, never 100%.
- Average and p95 latency include only checks with an HTTP response, including unexpected statuses. DNS, connection and timeout failures without a response affect availability but are excluded from latency aggregates.
- p95 uses PostgreSQL `percentile_cont(0.95)`, including interpolation between adjacent values.
- Check history exposes checked time, success, nullable HTTP status, nonnegative latency and a safe error summary, with pagination and organization checks.

Endpoints:

```text
GET /api/v1/orgs/{organizationID}/monitors/{monitorID}/metrics
GET /api/v1/orgs/{organizationID}/monitors/{monitorID}/checks?page=1&page_size=10
```

The latency chart uses the same ordered series as the metrics endpoint and does not convert no-response failures to zero latency.

## Public status pages

Every organization owns one status page created at registration with a unique lowercase slug and an unpublished state. Owners read and update it through authenticated endpoints:

```text
GET   /api/v1/orgs/{organizationID}/status-page
PATCH /api/v1/orgs/{organizationID}/status-page
```

The PATCH body requires `slug`, `title` and `is_published`. Slugs are 3–64 lowercase letters, numbers or hyphens and cannot start or end with a hyphen; a slug used by another organization returns `409`.

Published pages are public and need no session:

```text
GET /api/v1/public/status/{slug}
```

The public view shows the page title, only monitors with `show_public` enabled, their current status and nullable 24-hour availability, and at most five recent incidents with monitor name, opened/resolved timestamps, duration and status. Incident reasons, monitor URLs, identifiers, emails and monitor configuration are never exposed. Unpublished and unknown slugs return the same `404`, so the response does not reveal whether a page exists.

The SPA renders the same data anonymously at `/status/{slug}` with loading, empty, not-found and error states.

### Public status page demo

1. Register, then read the generated slug from `GET /api/v1/orgs/{organizationID}/status-page`.
2. Create a monitor for `http://demo-target:8080/health` and enable `show_public`.
3. Publish the page with `PATCH` using that slug, a title and `is_published` set to `true`.
4. Open `/status/{slug}` in a private window without logging in. The monitor appears with its availability; any hidden monitor stays invisible.
5. Request the public endpoint with `curl` and confirm the body contains no monitor URL, identifier or incident reason.

## Incidents and email alerts

The second consecutive failed check opens one incident and inserts one OPENED outbox job. Further failures preserve that incident. The next successful check resolves it with a duration and inserts one RECOVERED job. Result, monitor state, incident transition and outbox creation commit atomically.

Notifications are claimed with `FOR UPDATE SKIP LOCKED` and sent outside database transactions. Each job has a unique idempotency key, persisted attempt history, capped exponential backoff and at most five attempts. Delivery is at-least-once across process crashes; SMTP and PostgreSQL do not share a distributed transaction, so a crash immediately after a successful send cannot be prevented from producing a retry.

Configure delivery with `PULSO_SMTP_HOST`, `PULSO_SMTP_PORT`, and `PULSO_SMTP_FROM`. Local Compose uses Mailpit on http://localhost:8025.

List private incident history with:

```text
GET /api/v1/orgs/{organizationID}/incidents?page=1&page_size=10&monitor_id={optionalMonitorID}
```

Responses include incident ID, monitor ID, opened/resolved timestamps, duration, safe reason and OPEN/RESOLVED status, never monitor URLs or recipient emails. Organization membership is required.

### Incident and recovery demo

1. Register and create a monitor for `http://demo-target:8080/health` with a 30-second interval.
2. Set the target unhealthy with `curl -H 'Content-Type: application/json' -d '{"healthy":false}' http://localhost:8081/control`.
3. After two failed checks, open the incident API and Mailpit. Exactly one incident and one `[Pulso] Demo Target is DOWN` message should exist.
4. Set the target healthy with `curl -H 'Content-Type: application/json' -d '{"healthy":true}' http://localhost:8081/control`.
5. After the next check, the same incident is resolved with a duration and Mailpit contains one `[Pulso] Demo Target recovered` message. No duplicate incidents or notification jobs are created.

## Local development without containers for the applications

Start PostgreSQL and Mailpit, then run each application in a separate terminal:

```sh
docker compose up -d postgres mailpit
```

```sh
export PULSO_DATABASE_URL='postgres://pulso:pulso@localhost:55432/pulso?sslmode=disable'
go run ./cmd/migrate
go run ./cmd/api
go run ./cmd/worker
```

```sh
cd web
npm ci
npm run dev
```

## Database generation

Apply migrations with `go run ./cmd/migrate` or the Compose `migrate` service. Regenerate typed query code after changing a migration or query:

```sh
make generate
```

The sqlc version is pinned through the Go tool directive in `go.mod`. `make generate-check` fails when generated files differ from the committed output.

## Quality checks

Install frontend dependencies once with `cd web && npm ci`, then run:

```sh
make check
```

This runs Go formatting checks, `go vet`, Go tests, binary builds, Prettier checks, ESLint, Vue type checking, Vitest, and the production frontend build.

Useful focused commands:

```sh
make fmt              # Normalize Go and frontend source formatting
make lint             # Go vet and ESLint
make test             # Go and Vue tests; integration runs when its env is set
make migrate          # Apply migrations to the configured PostgreSQL database
make test-integration # Migrate and run real PostgreSQL auth/tenancy tests
make build            # Go binaries and frontend production build
make compose-config   # Validate Compose configuration
```

## Branch and delivery policy

- Every feature, module, or change uses its own branch.
- Reviewable units use stacked pull requests targeting `main`.
- An independent branch is deleted after its pull request merges.
- Branches in a stack are retained until the entire stack merges, then deleted together.
- Tests and documentation travel with the behavior they cover.

## Demonstration status

The current smoke path starts the stack, applies migrations, registers an owner, creates a monitor, forces two failures to open an incident and send its alert, then restores the target to close the incident and send recovery. Monitor detail exposes 24-hour metrics, a latency chart, check history and incident history. Published status pages serve anonymous visitors at `/status/{slug}` with public monitors and recent incidents only.
