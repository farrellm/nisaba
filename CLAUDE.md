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
make frontend-build    # production build to frontend/dist/ (runs tsc first — also the typecheck)
```

Quick checks: `gofmt -l backend/` (format), `cd frontend && npx tsc --noEmit` (typecheck only). There is no separate frontend lint step.

Install golang-migrate before running migrations:
```sh
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

## Core Concept: Modes & Blocks

Nisaba is a tool for **writing with LLMs**. A **document** holds an ordered list of **blocks** plus its own key/value **attributes** (a shared namespace) and a **selected model**.

There is a **fixed, code-defined set of modes** (`backend/internal/mode`). Each mode declares a fixed set of input **keys**, an **output** key, a **mustache template**, and an optional set of **tools** (`Tools []llm.ToolDef`, `json:"-"`). The set is fixed at build time — there is no runtime CRUD.

The lifecycle:
1. **Add block** — the user picks a mode. The new block's attributes are seeded from the document's attributes for that mode's keys (empty string where the document has no value).
2. **Edit** — the user edits the block's key/values.
3. **Run** — the mode's mustache template is rendered against the block's key/values to produce a **prompt**; the prompt is sent to the document's selected model; the **response** is appended to the block's responses. The response is then processed into the document's attributes two ways (both apply): every **top-level XML tag** becomes an attribute (tag name → key, inner text → value; nested tags stay verbatim in the value), and when the mode declares a non-empty `output` key the full response is also saved under it. The `output` key is optional.

