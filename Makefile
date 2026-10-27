GO_PACKAGES := ./cmd/... ./db/... ./internal/...

.PHONY: generate generate-check fmt fmt-check lint test build check compose-config

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
	go test $(GO_PACKAGES)
	cd web && npm test

build:
	go build ./cmd/api ./cmd/worker ./cmd/migrate
	cd web && npm run build

check: fmt-check lint test build

compose-config:
	docker compose config --quiet
