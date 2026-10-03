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

Sign in with any demo account. The password is `goadmin`, and in development the login page has one-click buttons:

| Account | Role |
| ------- | ---- |
| artur@acme.io (Artur DCS) | owner |
| mark@acme.io | admin |
| yuki@acme.io | editor |
| priya@acme.io | support |
| jon@acme.io | viewer |

Environment variables: `ADDR` (default `:8080`), `APP_ENV` (`development` | `production`;
production switches logs to JSON, sets `Secure` cookies and hides the demo accounts), `LOG_LEVEL`,
`APP_VERSION`, `UPLOAD_DIR` (default `data/uploads`), `SESSION_TTL` (default `12h`).

## Layout

```
cmd/server/            entrypoint: config, logger, graceful shutdown
internal/config/       env config
internal/auth/         PBKDF2 password hashing, server-side sessions
internal/media/        upload storage + generated SVG product artwork
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
Requests without a session get 401, and requests the role doesn't allow get 403.

| Method | Path | Notes |
| ------ | ---- | ----- |
| POST | `/auth/login` · `/auth/logout` | session cookie (HttpOnly, SameSite=Lax) |
| GET | `/auth/me` | current member + permissions |
| GET | `/roles` | role × permission matrix (what the API enforces) |
| GET | `/meta` | version, env, sidebar counters |
| GET | `/runtime` | real Go runtime stats (goroutines, heap, GC); req/s is simulated |
| GET | `/dashboard?range=7\|30\|90` | everything the dashboard renders |
| GET/POST | `/products` | `?q&category&status&sort`; list also returns stats |
| GET/PUT/DELETE | `/products/{id}` | |
| POST | `/products/bulk` | `{ids, action: publish\|archive\|delete}` |
| POST | `/products/{id}/images` | multipart `file`; JPEG/PNG/WebP/GIF ≤ 5 MB (sniffed, SVG rejected) |
| DELETE | `/products/{id}/images/{imageId}` | also removes the file |
| POST | `/products/{id}/images/{imageId}/primary` | make cover image |
| GET | `/orders` | `?q&status&customer&limit&offset`; returns `items`, `total`, per-status `counts` |
| GET | `/orders/{id}` | |
| PATCH | `/orders/{id}/status` | `{status}` |
| GET | `/customers` | `?q&segment`; also returns segment summary |
| GET | `/customers/{id}` | profile, stats, full order history, products, monthly spend |
| POST | `/customers/{id}/notes` | `{text}` |
| GET/POST | `/team` | `?role` |
| PUT/DELETE | `/team/{id}` | the owner cannot be edited or removed (403) |
| GET | `/requests` | `?q&class=2\|4\|5&method&actor&limit`; list + analytics (p50/p95/p99, per-minute, top endpoints) |
| GET | `/requests/{id}` | headers, bodies (redacted), user, route, timing |

Plus `GET /healthz` and the public `GET /media/...` image files.

## Roles

| Permission | owner | admin | editor | support | viewer |
| ---------- | :---: | :---: | :----: | :-----: | :----: |
| Dashboard | ✓ | ✓ | ✓ | ✓ | ✓ |
| View products / orders / customers | ✓ | ✓ | ✓ | ✓ | ✓ |
| Edit products, upload images | ✓ | ✓ | ✓ | | |
| Change order status, refund | ✓ | ✓ | ✓ | ✓ | |
| Customer notes | ✓ | ✓ | | ✓ | |
| View team | ✓ | ✓ | ✓ | | |
| Manage team | ✓ | ✓ | | | |
| Request log | ✓ | ✓ | | | |
| Edit settings | ✓ | ✓ | | | |
| Danger zone | ✓ | | | | |

The matrix lives in `internal/domain/team.go`. The API enforces it on every route,
and the UI reads it from `/roles` to hide or disable controls. A few extra rules
apply: the owner can't be edited or removed, only the owner can grant or manage
the admin role, and nobody can remove themselves.

## Tests

```bash
go test ./...          # handler tests: auth, role matrix, CSRF, uploads, request-log redaction
cd web && npm run e2e  # builds the UI, boots the Go server on :8099, runs Playwright
```

Locally the e2e suite runs on the installed Chrome. With `CI=1` it uses
Playwright's bundled Chromium instead (`npx playwright install chromium`).

## Patterns worth copying

- **Consumer-side interface.** `httpapi.Store` is declared where it is used,
  so a Postgres store can replace `memory.Store` without touching handlers.
- **Strict JSON decoding.** Bodies are capped, unknown fields are rejected and
  trailing data is rejected (`decodeJSON`).
- **Domain errors map to HTTP in one place.** See `writeDomainError`.
- **URL-driven sheets.** `/products?edit=12` or `?edit=new` opens the editor,
  so it can be deep-linked and the ⌘K menu can open it.
- **Session auth without dependencies.** PBKDF2 from `crypto/pbkdf2`, opaque
  tokens and an Origin check on writes on top of SameSite cookies.
- **Safe uploads.** The content type is sniffed (not trusted from the client),
  files get random names, SVG is refused and uploads are served with a strict CSP.
- **Request log with redaction.** The access-log middleware tees bodies up to 4 KB
  and masks cookies, auth headers and password or token fields.
- **Accent as a runtime token.** Tailwind v4 `@theme` colors are CSS
  variables, so Settings → Appearance recolors the whole UI live.

## Roadmap

- Postgres store (pgx + sqlc + goose migrations)
- Persisting sessions and uploads metadata alongside Postgres
- Password reset and invite acceptance flows, 2FA
- OpenAPI spec and a generated TS client
