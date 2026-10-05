# GoAdmin reference

A reference admin panel: Go JSON API plus a React SPA, shipped as one binary.
It is not built for a specific business. Use it as a starting point and as a
catalogue of patterns. The domain is a small store with products, orders,
customers, a team and a request log. All of it is seeded fake data.

## Stack

| Layer    | Choice |
| -------- | ------ |
| API      | Go 1.25+, stdlib `net/http` (method + path patterns), `log/slog` |
| Storage  | PostgreSQL 17 via pgx/v5 + sqlc, goose migrations embedded in the binary; in-memory fallback behind the same `httpapi.Store` interface |
| Frontend | React 19, TypeScript, Vite, Tailwind CSS v4 |
| Data     | TanStack Query (server state), TanStack Router (routes, URL-driven sheets) |
| UI       | Hand-rolled shadcn-style primitives, Radix Dialog, cmdk, sonner, lucide icons, Geist fonts |
| Shipping | `web/dist` is embedded with `go:embed`, so the result is one binary |

The design is dark graphite with a white accent (lime and four other accents are one click away in Settings → Appearance). The original clickable mockup
is in [`design/admin-prototype.html`](design/admin-prototype.html).

## Run it

Everything in Docker (Postgres + UI + API on http://localhost:8080):

```bash
docker compose up -d --build --wait     # or: make up
```

The `app` image is a multi-stage build ([`Dockerfile`](Dockerfile)): it builds `web/dist`, embeds
it into a static Go binary and runs on Alpine. Uploads live in the `uploads` volume. `APP_PORT`
changes the host port; `APP_ENV` defaults to `development` (plain-HTTP cookies, demo-login buttons).

For development:

```bash
# 1. Postgres in Docker on :5433 (also creates goadmin_test)
docker compose up -d --wait postgres

# 2. API on :8080. Applies migrations and seeds an empty database on boot.
DATABASE_URL='postgres://goadmin:goadmin@localhost:5433/goadmin?sslmode=disable' go run ./cmd/server

# 3. UI on :5173 with hot reload (proxies /api → :8080)
cd web && npm install && npm run dev
```

Without `DATABASE_URL` the server falls back to the in-memory store, so it runs
without Docker but resets on every restart. The Makefile wraps all of this:
`make db-up dev-api`, `make dev-api-mem`, `make db-reset`.

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
production switches logs to JSON, sets `Secure` cookies and hides the demo accounts), `LOG_LEVEL` (start-up level;
owners and admins can change it at runtime in Settings),
`APP_VERSION`, `DATABASE_URL` (empty = in-memory), `SEED` (default `true`: load demo data into an empty
database), `UPLOAD_DIR` (default `data/uploads`), `SESSION_TTL` (default `12h`). See `.env.example`.

## Layout

