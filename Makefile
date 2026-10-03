.PHONY: db-up db-down db-reset dev-api dev-api-mem dev-web web build run sqlc test test-pg e2e e2e-pg lint clean

DATABASE_URL ?= postgres://goadmin:goadmin@localhost:5433/goadmin?sslmode=disable
TEST_DATABASE_URL ?= postgres://goadmin:goadmin@localhost:5433/goadmin_test?sslmode=disable
SQLC_VERSION ?= 1.29.0

# Postgres in Docker on :5433 (creates goadmin and goadmin_test).
db-up:
	docker compose up -d --wait

db-down:
	docker compose down

# Drops all data; the server re-migrates and re-seeds on next start.
db-reset:
	docker compose exec -T postgres psql -U goadmin -d goadmin -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public"

# API on :8080 backed by Postgres (migrates + seeds an empty database).
dev-api: db-up
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/server

# API with the in-memory store — no Docker needed.
dev-api-mem:
	go run ./cmd/server

# Vite dev server on :5173, proxying /api to :8080.
dev-web:
	cd web && npm run dev

web:
	cd web && npm ci && npm run build

# Single binary with the built UI embedded.
build: web
	go build -trimpath -ldflags="-s -w" -o bin/server ./cmd/server

run: build
	DATABASE_URL="$(DATABASE_URL)" ./bin/server

# Regenerate internal/store/postgres/db from queries/*.sql (sqlc needs cgo, so run it in Docker).
sqlc:
	docker run --rm -v "$(CURDIR):/src" -w /src sqlc/sqlc:$(SQLC_VERSION) generate

test:
	go test ./...

# Handler tests + memory/Postgres parity test against a real database.
test-pg: db-up
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -count=1 ./...

e2e:
	cd web && npm run e2e

e2e-pg: db-up
	cd web && npm run e2e:pg

lint:
	go vet ./...
	cd web && npm run typecheck

clean:
	rm -rf bin web/dist/assets web/dist/index.html web/dist/favicon.svg