**The LLM call is provider-agnostic.** `RunBlock` (`internal/handler/block.go`) renders the prompt and calls `llm.Generate(ctx, model, prompt, tools)` (`internal/llm`), a thin wrapper over [`dragon-born/go-llm`](https://gopkg.in/dragon-born/go-llm.v1) that routes through go-llm's default OpenRouter gateway — one key (`OPENROUTER_API_KEY`) reaches every model. `RunBlock` passes the mode's `Tools`; when non-empty, `Generate` attaches each tool and runs go-llm's agentic loop (`RunTools`, bounded by `maxToolIterations`) so the model can invoke tools and have results fed back, otherwise it uses the plain ask path. Each document picks a model from a single **fixed, cross-provider list** (`llm.Models()`, served at `GET /api/models`); `Run` 400s if no model is selected. Keep all vendor-specific code behind `internal/llm` — never import a single-vendor SDK directly. `llm.ToolDef` (a tool's name/description/JSON-schema params/handler) and `llm.Params()` (param-schema builder) are re-exported from go-llm so modes configure tools without importing the vendor library.

## Architecture

The app is split into three independent directories: `frontend/`, `backend/`, and `db/`.

**Frontend** (`frontend/`) — Vite + React 18 + TypeScript + MUI v6. The Vite dev server proxies all `/api/*` requests to `http://localhost:8080`, so the browser never makes a cross-origin request during development. Production builds output to `frontend/dist/`.
Routing uses `react-router-dom` (routes in `src/App.tsx`, providers in `src/main.tsx`). Call the API via `src/api/client.ts` (`api.get/post/put`, sends the session cookie) rather than raw `fetch`. Frontend API types are in `src/api/types.ts` (`Document`, `DocumentDetail`, `Block`, `Response`, `Mode`, `LLMModel`); treat any array from the API as possibly `null` and guard with `?? []`. Current user comes from `useAuth()` (`src/auth/AuthContext.tsx`); wrap protected routes in `RequireAuth`, which redirects to `/login`. The document view (`pages/DocumentPage.tsx`) renders blocks via `components/BlockCard.tsx`, adds them via `components/AddBlockDialog.tsx`, and picks the model via the fixed lower-left `components/ModelSelector.tsx` (auto-saves through `PUT /api/documents/{id}`).

**Backend** (`backend/`) — Go module `github.com/farrellm/nisaba`. Entry point is `cmd/server/main.go`. Internal packages:
- `internal/config` — reads `ADDR`, `DATABASE_URL`, `CORS_ORIGINS` from env with local dev defaults
- `internal/auth` — cookie-session helper (gorilla/sessions, signed, HttpOnly). `SESSION_SECRET` signs the cookie (dev default; prod must override); `SESSION_SECURE=true` sets the Secure flag for HTTPS
- `internal/db` — opens a `pgxpool.Pool` and pings on startup to fail fast
- `internal/handler` — `http.HandlerFunc` closures; data-access handlers take `*store.Store` (built via `store.New(pool)` in main.go), auth-aware ones also take `*auth.Sessions`. Only `Health` still takes the raw pool. Auth flow lives in `auth.go`: `/api/auth/{register,login,logout,me}`, bcrypt-hashed passwords, generic 401 on bad login, 409 on duplicate username. Document CRUD lives in `document.go`: `/api/documents` (list/create) and `/api/documents/{id}` (get); list takes `?archived=true`. `PUT /api/documents/{id}` (`document.go`) updates the selected model and/or document attribute values — body fields are optional pointers, each applied only when present (attributes via `MergeDocumentAttributes`, so absent keys survive). Block flow lives in `block.go`: `POST /api/documents/{id}/blocks` (add, seeds attrs from the document), `PUT .../blocks/{blockId}` (edit key/values), `POST .../blocks/{blockId}/run` (assemble prompt + real model call). `mode.go` serves `GET /api/modes`; `model.go` serves `GET /api/models`. Shared helpers `ownedDocument`/`findBlock` enforce ownership. Conventions: resources owned by another user return **404, not 403** (don't leak existence); list endpoints guard `nil` slices so the JSON body is `[]`, never `null`
- `internal/mode` — the fixed mode registry: `Mode{Name, Label, Keys, Output, Template, Tools}`, with mustache templates embedded from `templates/*.mustache` via `go:embed`. `All()` / `Get(name)`. The `Template` and `Tools` fields are `json:"-"` so they stay server-side. Add a mode by adding an entry plus its template file; attach tools by setting `Tools: []llm.ToolDef{...}` (see the LLM section). **Per-user template overrides**: `TemplateFor(username, mode)` (used by `RunBlock`) reads a runtime override from `<TemplatesBaseDir>-<username>/<mode>.mustache` (sibling of the default dir) when present, else falls back per-file to the embedded default. `TemplatesBaseDir` is set from `cfg.ModeTemplatesDir` (env `MODE_TEMPLATES_DIR`, default `internal/mode/templates`). Usernames are validated (`[A-Za-z0-9_-]` only) to prevent path traversal; the override dirs (`templates-*/`) are git-ignored
- Response processing lives in `internal/handler/response.go`: `parseTopLevelTags` byte-scans free-form responses (not `encoding/xml`) for top-level tags, counting same-name nesting depth and degrading gracefully on malformed input. `RunBlock` merges the parsed tags (plus the `output` key when set) via `MergeDocumentAttributes`. Covered by `response_test.go` (the repo's first Go test; run with `make backend-test`)
- `internal/llm` — provider-agnostic LLM wrapper over `dragon-born/go-llm`. Holds the fixed cross-provider model list (`Models()` / `Valid(id)`) and `Generate(ctx, model, prompt, tools)`. Also re-exports `ToolDef` and `Params()` so modes attach tools without importing go-llm. The only place vendor/provider code lives; add a model by editing the list (IDs come from go-llm's `Model` constants)
- `internal/model` — plain domain structs mirroring the DB schema (no data-access logic); JSON-tagged, aggregate-shaped for API bodies
- `internal/store` — `Store` wraps the pool with raw-SQL CRUD methods over the models; returns `store.ErrNotFound` for missing rows. `GetDocument` loads the full aggregate (blocks → attributes/responses) with batched queries
- **Empty Go slices marshal to JSON `null`, not `[]`** — embedded aggregate slices (`Document.Blocks`/`Labels`, `Block.Responses`) must be defaulted to an empty slice in the store, or the frontend crashes on `.length`/`.map`. `GetDocument`/`GetBlock` guard `Blocks`/`Responses`; `Labels` is not yet guarded, so guard slice access with `?? []` on the frontend too

Routing uses `go-chi/chi`. Handlers are plain `http.HandlerFunc` (no framework-specific types). CORS is handled by `rs/cors` middleware — it's unused during local dev (covered by the Vite proxy) but activates in production.

**Database** (`db/migrations/`) — plain SQL files managed by `golang-migrate`. Naming convention: `000001_<name>.up.sql` / `000001_<name>.down.sql`. Local credentials: `nisaba/nisaba/nisaba` (user/password/db).
Domain tables use `BIGSERIAL` ids and `ON DELETE CASCADE` FKs. String key/value attributes live in normalized child tables with `PRIMARY KEY (parent_id, key)` for uniqueness; free-form `metadata` is a `JSONB` column.

## Adding a New API Endpoint

1. Add a handler function in `backend/internal/handler/` returning `http.HandlerFunc`
2. For user-scoped data, read the caller via `sess.UserID(r)` (401 if absent) and scope/own-check on it — see `document.go`
3. For resources nested under a document (`/documents/{id}/...`), reuse the `ownedDocument`/`findBlock` helpers in `block.go` and register inside the nested `r.Route("/{id}", ...)` group rather than re-checking ownership
4. Register the route in `backend/cmd/server/main.go` inside the `/api` route group
5. Call it from `frontend/src/` using a relative `/api/...` path
