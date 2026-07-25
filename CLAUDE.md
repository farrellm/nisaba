# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

```sh
# Database
make db              # start Postgres container
make migrate         # run pending migrations
make migrate-down    # roll back last migration
make db-clean        # wipe container and data volume
# Inspect the running dev DB:
#   docker exec nisaba-postgres psql -U nisaba -d nisaba -c "SELECT ..."

# Backend
make backend         # go run ./cmd/server (port 8080)
make backend-watch   # like backend, auto-restarts on changes (needs wgo)
make backend-build   # compile to backend/bin/server
make backend-test    # go test ./...

# Frontend
make frontend-install  # npm install (first time only)
make frontend          # vite dev server (port 5173)
make frontend-build    # production build to frontend/dist/ (runs tsc first — also the typecheck)
```

Quick checks: `gofmt -l backend/` (format), `cd frontend && npx tsc --noEmit` (typecheck only), `cd frontend && npm run lint` (ESLint), `cd frontend && npm run format` (Prettier; `format:check` to verify).

Install golang-migrate before running migrations:
```sh
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

## Core Concept: Modes & Blocks

Nisaba is a tool for **writing with LLMs**. A **document** holds an ordered list of **blocks** plus its own key/value **attributes** (a shared namespace) and a **selected model**.

There is a **fixed, code-defined set of modes** (`backend/internal/mode`). Each mode declares a fixed set of input **keys**, an **output** key, a **mustache template**, and an optional set of **tools** (`Tools []llm.Tool`, `json:"-"`). The set is fixed at build time — there is no runtime CRUD.

The lifecycle:
1. **Add block** — the user picks a mode. The new block's attributes are seeded from the document's attributes for that mode's keys (empty string where the document has no value).
2. **Edit** — the user edits the block's key/values.
3. **Run** — the mode's mustache template is rendered against the block's key/values to produce a **prompt**; the prompt is sent to the document's selected model; the **response** is appended to the block's responses. The response is then processed into the document's attributes two ways (both apply): every **top-level XML tag** becomes an attribute (tag name → key, inner text → value; nested tags stay verbatim in the value), and when the mode declares a non-empty `output` key the full response is also saved under it. The `output` key is optional.

**The LLM call is provider-agnostic.** `blockrun.Service` (`internal/blockrun`) orchestrates a run — `Prepare` persists the caller's edits and renders the prompt + system prompt (`mode.Templates.SystemPrompt`, a per-user-overridable mustache template), `Execute`/`ExecuteStream` make the model call (detached from the client connection, bounded at 15m) and save/reparse the response; the HTTP handlers in `internal/handler/block.go` map its step-typed errors onto responses. It calls `llm.Generate(ctx, model, system, prompt, tools)` (`internal/llm`), a thin wrapper over the [GoAI SDK](https://github.com/zendev-sh/goai) that routes each model through the provider named in its `Model.Provider` — its own vendor (Anthropic/OpenAI/Google) or the OpenRouter aggregator — GoAI reads that provider's key from the environment (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`/`GOOGLE_GENERATIVE_AI_API_KEY`, `OPENROUTER_API_KEY`, `DEEPSEEK_API_KEY`). The service passes the mode's `Tools`; when non-empty, `Generate` attaches each tool and runs GoAI's agentic loop (`WithMaxSteps`, bounded by `maxToolIterations`) so the model can invoke tools and have results fed back, otherwise it does a single generation. Each document picks a model from a single **fixed, cross-provider list** (`llm.Models()`, served at `GET /api/models`); `Run` 400s if no model is selected. Keep all vendor-specific code behind `internal/llm` — never import a single-vendor SDK directly. `llm.Tool` is an alias for GoAI's tool type; build one with `goai.NewTool` (schema auto-generated from a typed input struct's `json`/`jsonschema` tags — no hand-written param schema). `internal/llm/names.go` ships a ready-made `llm.GenerateNameTool` (`generate_name`, returns a non-blocklisted character name) as a worked example — currently attached to the `brainstorm-tools-*` modes. `internal/llm/labels.go` is a second worked example: `llm.SuggestLabels(ctx, story)` hard-codes a model, renders the embedded `templates/suggest-labels.mustache`, and parses `<label>` tags out of the reply (templates that drive a hard-coded-model helper live under `internal/llm/templates/`, since `go:embed` can't reach a parent dir). The same file's `llm.SelectLabels(ctx, story, available)` renders `templates/available-labels.mustache` and filters the reply down to the supplied label set (picking relevant ones from an existing pool rather than inventing new labels).

## Architecture

Per-directory guidance lives in `frontend/CLAUDE.md`, `backend/CLAUDE.md`, and `db/CLAUDE.md` — each loads when you work under that directory.

Cross-cutting: `GET /api/public/documents/{id}/attributes/{key}` is the **only** route outside the backend's `RequireUser` group, and it is deliberately public (the shareable `AttributePage` reads it). Access is open by guessable sequential id by design; it returns an empty value (never 404) so it can't be used to probe which documents exist. Don't add an auth check there, and don't extend it to return anything beyond one attribute value plus the title.
