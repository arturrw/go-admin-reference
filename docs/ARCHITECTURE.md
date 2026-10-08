# Architecture

[← README](../README.md) · [API reference](API.md) · [Contributing](../CONTRIBUTING.md)

GoAdmin is one Go binary. It serves a JSON API under `/api/v1` and the React
single-page app that is built into it with `go:embed`. State lives in
PostgreSQL, or in memory when no database is configured.

## System overview

```mermaid
flowchart LR
    subgraph Browser
        SPA["React SPA<br/>TanStack Router + Query"]
    end

    subgraph Binary["Go binary (cmd/server)"]
        direction TB
        MW["Middleware<br/>request id → access log → recover → security headers"]
        MUX["net/http ServeMux<br/>method + path patterns"]
        AUTH["authorize()<br/>session → member → Origin check → permission"]
        H["Handlers<br/>internal/httpapi"]
        STATIC["Embedded UI<br/>web/dist, gzipped once"]
        MEDIA["Media<br/>uploads + generated art"]
        RL["Request log<br/>ring buffer"]
        STORE{{"httpapi.Store<br/>interface"}}
        PG["postgres.Store<br/>pgx + sqlc"]
        MEM["memory.Store<br/>seeded, no deps"]
    end

    DB[("PostgreSQL 17<br/>goose migrations")]
    DISK[("Upload dir")]
    CDN["Unsplash CDN<br/>seed photos"]

    SPA -- "fetch /api/v1 (cookie)" --> MW
    SPA -- "GET /, /assets" --> MW
    MW --> MUX
    MW -. records .-> RL
    MUX --> AUTH --> H
    MUX --> STATIC
    MUX --> MEDIA --> DISK
    H --> STORE
    STORE --> PG --> DB
    STORE --> MEM
    SPA -. "img, sized per thumbnail" .-> CDN
```

- **One deployable.** `web/embed.go` serves `web/dist` with an SPA fallback.
  Hashed assets are cached forever, and text files are gzipped once and kept in
  memory.
- **Two stores, one interface.** `httpapi.Store` is declared where it is used.
  `postgres.Store` is the real one. `memory.Store` runs without dependencies. The
  same handler tests run against both, and a parity test compares their output.
- **No framework.** Routing is the stdlib `ServeMux` with method and path
  patterns, and logging is `log/slog`. Auth is opaque session tokens and PBKDF2.

## Request lifecycle

```mermaid
sequenceDiagram
    autonumber
    participant B as Browser
    participant M as Middleware
    participant A as authorize()
    participant H as Handler
    participant S as Store
    participant L as Audit log

    B->>M: PATCH /api/v1/orders/10231/status {status, reason}
    M->>M: X-Request-ID, start timer, tee body (≤ 4 KB, redacted)
    M->>A: route has permission orders:write
    A->>S: session token → member (sliding expiry)
    A->>A: member active? Origin = Host? role + granted − revoked ∋ orders:write?
    A->>S: touch last_active_at (≤ once a minute)
    A->>H: request with member in context
    H->>H: strict JSON decode, domain validation (reason required)
    H->>S: UpdateOrderStatus(id, refunded, refund)
    S-->>H: order (or domain error → 404 / 409 / 422)
    H->>L: audit(refund, order 10231, "refunded … — reason")
    H-->>B: 200 JSON
    M->>M: access log line + request-log entry (status, latency, actor)
```

Every write goes through these steps. The UI then invalidates its cached
queries. A global `MutationCache.onSuccess` refreshes the activity feed and the
dashboard after any change.

## Backend layers

