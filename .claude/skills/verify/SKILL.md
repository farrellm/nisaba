---
name: verify
description: Build, launch, and drive Nisaba locally to verify a change end-to-end (backend API + browser UI).
---

# Verifying Nisaba changes

## Launch

- Postgres usually already runs (`docker ps --filter name=nisaba-postgres`); else `make db && make migrate`.
- LLM API keys (`ANTHROPIC_API_KEY` etc.) are in the environment — real model calls work.
- Backend: `cd backend && go build -o <scratch>/server ./cmd/server && <scratch>/server &` — **run it with cwd `backend/`** or it dies at startup on the `../reflex.db` default (`REFLEX_DB_PATH`). 8080 is usually taken by the user's own dev server: set `ADDR=127.0.0.1:<free port>` (check with `ss -ltn` first).
- Reddit endpoints need `REDDIT_SESSION`; `source .envrc` before launching. `REDDIT_SESSION=bogus` exercises the error paths safely, and never submit with the real session without the user's OK.
- Frontend: `npm run dev` proxies `/api` to a hard-coded 8080. On another port, use `npx vite build --outDir <scratch>/web` and launch the server with `WEB_DIR=<scratch>/web` so it serves the UI itself.

## Drive the API (no browser needed)

Cookie-jar curl against the backend's port works for the whole flow:

1. `POST /api/auth/login` with `-c jar`. There is no register endpoint — accounts come from the binary itself: `<scratch>/server -list-users` shows the existing ones, `echo 'testpassword' | <scratch>/server -create-user verifyuser` makes a new one (`-delete-user <name> -force` cleans it up). Don't assume `streamingEnabled` — read it off `GET /api/auth/me`.
2. `POST /api/documents` → id; `PUT /api/documents/{id}` `{"selectedModel":"claude-haiku-5-5"}`. Always verify with `claude-haiku-5-5` (cheapest/fastest) unless the change under test is model-specific. **Never test with Fable.** If Haiku is not in `GET /api/models` (models get dropped from the list via `Model.Hidden`), fall back to `deepseek-v4-pro`.
3. `POST /api/documents/{id}/blocks` `{"mode":"brainstorm-tools-1"}` — the tools modes attach `generate_name`.
4. `curl -N -b jar -X POST .../blocks/{bid}/run/stream` with `{"attributes":{"prompt":"..."}}` streams NDJSON; add "Keep all prose extremely brief" to the prompt to shorten runs (~40s vs ~60s+).
5. The `done` event's `block.responses[-1].value` is the persisted output — compare against concatenated deltas.

Gotchas: runs are detached server-side (`context.WithoutCancel`), so a client that dies mid-request still costs a model call and saves a response. Delete the test document afterwards (`DELETE /api/documents/{id}`, cascades).

## Drive the UI

Playwright MCP against `localhost:5173`: log out first if a stale session user is wrong (streaming toggle lives in the account menu). Document page: fill `prompt`, click Run, the live preview is the "streaming…" block under the buttons.
