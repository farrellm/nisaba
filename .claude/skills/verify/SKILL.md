---
name: verify
description: Build, launch, and drive Nisaba locally to verify a change end-to-end (backend API + browser UI).
---

# Verifying Nisaba changes

## Launch

- Postgres usually already runs (`docker ps --filter name=nisaba-postgres`); else `make db && make migrate`.
- LLM API keys (`ANTHROPIC_API_KEY` etc.) are in the environment — real model calls work.
- Backend: `cd backend && go build -o <scratch>/server ./cmd/server && <scratch>/server &` — **run it with cwd inside the repo** (repo root or `backend/`) or it dies at startup on the `../reflex.db` default (`REFLEX_DB_PATH`). Port 8080; check `curl localhost:8080/api/models`.
- Frontend: `cd frontend && npm run dev &` — port 5173, proxies `/api` to 8080.

## Drive the API (no browser needed)

Cookie-jar curl against 8080 works for the whole flow:

1. `POST /api/auth/register` (or `/login`) with `-c jar` — a fresh user has `streamingEnabled:false`.
2. `POST /api/documents` → id; `PUT /api/documents/{id}` `{"selectedModel":"claude-haiku-4-5"}`. Always verify with `claude-haiku-4-5` (cheapest/fastest) unless the change under test is model-specific.
3. `POST /api/documents/{id}/blocks` `{"mode":"brainstorm-tools-1"}` — the tools modes attach `generate_name`.
4. `curl -N -b jar -X POST .../blocks/{bid}/run/stream` with `{"attributes":{"prompt":"..."}}` streams NDJSON; add "Keep all prose extremely brief" to the prompt to shorten runs (~40s vs ~60s+).
5. The `done` event's `block.responses[-1].value` is the persisted output — compare against concatenated deltas.

Gotchas: runs are detached server-side (`context.WithoutCancel`), so a client that dies mid-request still costs a model call and saves a response. Delete the test document afterwards (`DELETE /api/documents/{id}`, cascades).

## Drive the UI

Playwright MCP against `localhost:5173`: log out first if a stale session user is wrong (streaming toggle lives in the account menu). Document page: fill `prompt`, click Run, the live preview is the "streaming…" block under the buttons.
