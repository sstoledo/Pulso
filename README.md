# Pulso

Pulso is a multi-organization platform for monitoring APIs, websites, and services. The MVP will provide availability and latency metrics, incident tracking, email alerts, a private dashboard, and public status pages.

The current delivery establishes the executable foundation: Go API and worker processes, PostgreSQL migrations, sqlc generation, a Vue SPA, Mailpit, Docker Compose, and CI quality checks. Business monitoring flows are implemented in later deliveries.

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
make test             # Go and Vue tests
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

The current smoke path starts the stack, applies migrations, verifies API health and readiness, loads the SPA, and opens Mailpit. The complete monitor/incident/alert demonstration will be added with the corresponding MVP deliveries.
