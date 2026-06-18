# Nisaba

Web application skeleton: React + MUI frontend, Go backend, PostgreSQL database.

## Stack

- **Frontend**: React 18, TypeScript, Vite, MUI v6
- **Backend**: Go, chi, pgx
- **Database**: PostgreSQL 17 (Docker)
- **Migrations**: golang-migrate

## Prerequisites

- [Go](https://go.dev/dl/) 1.24+
- [Node.js](https://nodejs.org/) 20+
- [Docker](https://docs.docker.com/get-docker/) (for Postgres)
- [golang-migrate](https://github.com/golang-migrate/migrate):
  ```sh
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  ```

## Getting Started

```sh
# 1. Start Postgres and run migrations
make db
make migrate

# 2. Start the Go API server (port 8080)
make backend

# 3. In another terminal, install deps and start Vite (port 5173)
make frontend-install
make frontend
```

Open http://localhost:5173. You should see two green status chips confirming the API and database are reachable.

## Make Targets

```
make help           Show all targets
make db             Start Postgres container (data persists across restarts)
make db-stop        Stop Postgres container
make db-clean       Remove container and wipe data volume
make migrate        Run pending migrations
make migrate-down   Roll back the last migration
make backend        Run Go server (go run)
make backend-build  Compile binary to backend/bin/server
make backend-test   Run Go tests
make frontend-install  Install npm dependencies
make frontend       Start Vite dev server
make frontend-build Build frontend for production
```

## Configuration

The backend reads configuration from environment variables with development defaults:

| Variable | Default | Description |
|---|---|---|
| `ADDR` | `:8080` | HTTP listen address |
| `DATABASE_URL` | `postgres://nisaba:nisaba@localhost:5432/nisaba?sslmode=disable` | Postgres connection string |
| `CORS_ORIGINS` | `http://localhost:5173` | Comma-separated allowed origins |

## Project Structure

```
├── backend/
│   ├── cmd/server/main.go       # Entry point
│   └── internal/
│       ├── config/config.go     # Environment config
│       ├── db/db.go             # Connection pool
│       └── handler/health.go    # GET /api/healthz
├── db/
│   └── migrations/              # golang-migrate SQL files
├── frontend/
│   └── src/
│       ├── App.tsx              # Root component
│       ├── main.tsx             # React entry point
│       └── theme.ts             # MUI theme
├── docker-compose.yml
└── Makefile
```

## API

| Method | Path | Description |
|---|---|---|
| GET | `/api/healthz` | Returns API and database status |
