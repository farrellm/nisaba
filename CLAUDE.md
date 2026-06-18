# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

```sh
# Database
make db              # start Postgres container
make migrate         # run pending migrations
make migrate-down    # roll back last migration
make db-clean        # wipe container and data volume

# Backend
make backend         # go run ./cmd/server (port 8080)
make backend-build   # compile to backend/bin/server
make backend-test    # go test ./...

# Frontend
make frontend-install  # npm install (first time only)
make frontend          # vite dev server (port 5173)
make frontend-build    # production build to frontend/dist/
```

Install golang-migrate before running migrations:
```sh
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

## Architecture

The app is split into three independent directories: `frontend/`, `backend/`, and `db/`.

**Frontend** (`frontend/`) — Vite + React 18 + TypeScript + MUI v6. The Vite dev server proxies all `/api/*` requests to `http://localhost:8080`, so the browser never makes a cross-origin request during development. Production builds output to `frontend/dist/`.

**Backend** (`backend/`) — Go module `github.com/farrellm/nisaba`. Entry point is `cmd/server/main.go`. Internal packages:
- `internal/config` — reads `ADDR`, `DATABASE_URL`, `CORS_ORIGINS` from env with local dev defaults
- `internal/db` — opens a `pgxpool.Pool` and pings on startup to fail fast
- `internal/handler` — `http.HandlerFunc` closures that accept the pool; add new handlers here
- `internal/model` — plain domain structs mirroring the DB schema (no data-access logic); JSON-tagged, aggregate-shaped for API bodies
- `internal/store` — `Store` wraps the pool with raw-SQL CRUD methods over the models; returns `store.ErrNotFound` for missing rows. `GetDocument` loads the full aggregate (blocks → attributes/responses) with batched queries

Routing uses `go-chi/chi`. Handlers are plain `http.HandlerFunc` (no framework-specific types). CORS is handled by `rs/cors` middleware — it's unused during local dev (covered by the Vite proxy) but activates in production.

**Database** (`db/migrations/`) — plain SQL files managed by `golang-migrate`. Naming convention: `000001_<name>.up.sql` / `000001_<name>.down.sql`. Local credentials: `nisaba/nisaba/nisaba` (user/password/db).
Domain tables use `BIGSERIAL` ids and `ON DELETE CASCADE` FKs. String key/value attributes live in normalized child tables with `PRIMARY KEY (parent_id, key)` for uniqueness; free-form `metadata` is a `JSONB` column.

## Adding a New API Endpoint

1. Add a handler function in `backend/internal/handler/` returning `http.HandlerFunc`
2. Register the route in `backend/cmd/server/main.go` inside the `/api` route group
3. Call it from `frontend/src/` using a relative `/api/...` path
