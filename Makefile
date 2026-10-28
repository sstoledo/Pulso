GO_PACKAGES := ./cmd/... ./db/... ./internal/...
PULSO_DATABASE_URL ?= postgres://pulso:pulso@localhost:55432/pulso?sslmode=disable
PULSO_TEST_DATABASE_URL ?= $(PULSO_DATABASE_URL)

.PHONY: generate generate-check fmt fmt-check lint test test-integration migrate build check compose-config

generate:
	go tool sqlc generate

generate-check: generate
	git diff --exit-code -- internal/store/sqlc

fmt:
	gofmt -w cmd db internal
	cd web && npm run format

fmt-check:
	@unformatted="$$(gofmt -l cmd db internal)"; \
	if [ -n "$$unformatted" ]; then \
		printf 'Files need gofmt:\n%s\n' "$$unformatted"; \
		exit 1; \
	fi
	cd web && npm run format:check

lint:
	go vet $(GO_PACKAGES)
	cd web && npm run lint

test:
	go test -p 1 $(GO_PACKAGES)
	cd web && npm test

migrate:
	PULSO_DATABASE_URL="$(PULSO_DATABASE_URL)" go run ./cmd/migrate

test-integration: migrate
	PULSO_TEST_DATABASE_URL="$(PULSO_TEST_DATABASE_URL)" go test -p 1 -count=1 ./internal/auth ./internal/monitors ./internal/checks ./internal/incidents

build:
	go build ./cmd/api ./cmd/worker ./cmd/migrate
	cd web && npm run build

check: fmt-check lint test build

compose-config:
	docker compose config --quiet
