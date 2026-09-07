# Nisaba

Nisaba is a tool for **writing with LLMs**. A document is built from **blocks**, and every block is created in one of a fixed set of **modes**. Each mode has a fixed set of keys and a mustache prompt template. When you add a block its values are seeded from the document's shared key/values; when you **run** it, the template renders those values into a prompt, the prompt goes to the document's selected model, and the response is saved to the block and fed back into the document's key/values. The model call is provider-agnostic (via the [GoAI SDK](https://github.com/zendev-sh/goai), routing each model directly to its own provider), and the model is chosen per document from a fixed, cross-provider list.

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

Open http://localhost:5173. You should see two green status chips confirming the API and database are reachable. To log in you need an account — see below.

## Managing Users

There is no sign-up page. Accounts exist only where the server binary puts them:

```sh
cd backend
go run ./cmd/server -create-user alice   # prompts for the password twice, echo off
go run ./cmd/server -list-users
go run ./cmd/server -delete-user alice   # confirms first; -force skips the prompt
```

Each flag performs its action against `DATABASE_URL` and exits; with no flags the binary serves HTTP as usual. Passwords must be at least 8 characters, and are read from stdin when it isn't a terminal (`echo 's3cretpw' | ./bin/server -create-user alice`), so the password never lands in shell history. Deleting a user cascades to all of their documents and labels.

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
make deploy         Build both and restart the deployed systemd unit
```

## Configuration

The backend reads configuration from environment variables with development defaults:

| Variable | Default | Description |
|---|---|---|
| `ADDR` | `:8080` | HTTP listen address |
| `DATABASE_URL` | `postgres://nisaba:nisaba@localhost:5432/nisaba?sslmode=disable` | Postgres connection string |
| `CORS_ORIGINS` | `http://localhost:5173` | Comma-separated allowed origins |
| `WEB_DIR` | *(empty)* | Directory of the built frontend to serve at `/` with an SPA fallback. Empty disables static serving — what local dev wants, since Vite serves the app and proxies `/api` here |
| `SESSION_SECRET` | `dev-insecure-session-secret-change-me` | Signs the session cookie. **Production must override this** with a long random value (`openssl rand -base64 48`) |
| `SESSION_SECURE` | `false` | `true` marks the session cookie `Secure` (HTTPS only) |
| `MODE_TEMPLATES_DIR` | `internal/mode/templates` | Base mode-template dir; per-user overrides are read from the sibling `<dir>-<username>/`. Relative to the working directory |
| `REFLEX_DB_PATH` | `../reflex.db` | Legacy SQLite file browsed read-only by the Anansi pages. Relative to the working directory, and **a missing file fails startup** |
| `CHARLOTTE_CLI` | `charlotte-cli` | Executable behind the Charlotte pages; resolved on `PATH`. A missing binary does *not* fail startup |

### LLM provider

The LLM call is provider-agnostic via the [GoAI SDK](https://github.com/zendev-sh/goai), wrapped in `backend/internal/llm`. Each model in the fixed list routes directly to its own provider (Anthropic, OpenAI, Google), so set that provider's key before running a block:

```sh
export ANTHROPIC_API_KEY=...   # Claude models
export OPENAI_API_KEY=...      # GPT models
export GEMINI_API_KEY=...      # Gemini models (or GOOGLE_GENERATIVE_AI_API_KEY)
```

GoAI reads each key from the environment automatically; you only need the keys for the providers whose models you actually run.

## Deployment

The deployed instance is a **systemd user unit**, `~/.config/systemd/user/nisaba.service`,
bound to `127.0.0.1:8092` — loopback only, because the sole way in is the Tailscale proxy
below. `tailscale funnel` publishes it on **port 443**, so it is reachable both on the
tailnet and from the public internet at `https://<node>.<tailnet>.ts.net/` (`tailscale
status` prints the node's name), with a real cert terminated by `tailscaled`.

```sh
make deploy                     # build backend + frontend, restart the unit
journalctl --user -u nisaba -f  # logs
tailscale funnel status         # confirm the mapping is public
```

**One process serves everything.** The unit sets `WEB_DIR=frontend/dist`, so the Go server
serves the built SPA at `/` with a `try_files`-style fallback to `index.html` alongside its
own `/api` routes. That matters because `tailscale serve` proxies a *single* upstream: the
frontend only ever fetches relative `/api/...` paths and the session cookie is host-only
`SameSite=Lax`, so splitting the two across origins does not work — and Tailscale's own
static-file mode has no SPA fallback, which would break deep links like `/documents/5`.

**Streaming needs no proxy tuning here.** Running a block streams the reply over a single
long-lived NDJSON connection, kept warm by a keepalive `ping` every 10s, and a
thinking-heavy model can run for minutes — a buffering proxy would drop it mid-run with
`context canceled`. `tailscale serve` builds a Go `httputil.ReverseProxy`, which flushes
immediately whenever the response has no `Content-Length`, as this one does. Nothing to
configure. (An nginx or Traefik front end *would* need `proxy_buffering off` and a raised
`proxy_read_timeout`; this host used to run one and no longer does.)

### Deployment notes

- **Funnel must be enabled for the tailnet** before `tailscale funnel` will do anything.
  The CLI prints a `login.tailscale.com/f/funnel?node=…` enrollment link on first use;
  alternatively add `"nodeAttrs": [{"target": ["autogroup:member"], "attr": ["funnel"]}]`
  to the tailnet policy file. Funnel accepts only ports **443, 8443, and 10000**.
- The unit's `ExecStartPost` owns the funnel mapping, so the unit stays the single source of
  truth for how nisaba is exposed even though `tailscaled` persists it separately. It is
  bounded by `timeout` and failure-tolerant: if Funnel is not enabled the CLI blocks on that
  enrollment link, which would otherwise tear down a perfectly healthy server.
- The backend does no dotenv loading, and systemd's `EnvironmentFile=` cannot parse
  `.envrc`'s `export VAR=…` lines, so `ExecStart` sources `.envrc` directly. It stays the
  single source of truth for the provider keys and `SESSION_SECRET` — rotating one is an
  edit plus `systemctl --user restart nisaba`.
- `ExecStartPre` waits on the Postgres container's healthcheck and runs `migrate … up`, so a
  deploy carrying a schema change cannot start against the old schema.
- Port **8092**, not the `:8080` default, so `make backend-watch` keeps 8080 for local dev
  and the two can run side by side.
- **nisaba is now on the public internet.** `GET /api/public/documents/{id}/attributes/{key}`
  is deliberately unauthenticated and readable by guessable sequential id (it returns an
  empty value rather than 404, so it cannot be used to probe which documents exist) — that
  is now world-reachable. Login is rate-limited per IP; note that Funnel requests can share
  an ingress source address, so the limit throttles brute force in aggregate.

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
| POST | `/api/auth/{login,logout}` | Session auth (accounts are created from the CLI, not over HTTP) |
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
