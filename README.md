# Nisaba

Nisaba is a tool for **writing with LLMs**. A document is built from **blocks**, and every block is created in one of a fixed set of **modes**. Each mode has a fixed set of keys and a mustache prompt template. When you add a block its values are seeded from the document's shared key/values; when you **run** it, the template renders those values into a prompt, the prompt goes to the document's selected model, and the response is saved to the block and fed back into the document's key/values. The model call is provider-agnostic (via `dragon-born/go-llm`), and the model is chosen per document from a fixed, cross-provider list.

Built on a React + MUI frontend, Go backend, and PostgreSQL database.

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

### LLM provider

The LLM call is provider-agnostic via [`dragon-born/go-llm`](https://gopkg.in/dragon-born/go-llm.v1), wrapped in `backend/internal/llm`. Requests route through go-llm's default gateway (OpenRouter), so a single key reaches every model in the fixed list (Anthropic, OpenAI, Google, …). Set it before running a block:

```sh
export OPENROUTER_API_KEY=...
```

go-llm also supports per-provider keys directly (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GOOGLE_API_KEY`, …) if you switch the default provider.

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
| POST | `/api/auth/{register,login,logout}` | Session auth |
| GET | `/api/auth/me` | Current user |
| GET | `/api/modes` | The fixed set of writing modes (name, keys, output) |
| GET | `/api/models` | The fixed, cross-provider list of selectable models |
| GET | `/api/documents` | List the user's documents (`?archived=true` to include archived) |
| POST | `/api/documents` | Create a document |
| GET | `/api/documents/{id}` | Get a document with its blocks, attributes, and responses |
| PUT | `/api/documents/{id}` | Update the document's selected model |
| POST | `/api/documents/{id}/blocks` | Add a block (choose a mode); seeds attributes from the document |
| PUT | `/api/documents/{id}/blocks/{blockId}` | Update a block's key/values |
| POST | `/api/documents/{id}/blocks/{blockId}/run` | Render the prompt, send it to the selected model, and save the response |