```mermaid
flowchart TB
    subgraph httpapi["internal/httpapi"]
        R["server.go<br/>routes + permissions"]
        HA["resources.go · team.go · activity.go<br/>system.go · csv.go · live.go"]
        MWX["middleware.go · respond.go · auth.go"]
    end
    subgraph domain["internal/domain"]
        T["types, validation, errors<br/>RBAC matrix, segments, targets"]
    end
    subgraph stores["internal/store"]
        P["postgres/<br/>migrations · queries · db (sqlc)"]
        MM["memory/"]
    end
    SEED["internal/seed<br/>deterministic demo data + history"]
    AUX["internal/auth · media · reqlog · config"]

    R --> HA --> T
    HA --> P & MM
    P --> T
    MM --> T
    MM --> SEED
    P -. "SeedIfEmpty / backfills" .-> SEED
    HA --> AUX
```

`domain` imports nothing from the project. Handlers depend on `domain` and the
`Store` interface, never on a concrete store.

## Data model

```mermaid
erDiagram
    members ||--o{ sessions : "signs in with"
    members ||--o{ activity : "acts in"
    products ||--o{ product_images : "has gallery"
    products |o--o{ order_items : "snapshot of"
    customers ||--o{ orders : places
    customers ||--o{ customer_notes : "annotated by staff"
    orders ||--|{ order_items : contains

    members {
        bigint id PK
        text email UK "lower(email)"
        text role "owner|admin|editor|support|viewer"
        text status "active|invited|suspended"
        text_arr granted "owner-set exceptions"
        text_arr revoked
        timestamptz last_active_at "presence"
    }
    sessions {
        bytea token_hash PK "SHA-256 only"
        bigint member_id FK
        timestamptz expires_at "sliding"
    }
    products {
        bigint id PK
        text sku UK
        bigint price_cents
        int stock
        text status "active|draft|archived"
    }
    orders {
        bigint id PK
        bigint customer_id FK
        text status
        bigint total_cents
        text refund_reason "kept with the refund"
        text refunded_by
        timestamptz refunded_at
    }
    order_items {
        bigint order_id PK
        int line PK
        bigint product_id FK "SET NULL on delete"
        bigint price_cents "price at purchase"
    }
    activity {
        bigint id PK
        text kind
        bigint actor_id FK
        text message
        text entity "product|order|customer|member"
        bigint entity_id
    }
    api_keys {
        bigint id PK
        text scope "read|write"
        text last4 "for recognising the key"
        bytea token_hash UK "SHA-256 only"
        timestamptz revoked_at
    }
    targets {
        text quarter PK "2026-Q4"
        bigint goal_cents
        text updated_by
    }
```

Not shown: `customer_notes`, `product_images`, the dashboard rollups
(`revenue_daily`, `orders_heatmap`), and the `customers_v` view, which derives
order count, LTV, last order and segment.

- **Money is `bigint` cents** everywhere, in the API too.
- **Derived data stays derived.** Customer metrics come from `customers_v`. The
  view mirrors `domain.CustomerSegment`, and the parity test keeps them in sync.
- **Order lines are snapshots**, so editing or deleting a product doesn't
  rewrite history. Order images follow the product's current cover while it
  exists.
- **Sessions live in Postgres** as token hashes, with sliding expiry and an
  hourly purge. Logins survive restarts and work across several instances.
- **Errors map to the domain.** In `mapErr`, unique and check violations become
  422 and missing rows become 404, so handlers never see Postgres errors.
- **Migrations** are goose files embedded in the binary and applied on boot:

  | File | Adds |
  | ---- | ---- |
  | `00001_init.sql` | core schema, `customers_v` |
  | `00002_activity_audit.sql` | actor and record links on `activity` |
  | `00003_order_refunds.sql` | refund reason, who and when |
  | `00004_member_access.sql` | per-member `granted` / `revoked` |
  | `00005_targets.sql` | quarterly revenue goals |
  | `00006_api_keys.sql` | server-to-server API keys (hashed) |

- **Seeding and backfills.** An empty database gets the demo dataset. Older
  demo databases get refund reasons and a week of staff history once, on boot
  (with `SEED=true`).

## Access control