```
cmd/server/            entrypoint: config, logger, graceful shutdown
internal/config/       env config
internal/auth/         PBKDF2 password hashing, server-side sessions
internal/media/        upload storage + generated SVG artwork (fallback)
internal/domain/       types, validation, domain errors, shared aggregations (money in cents)
internal/seed/         deterministic demo dataset used by both stores
internal/store/postgres/
  migrations/              goose SQL migrations (embedded, applied on boot)
  queries/                 SQL for sqlc
  db/                      sqlc-generated code — don't edit, run `make sqlc`
  store.go, sessions.go    httpapi.Store / SessionStore on pgxpool
internal/store/memory/ in-memory store (no-dependency mode, quick tests)
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
| GET | `/live` | simulated storefront traffic (2-minute history) + the hottest product: the active, in-stock best sellers compete on a demand signal that drifts minute to minute |
| GET | `/dashboard?range=7\|30\|90` | everything the dashboard renders |
| PUT | `/target` | `{goalCents}`: this quarter's revenue goal; owner only (`workspace:manage`) |
| GET | `/activity` | `?actor&kind&q&limit&offset`: the audit log of staff actions (`team:read`) |
| GET/POST | `/products` | `?q&category&status&sort`; list also returns stats |
| GET | `/products/export` · `/orders/export` · `/customers/export` | CSV with the same filters as the list |
| POST | `/products/import` | CSV (multipart `file` or `text/csv` body), upsert by SKU; all-or-nothing, 422 lists row errors |
| GET/PUT/DELETE | `/products/{id}` | |
| POST | `/products/bulk` | `{ids, action: publish\|archive\|delete}` |
| POST | `/products/{id}/images` | multipart `file`; JPEG/PNG/WebP/GIF ≤ 5 MB (sniffed, SVG rejected) |
| DELETE | `/products/{id}/images/{imageId}` | also removes the file |
| POST | `/products/{id}/images/{imageId}/primary` | make cover image |
| GET | `/orders` | `?q&status&customer&from&to&limit&offset` (`from`/`to` are RFC 3339); returns `items`, `total`, per-status `counts` |
| GET | `/orders/{id}` | |
| PATCH | `/orders/{id}/status` | `{status, reason}`: a refund needs a `reason`, which is kept on the order with who refunded it and when |
| GET | `/customers` | `?q&segment`; also returns segment summary |
| GET | `/customers/{id}` | profile, stats, full order history, products, monthly spend |
| POST | `/customers/{id}/notes` | `{text}` |
| DELETE | `/customers/{id}/notes/{noteId}` | |
| GET/POST | `/team` | `?role` |
| GET | `/team/{id}` | member, effective permissions, `online` (a request in the last 5 minutes) |
| PUT/DELETE | `/team/{id}` | the owner cannot be edited or removed (403) |
| PUT | `/team/{id}/access` | `{granted, revoked}`: per-member exceptions to the role; owner only |
| PUT | `/team/{id}/status` | `{status: active\|suspended}`; suspending signs the member out everywhere |
| GET | `/requests` | `?q&class=2\|4\|5&method&actor&limit`; list + analytics (p50/p95/p99, per-minute, top endpoints) |
| GET | `/requests/{id}` | headers, bodies (redacted), user, route, timing |
| GET/PUT | `/settings/log-level` | `{level: debug\|info\|warn\|error}`; PUT needs `settings:write`, applies until restart |

Plus `GET /healthz` and the public `GET /media/...` image files. Seed products use photos from
[Unsplash](https://unsplash.com/license) (`internal/seed/photos.go`); databases seeded earlier get them on the next start.

## Roles

| Permission | owner | admin | editor | support | viewer |
| ---------- | :---: | :---: | :----: | :-----: | :----: |
| Dashboard | ✓ | ✓ | ✓ | ✓ | ✓ |
| View products / orders / customers | ✓ | ✓ | ✓ | ✓ | ✓ |
| Edit products, upload images | ✓ | ✓ | ✓ | | |
| Change order status, refund | ✓ | ✓ | ✓ | ✓ | |
| Customer notes | ✓ | ✓ | | ✓ | |
| View team, activity log | ✓ | ✓ | ✓ | | |
| Manage team | ✓ | ✓ | | | |
| Request log | ✓ | ✓ | | | |
| Edit settings | ✓ | ✓ | | | |
| Danger zone, quarterly target | ✓ | | | | |

The matrix lives in `internal/domain/team.go`. The API enforces it on every route,
and the UI reads it from `/roles` to hide or disable controls. On top of the role,
the owner can grant or revoke single permissions for one member (Team → member →
Access); the API checks the effective set, role + granted − revoked. A few extra
rules apply: the owner can't be edited, suspended or removed, only the owner can
grant or manage the admin role, the danger zone can't be granted, and nobody can
remove or suspend themselves.

## Database

- **Schema.** See `migrations/00001_init.sql`. Money is stored as `bigint` cents,
  CHECK constraints mirror domain validation, emails are unique case-insensitively,
  and a partial unique index allows exactly one owner.
- **Derived data stays derived.** Customer order count, LTV, last order and
  segment come from the `customers_v` view, not denormalized columns. The view
  mirrors `domain.CustomerSegment`, and a parity test keeps the two in sync.
- **Order lines are snapshots.** `order_items` copy name, price and image, so
  editing or deleting a product doesn't rewrite history (`ON DELETE SET NULL`).
- **Sessions live in Postgres.** Only SHA-256 hashes of the tokens are stored,
  expiry slides forward, and expired rows are purged hourly. Logins survive
  restarts and work across several API instances.
- **Errors map to the domain.** Unique, FK and check violations become 422/404/403
  in `mapErr`, so handlers don't know they're talking to Postgres.
- **New migration.** Add `internal/store/postgres/migrations/0000N_name.sql`
  (goose `-- +goose Up/Down`) and restart. For new queries, edit `queries/*.sql`
  and run `make sqlc`. sqlc needs cgo, so the Makefile runs it in Docker.

## Tests

```bash
go test ./...                     # handler tests on the in-memory store
make test-pg                      # same tests on Postgres + memory/Postgres parity test
cd web && npm run e2e             # Playwright against the in-memory server on :8099
cd web && npm run e2e:pg          # Playwright against a fresh goadmin_e2e database
```

Locally the e2e suite runs on the installed Chrome. With `CI=1` it uses
Playwright's bundled Chromium instead (`npx playwright install chromium`).

## Patterns worth copying

- **Consumer-side interface.** `httpapi.Store` is declared where it is used.
  Postgres and memory implement it, handlers don't care, and one test suite
  covers both.
- **Strict JSON decoding.** Bodies are capped, unknown fields are rejected and
  trailing data is rejected (`decodeJSON`).
- **Domain errors map to HTTP in one place.** See `writeDomainError`.
- **URL-driven sheets.** `/products?edit=12` or `?edit=new` opens the editor,
  so it can be deep-linked and the ⌘K menu can open it.
- **Peek instead of navigate.** Links to a customer, order, product or member
  from another page open its sheet in place (`lib/peek.tsx`). Sheets opened from
  a sheet stack, and closing one returns to the previous.
- **Audit log next to the change.** Handlers call `s.audit(...)` after a write
  succeeds; entries keep the actor and the record they touched, so the feed can
  open it. Logging failures never fail the request.
- **Session auth without dependencies.** PBKDF2 from `crypto/pbkdf2`, opaque
  tokens and an Origin check on writes on top of SameSite cookies.
- **Safe uploads.** The content type is sniffed (not trusted from the client),
  files get random names, SVG is refused and uploads are served with a strict CSP.
- **Request log with redaction.** The access-log middleware tees bodies up to 4 KB
  and masks cookies, auth headers and password or token fields.
- **Accent as a runtime token.** Tailwind v4 `@theme` colors are CSS
  variables, so Settings → Appearance recolors the whole UI live.

## Roadmap

- Object storage (S3) for uploads instead of local disk
- Background job that rebuilds the dashboard rollups from orders
- Password reset and invite acceptance flows, 2FA
- OpenAPI spec and a generated TS client
