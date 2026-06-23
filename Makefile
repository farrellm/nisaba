.PHONY: help db db-stop db-clean migrate migrate-down backend backend-watch backend-build backend-test frontend-install frontend frontend-build

MIGRATE_BIN := migrate
MIGRATE_DIR := db/migrations
DB_URL      := postgres://nisaba:nisaba@localhost:5432/nisaba?sslmode=disable

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

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
	cd backend && go run ./cmd/server

backend-watch: ## Run the Go API server, restarting on file changes (needs wgo)
	cd backend && wgo run ./cmd/server

backend-build: ## Compile the Go binary to backend/bin/server
	cd backend && go build -o bin/server ./cmd/server

backend-test: ## Run Go tests
	cd backend && go test ./...

# ── Frontend ──────────────────────────────────────────────────────────────────

frontend-install: ## Install npm dependencies
	cd frontend && npm install

frontend: ## Start Vite dev server
	cd frontend && npm run dev

frontend-build: ## Build frontend for production (output: frontend/dist/)
	cd frontend && npm run build