```mermaid
flowchart LR
    ROLE["Role permissions<br/>domain.RolePermissions"] --> EFF
    G["+ granted<br/>(owner only)"] --> EFF
    RV["− revoked<br/>(owner only)"] --> EFF
    EFF["Effective permissions<br/>Member.Permissions()"] --> API["authorize() on every route"]
    EFF --> ME["GET /auth/me → UI hides or disables controls"]
```

The owner always has everything, and the danger zone (`workspace:manage`) can't
be granted. Only the owner manages admins, individual exceptions and the
quarterly target. Nobody can remove or suspend themselves. The full matrix is
in the [README](../README.md#roles).

## Frontend

```mermaid
flowchart TB
    MAIN["main.tsx<br/>QueryClient · 401 → login · refetch feed after mutations"]
    ROUTER["router.tsx<br/>guards per permission · lazy page chunks"]
    SHELL["AppShell<br/>sidebar · header · ⌘K (lazy)"]
    PEEK["PeekProvider<br/>customer / order / product / member sheets<br/>stacked, opened in place"]
    PAGES["features/*<br/>dashboard · products · orders · customers<br/>team · activity · requests · settings"]
    UI["components/ui + charts<br/>Sheet · Dialog · Bars · AreaChart · IconTile…"]
    LIB["lib/<br/>api.ts (typed client) · queries.ts (hooks + keys)<br/>auth · peek · format · image · theme"]

    MAIN --> ROUTER --> SHELL --> PAGES
    SHELL --> PEEK
    PAGES --> UI
    PAGES --> LIB
    PEEK --> LIB
```

- **Server state is TanStack Query.** Keys are grouped by resource
  (`['orders', filter]`), so a mutation can invalidate everything that depends
  on it with one prefix.
- **Sheets come in two kinds.** A list page's own sheet is URL-driven
  (`/orders?view=10231`, deep-linkable). Anything opened from another page
  peeks in place, and closing it returns to the previous sheet.
- **Code splitting.** Each page, the peeked sheets and the command menu are
  separate chunks.
- **Design tokens.** Tailwind v4 `@theme` colors are CSS variables, so Settings
  → Appearance recolors the UI live. White is the default accent.
- **Phones.** Below 640px, tables give way to compact rows (orders) or keep
  only the key columns. Wide grids scroll inside their card, never the page. The
  mobile e2e suite enforces this.

## Code layout

```
cmd/server/              entrypoint: config, logger, store selection, backfills, graceful shutdown
internal/config/         env config
internal/auth/           PBKDF2 password hashing, in-memory sessions
internal/domain/         types, validation, errors, RBAC, aggregations (money in cents)
internal/seed/           deterministic demo dataset and activity history
internal/store/postgres/ migrations/ (goose) · queries/ (sqlc input) · db/ (generated) · store.go · sessions.go · seed.go
internal/store/memory/   in-memory store
internal/reqlog/         ring buffer behind the Request log page
internal/media/          upload storage, generated SVG artwork
internal/httpapi/        routes, middleware, handlers, API tests
web/                     Vite app; embed.go serves dist/ (gzip, SPA fallback)
  src/components/          ui primitives, charts, layout
  src/features/<page>/     one folder per page
  src/lib/                 api client, query hooks, peek, formatters, theme
  e2e/                     Playwright specs (desktop + mobile)
  scripts/                 e2e on Postgres, README screenshots
docs/                    architecture, API reference, screenshots
```

## Patterns worth copying

- **Strict JSON decoding.** Bodies are capped, and unknown fields or trailing
  data are rejected (`decodeJSON`).
- **Domain errors map to HTTP in one place** (`writeDomainError`).
- **Audit next to the change.** Handlers call `s.audit(...)` after a write
  succeeds. A failure to log never fails the request.
- **Safe uploads.** The content type is sniffed, files get random names, SVG is
  refused, and uploads are served with a strict CSP.
- **Request log with redaction.** Cookies, auth headers and password or token
  fields are masked before anything is stored.
- **Simulated where there is no source.** Storefront traffic is a deterministic
  function of time, so polls agree. Catalogue data (the hottest product, stock,
  sales) stays real.
