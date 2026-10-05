# API reference

[← README](../README.md) · [Architecture](ARCHITECTURE.md) · [Contributing](../CONTRIBUTING.md)

All endpoints live under `/api/v1` and speak JSON. Money is integer **cents**,
and timestamps are RFC 3339. The UI uses exactly this API, through a typed
client in [`web/src/lib/api.ts`](../web/src/lib/api.ts).

## Conventions

### Authentication

Sessions are opaque tokens in an `HttpOnly`, `SameSite=Lax` cookie
(`goadmin_session`, `Secure` in production). Expiry slides forward on use
(`SESSION_TTL`, 12 h by default).

```bash
# Sign in: stores the cookie in a jar
curl -c jar -H 'Content-Type: application/json' \
  -d '{"email":"artur@acme.io","password":"goadmin"}' \
  http://localhost:8080/api/v1/auth/login

# Use it
curl -b jar http://localhost:8080/api/v1/orders?status=refunded&limit=5
```

Writes (`POST`, `PUT`, `PATCH`, `DELETE`) carrying an `Origin` header must come
from the same host. Browsers always send it, so cross-site requests are
rejected with 403 on top of the SameSite cookie. Clients like `curl` that send
no `Origin` are allowed.

### Permissions

Every route requires one permission, shown in the tables below. A member's
effective permissions are the role's, plus any the owner granted, minus any
the owner revoked. `GET /auth/me` returns them, and `GET /roles` returns the
whole matrix.

`dashboard:read` · `products:read` · `products:write` · `orders:read` ·
`orders:write` · `customers:read` · `customers:write` · `team:read` ·
`team:write` · `requests:read` · `settings:write` · `workspace:manage`

### Requests and errors

- Bodies are JSON objects of at most 1 MB. Unknown fields and trailing data are
  rejected with 400.
- Errors always look like this:

  ```json
  { "error": "validation failed", "fields": { "reason": "a refund needs a reason" } }
  ```

| Status | Meaning |
| ------ | ------- |
| 400 | malformed JSON, bad id or query parameter |
| 401 | no or expired session |
| 403 | missing permission, cross-origin write, or a rule such as "only the owner can change admins" |
| 404 | no such record (or endpoint) |
| 409 | conflicting state, for example refunding an unpaid order or activating a pending invite |
| 413 | body or upload too large |
| 415 | upload is not JPEG, PNG, WebP or GIF |
| 422 | validation failed; `fields` names each problem |

Lists that page take `limit` and `offset`, and return `items` and `total`.
Every response carries an `X-Request-ID` (a client-supplied one is kept).

## Endpoints

### Auth and session

| Method | Path | Permission | Notes |
| ------ | ---- | ---------- | ----- |
| POST | `/auth/login` | public | `{email, password}` → `{user, permissions}`; sets the cookie |
| POST | `/auth/logout` | public | clears the session |
| GET | `/auth/me` | signed in | current member + effective permissions |
| GET | `/auth/demo-accounts` | public, development only | one-click demo logins for the login page |
| GET | `/roles` | signed in | `{roles, permissions, matrix}`, the matrix the API enforces |
| GET | `/meta` | signed in | version, env, sidebar counters |

### Dashboard

| Method | Path | Permission | Notes |
| ------ | ---- | ---------- | ----- |
| GET | `/dashboard?range=7\|30\|90` | `dashboard:read` | revenue series, KPIs with daily series, heatmap, categories, top products, recent orders, activity feed, markets, quarterly target |
| PUT | `/target` | `workspace:manage` | `{goalCents}`: this quarter's revenue goal; returns the target with its recomputed pace |
| GET | `/live` | `dashboard:read` | simulated storefront traffic (2-minute history) and the hottest product, picked from active, in-stock best sellers on a demand signal that drifts minute to minute |
| GET | `/runtime` | `dashboard:read` | real Go runtime stats: goroutines, heap, GC, uptime |

### Products

| Method | Path | Permission | Notes |
| ------ | ---- | ---------- | ----- |
| GET | `/products?q&category&status&sort` | `products:read` | `{items, stats}`; sort by `revenue\|sales\|price\|stock\|name` |
| POST | `/products` | `products:write` | create |
| GET | `/products/{id}` | `products:read` | |
| PUT | `/products/{id}` | `products:write` | full update |
| DELETE | `/products/{id}` | `products:write` | also deletes uploaded files |
| POST | `/products/bulk` | `products:write` | `{ids, action: publish\|archive\|delete}` |
| GET | `/products/export` | `products:read` | CSV with the list's filters |
| POST | `/products/import` | `products:write` | CSV (multipart `file` or `text/csv` body), upsert by SKU; all-or-nothing, 422 lists the row errors |
| POST | `/products/{id}/images` | `products:write` | multipart `file`, at most 5 MB; the type is sniffed and SVG is refused |
| DELETE | `/products/{id}/images/{imageId}` | `products:write` | also removes the file |
| POST | `/products/{id}/images/{imageId}/primary` | `products:write` | make the cover image |

