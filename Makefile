.PHONY: help db db-stop db-clean migrate migrate-down backend backend-watch backend-build backend-test backend-lint frontend-install frontend frontend-build deploy tools hooks fmt fmt-check lint test check

MIGRATE_BIN := migrate
MIGRATE_DIR := db/migrations
DB_URL      := postgres://nisaba:nisaba@localhost:5432/nisaba?sslmode=disable

# Bake the absolute path of charlotte-cli (as found on the build shell's PATH) into
# the binary as the CHARLOTTE_CLI default; falls back to the bare name if absent.
CHARLOTTE_CLI_PATH := $(or $(shell command -v charlotte-cli 2>/dev/null),charlotte-cli)
GO_LDFLAGS  := -X github.com/farrellm/nisaba/internal/config.charlotteCLIDefault=$(CHARLOTTE_CLI_PATH)

# Pinned dev tools, installed into ./bin by `make tools` and put first on PATH so
# local runs, git hooks, and CI all use the same versions.
BIN                   := $(CURDIR)/bin
export PATH           := $(BIN):$(PATH)
GOLANGCI_LINT_VERSION := v2.14.0
GITLEAKS_VERSION      := v8.30.1
LEFTHOOK_VERSION      := v2.1.15
SQUAWK_VERSION        := v2.66.0
SQUAWK_SHA256_linux-x64    := e7965f8146b53cfa7a4625ceecfea5c6aeb35add39132210aa60b11ae180b9f4
SQUAWK_SHA256_linux-arm64  := 04455267bb895de1568dc24e52eb062b311096ad614c3e15a77cf8a7b3483c2a
SQUAWK_SHA256_darwin-x64   := e361365762a6bf11abbef746d29ff46ac0b4feba43df2f0570595f06c67d7daf
SQUAWK_SHA256_darwin-arm64 := d273f3a234b81b8d540b2cdf1b07ed84f719a3ab11b47d6f3efe96baaf21b40b
SQUAWK_PLATFORM       := $(shell uname -s | tr A-Z a-z)-$(shell uname -m | sed -e 's/x86_64/x64/' -e 's/aarch64/arm64/')

# Root-level YAML Prettier owns (frontend/ has its own `npm run format`; root and
# backend Markdown is left alone — Prettier would re-pad every README table).
ROOT_PRETTIER := '*.yml' 'backend/*.yml' '.github/**/*.yml'
PRETTIER      := frontend/node_modules/.bin/prettier

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Checks ────────────────────────────────────────────────────────────────────

tools: $(BIN)/.golangci-lint-$(GOLANGCI_LINT_VERSION) $(BIN)/.gitleaks-$(GITLEAKS_VERSION) $(BIN)/.lefthook-$(LEFTHOOK_VERSION) $(BIN)/.squawk-$(SQUAWK_VERSION) ## Install pinned dev tools into ./bin

# Go tools are verified by the Go checksum database; squawk by a pinned sha256.
$(BIN)/.golangci-lint-%:
	GOBIN=$(BIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$*
	@touch $@
$(BIN)/.gitleaks-%:
	GOBIN=$(BIN) go install github.com/zricethezav/gitleaks/v8@$*
	@touch $@
$(BIN)/.lefthook-%:
	GOBIN=$(BIN) go install github.com/evilmartians/lefthook/v2@$*
	@touch $@
$(BIN)/.squawk-%:
	@mkdir -p $(BIN)
	curl -fsSL -o $(BIN)/squawk.tmp https://github.com/sbdchd/squawk/releases/download/$*/squawk-$(SQUAWK_PLATFORM)
	echo "$(SQUAWK_SHA256_$(SQUAWK_PLATFORM))  $(BIN)/squawk.tmp" | sha256sum -c - 2>/dev/null \
		|| echo "$(SQUAWK_SHA256_$(SQUAWK_PLATFORM))  $(BIN)/squawk.tmp" | shasum -a 256 -c -
	chmod +x $(BIN)/squawk.tmp && mv $(BIN)/squawk.tmp $(BIN)/squawk
	@touch $@

hooks: tools ## Install the lefthook git hooks
	lefthook install

fmt: tools ## Format all Go, frontend, and root docs/config in place
	cd backend && golangci-lint fmt ./...
	cd frontend && npm run format
	$(PRETTIER) --write $(ROOT_PRETTIER)

fmt-check: tools ## Fail if anything is unformatted
	cd backend && golangci-lint fmt --diff ./...
	cd frontend && npm run format:check
	$(PRETTIER) --check $(ROOT_PRETTIER)

lint: tools backend-lint ## Lint and type-check everything, scan for secrets
	cd frontend && npm run lint && npx tsc --noEmit && npm run knip
	squawk $(MIGRATE_DIR)/*.up.sql
	gitleaks git --no-banner --redact

test: backend-test ## Run all tests
	cd frontend && npm test

check: fmt-check lint test ## Everything CI runs; must pass before pushing

# ── Database ──────────────────────────────────────────────────────────────────

db: ## Start Postgres in Docker (detached)
	docker compose up -d postgres

db-stop: ## Stop Postgres container (data preserved)
	docker compose stop postgres

db-clean: ## Remove Postgres container AND data volume
	docker compose down -v

migrate: ## Run all pending up migrations
	$(MIGRATE_BIN) -path $(MIGRATE_DIR) -database "$(DB_URL)" up

migrate-down: ## Roll back the last migration
	$(MIGRATE_BIN) -path $(MIGRATE_DIR) -database "$(DB_URL)" down 1

# ── Backend ───────────────────────────────────────────────────────────────────

backend: ## Run the Go API server
	cd backend && go run -ldflags "$(GO_LDFLAGS)" ./cmd/server

backend-watch: ## Run the Go API server, restarting on file changes (needs wgo)
	cd backend && wgo run -ldflags "$(GO_LDFLAGS)" ./cmd/server

backend-build: ## Compile the Go binary to backend/bin/server
	cd backend && go build -ldflags "$(GO_LDFLAGS)" -o bin/server ./cmd/server

backend-test: ## Run Go tests
	cd backend && go test -race -shuffle=on ./...

backend-lint: tools ## Lint the Go code (golangci-lint) and scan for known vulnerabilities
	cd backend && golangci-lint run ./... && go tool govulncheck ./...

# ── Frontend ──────────────────────────────────────────────────────────────────

frontend-install: ## Install npm dependencies
	cd frontend && npm install

frontend: ## Start Vite dev server
	cd frontend && npm run dev

frontend-build: ## Build frontend for production (output: frontend/dist/)
	cd frontend && npm run build

# ── Deploy ────────────────────────────────────────────────────────────────────

deploy: backend-build frontend-build ## Build both and restart the deployed systemd unit
	systemctl --user restart nisaba
