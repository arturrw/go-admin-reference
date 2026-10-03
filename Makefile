.PHONY: dev-api dev-web web build run test lint clean

# Run the API with live data on :8080 (frontend served by Vite in dev).
dev-api:
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
	./bin/server

test:
	go test ./...

lint:
	go vet ./...
	cd web && npm run typecheck

clean:
	rm -rf bin web/dist/assets web/dist/index.html web/dist/favicon.svg
