# Contributing

[← README](README.md) · [Architecture](docs/ARCHITECTURE.md) · [API reference](docs/API.md)

Thanks for helping. This is a reference project, so code is judged on how well
it reads as an example as much as on what it does. Keep changes small, typed
and tested.

## Setup

You need Go 1.26+, Node 24, Docker (for Postgres and for `sqlc`) and Chrome
(for the e2e suite locally).

```bash
docker compose up -d --wait postgres                 # Postgres on :5433, plus goadmin_test
make dev-api                                         # API on :8080, migrates and seeds
cd web && npm install && npm run dev                 # UI on :5173 with hot reload
```

`make dev-api-mem` runs the API on the in-memory store, with no Docker
needed. Sign in with any demo account; the password is `goadmin`.

## Before you open a pull request

```bash
gofmt -l . && go vet ./...
go test ./...                    # API tests on the in-memory store
make test-pg                     # the same tests on Postgres + memory/Postgres parity
cd web && npm run typecheck
cd web && npm run e2e            # Playwright, desktop and mobile, in-memory server
cd web && npm run e2e:pg         # Playwright on a fresh Postgres database
```

All of these must pass. If a change is visible in the UI, regenerate the README
screenshots with `cd web && npm run build && npm run screenshots` and commit
`docs/screenshots/`.

## How the code is organised

Read [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) first. In short:

- `internal/domain` holds types, validation, errors and RBAC. It imports
  nothing else from the project.
- `internal/httpapi` holds routes (with their permission), handlers and
  middleware. Handlers talk to the `Store` interface, never to a concrete store.
- `internal/store/postgres` and `internal/store/memory` implement that
  interface. **Both** must support every feature.
- `web/src/features/<page>` has one folder per page. Shared UI goes in
  `components/`, and the API client and query hooks in `lib/`.

## Recipes

### Add an API endpoint

1. Put types and validation in `internal/domain` (money in cents,
   `NewValidationError` for field errors).
2. Add the store method to `httpapi.Store`, and implement it in `memory` and
   `postgres`. For SQL, edit `queries/*.sql` and run `make sqlc`; never edit
   `db/` by hand.
3. Register the route in `server.go` with the narrowest permission that fits.
4. In the handler, use `decodeJSON`, validate, call the store and map errors
   with `writeDomainError`. If it changes data, call `s.audit(...)` afterwards
   so it shows in the activity log.
5. Test it in `internal/httpapi/*_test.go`. Tests run on both stores, so
   include the 4xx cases and the role checks.
6. Add it to the client in `web/src/lib/api.ts` and to [docs/API.md](docs/API.md).

### Change the schema

Add `internal/store/postgres/migrations/0000N_name.sql` with goose
`-- +goose Up` / `-- +goose Down` sections. Migrations are embedded and run on
boot. Never edit a migration that has shipped; add a new one. If the demo data
needs the new column, update `internal/seed` and `postgres/seed.go`. Existing
demo databases may also need a one-off backfill in `postgres/seed.go`.

### Add a page or a sheet

1. Create `web/src/features/<page>/<page>-page.tsx` and register a lazy route in
   `router.tsx` with `guard('<permission>')`.
2. Add it to `lib/nav.ts` with the same permission.
3. Use the hooks in `lib/queries.ts` (add keys there). Sheets for records that
   other pages link to should open with `usePeek()` / `PeekButton`, not by
   navigating.
4. Check it at 375 px. Tables either keep the key columns (`max-sm:hidden` on
   the rest) or switch to compact rows. The mobile e2e suite fails if a page
   scrolls sideways.
5. Add a Playwright spec in `web/e2e`.

## Style

- **Go:** `gofmt`, small handlers, errors wrapped with context, `log/slog` with
  key/value pairs. Comments say why, not what.
- **TypeScript/React:** strict types, function components, Tailwind utilities
  with design tokens (`bg-panel`, `text-muted`, `text-accent`). Don't hard-code
  colours, because the accent is a runtime variable.
- **Copy:** UI text and docs are in English, short and specific.
- **Security:** every new route goes through `authorize`. Never log secrets;
  the request log redacts by field name, so name password and token fields
  accordingly.

## Commits and pull requests

- One logical change per commit, with an imperative subject line ("Add refund
  reasons to the orders list") and a body that explains why.
- Keep pull requests focused. Describe what changed and how you tested it, and
  attach a screenshot for UI changes.
- CI (GitHub Actions) runs the same checks as the list above, on both stores, and
  must be green before merging. See [CI/CD in the README](README.md#cicd).

## Reporting issues

Open an issue with steps to reproduce, what you expected and what happened.
Include the `X-Request-ID` response header of a failing API call if you have one.
