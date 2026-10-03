# GoAdmin reference

A reference admin panel: Go JSON API plus a React SPA, shipped as one binary.
It is not built for a specific business. Use it as a starting point and as a
catalogue of patterns. The domain is a small store with products, orders,
customers, a team and a request log. All of it is seeded fake data.

## Stack

| Layer    | Choice |
| -------- | ------ |
| API      | Go 1.25+, stdlib `net/http` (method + path patterns), `log/slog` |
| Storage  | In-memory store behind the `httpapi.Store` interface (Postgres + pgx + sqlc planned) |
| Frontend | React 19, TypeScript, Vite, Tailwind CSS v4 |
| Data     | TanStack Query (server state), TanStack Router (routes, URL-driven sheets) |
| UI       | Hand-rolled shadcn-style primitives, Radix Dialog, cmdk, sonner, lucide icons, Geist fonts |
| Shipping | `web/dist` is embedded with `go:embed`, so the result is one binary |

The design is dark graphite with a lime accent. The original clickable mockup
is in [`design/admin-prototype.html`](design/admin-prototype.html).

## Run it

```bash
# 1. API on :8080
go run ./cmd/server

# 2. UI on :5173 with hot reload (proxies /api → :8080)
cd web && npm install && npm run dev
```

Single-binary build:

```bash
cd web && npm ci && npm run build && cd ..
go build -o bin/server ./cmd/server
./bin/server        # UI + API on http://localhost:8080
```

Environment variables: `ADDR` (default `:8080`), `APP_ENV` (`development` | `production`;
production switches logs to JSON), `LOG_LEVEL`, `APP_VERSION`.

## Layout

```
cmd/server/            entrypoint: config, logger, graceful shutdown
internal/config/       env config
internal/domain/       types, validation, domain errors (money in cents)
internal/store/memory/ in-memory store + deterministic fake-data seed
internal/reqlog/       ring buffer behind the Request log page
internal/httpapi/      routes, middleware (request id, access log, recover), handlers
web/                   Vite app; embed.go serves dist/ with SPA fallback
  src/components/ui        primitives (Button, Sheet, Segmented, Pill…)
  src/components/charts    SVG charts (area, sparkline, donut, heatmap)
  src/components/layout    shell, sidebar, ⌘K command menu
  src/features/<page>/     one folder per page
  src/lib/                 api client, query hooks, formatters, theme
design/                original HTML prototype
```

## API

All endpoints live under `/api/v1` and speak JSON. Errors look like
`{"error": "...", "fields": {...}}`. Validation failures return 422.

| Method | Path | Notes |
| ------ | ---- | ----- |
| GET | `/meta` | version, env, sidebar counters |
| GET | `/runtime` | real Go runtime stats (goroutines, heap, GC); req/s is simulated |
| GET | `/dashboard?range=7\|30\|90` | everything the dashboard renders |
| GET/POST | `/products` | `?q&category&status&sort`; list also returns stats |
| GET/PUT/DELETE | `/products/{id}` | |
| POST | `/products/bulk` | `{ids, action: publish\|archive\|delete}` |
| GET | `/orders` | `?q&status&limit`; also returns per-status counts |
| GET | `/orders/{id}` | |
| PATCH | `/orders/{id}/status` | `{status}` |
| GET | `/customers` | `?q&segment`; also returns segment summary |
| GET/POST | `/team` | `?role` |
| PUT/DELETE | `/team/{id}` | the owner cannot be edited or removed (403) |
| GET | `/requests` | `?q&class=2\|4\|5&limit`; recent API calls from the access-log middleware |

Plus `GET /healthz`.

## Patterns worth copying

- **Consumer-side interface.** `httpapi.Store` is declared where it is used,
  so a Postgres store can replace `memory.Store` without touching handlers.
- **Strict JSON decoding.** Bodies are capped, unknown fields are rejected and
  trailing data is rejected (`decodeJSON`).
- **Domain errors map to HTTP in one place.** See `writeDomainError`.
- **URL-driven sheets.** `/products?edit=12` or `?edit=new` opens the editor,
  so it can be deep-linked and the ⌘K menu can open it.
- **Accent as a runtime token.** Tailwind v4 `@theme` colors are CSS
  variables, so Settings → Appearance recolors the whole UI live.

## Roadmap

- Postgres store (pgx + sqlc + goose migrations)
- Auth: session cookies, roles enforced in middleware
- Server-side pagination for large tables
- OpenAPI spec and a generated TS client
- Tests: handler tests with `httptest`, Playwright smoke tests
