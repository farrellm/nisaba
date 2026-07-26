# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

`make help` lists every target (db/migrate/backend/frontend). Quick checks: `gofmt -l backend/`, and in `frontend/`: `npx tsc --noEmit`, `npm run lint`, `npm run format`.

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

**The LLM call is provider-agnostic** — kept entirely behind `internal/llm`; see `backend/CLAUDE.md` for the details.

## Architecture

Per-directory guidance lives in `frontend/CLAUDE.md`, `backend/CLAUDE.md`, and `db/CLAUDE.md` — each loads when you work under that directory.

Cross-cutting: `GET /api/public/documents/{id}/attributes/{key}` is the **only** route outside the backend's `RequireUser` group, and it is deliberately public (the shareable `AttributePage` reads it). Access is open by guessable sequential id by design; it returns an empty value (never 404) so it can't be used to probe which documents exist. Don't add an auth check there, and don't extend it to return anything beyond one attribute value plus the title.