### Orders

| Method | Path | Permission | Notes |
| ------ | ---- | ---------- | ----- |
| GET | `/orders?q&status&customer&from&to&limit&offset` | `orders:read` | `{items, total, counts}`; `from` and `to` (RFC 3339) bound `placed_at` |
| GET | `/orders/{id}` | `orders:read` | |
| PATCH | `/orders/{id}/status` | `orders:write` | `{status, reason}`; see below |
| GET | `/orders/export` | `orders:read` | CSV, including the refund reason, who refunded and when |

Refunding needs a reason, which stays with the order:

```bash
curl -b jar -X PATCH -H 'Content-Type: application/json' \
  -d '{"status":"refunded","reason":"Damaged in transit"}' \
  http://localhost:8080/api/v1/orders/10231/status
```

```json
{ "id": 10231, "status": "refunded", "refund": { "reason": "Damaged in transit", "by": "Priya Shah", "at": "2026-10-05T16:20:11Z" }, "...": "..." }
```

The call returns 422 without a reason and 409 if the order is already refunded
or failed.

### Customers

| Method | Path | Permission | Notes |
| ------ | ---- | ---------- | ----- |
| GET | `/customers?q&segment` | `customers:read` | `{items, segments}`; segment is `VIP\|Regular\|New\|At risk` |
| GET | `/customers/{id}` | `customers:read` | profile, stats, full order history (with refund reasons), products bought, categories, monthly spend |
| POST | `/customers/{id}/notes` | `customers:write` | `{text}` |
| DELETE | `/customers/{id}/notes/{noteId}` | `customers:write` | written to the activity log |
| GET | `/customers/export` | `customers:read` | CSV |

### Team

| Method | Path | Permission | Notes |
| ------ | ---- | ---------- | ----- |
| GET | `/team?role` | `team:read` | members, including `granted` / `revoked` exceptions |
| POST | `/team` | `team:write` | `{name, email, role}` invites a member; only the owner can grant `admin` |
| GET | `/team/{id}` | `team:read` | `{member, permissions, online}`; online means a request in the last 5 minutes |
| PUT | `/team/{id}` | `team:write` | `{name, email, role}`; the owner can't be edited |
| PUT | `/team/{id}/access` | `team:write`, owner only | `{granted, revoked}` exceptions to the role; `workspace:manage` can't be granted |
| PUT | `/team/{id}/status` | `team:write` | `{status: active\|suspended}`; suspending signs the member out everywhere |
| DELETE | `/team/{id}` | `team:write` | nobody can remove themselves or the owner |

### Activity (audit log)

| Method | Path | Permission | Notes |
| ------ | ---- | ---------- | ----- |
| GET | `/activity?actor&kind&q&limit&offset` | `team:read` | `{items, total, kinds}`, newest first |

Each entry records who acted (`actorId`, `actor`), what they did (`message`) and
the record touched (`entity`, `entityId`). The UI opens that record when the
entry is clicked. The kinds are `product`, `publish`, `image`, `import`,
`order`, `refund`, `note`, `team`, `role`, `target`, `settings`, `auth` (sign-ins,
which the dashboard feed leaves out), `deploy` and `stock`.

### System

| Method | Path | Permission | Notes |
| ------ | ---- | ---------- | ----- |
| GET | `/requests?q&class=2\|4\|5&method&actor&limit` | `requests:read` | recent API calls and analytics (p50/p95/p99, per minute, top endpoints) |
| GET | `/requests/{id}` | `requests:read` | headers and bodies (redacted), user, route, timing |
| GET | `/settings/log-level` | signed in | `{level}` |
| PUT | `/settings/log-level` | `settings:write` | `{level: debug\|info\|warn\|error}`; applies until restart |

### Outside `/api/v1`

| Method | Path | Notes |
| ------ | ---- | ----- |
| GET | `/healthz` | `{status, database}`; the database is pinged when Postgres is used |
| GET | `/media/uploads/{file}` | public uploaded images, served with a strict CSP |
| GET | `/media/generated/{product}/{n}.svg` | fallback artwork for products without photos |
| GET | `/*` | the embedded UI (gzip, SPA fallback to `index.html`) |
