SHELL := /bin/bash
.DEFAULT_GOAL := help

BACKEND_DIR := apps/backend
FRONTEND_DIR := apps/frontend
LOAD_DIR := apps/load

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS=":.*## "}; {printf "  %-18s %s\n", $$1, $$2}'

fmt: ## Format all code (Go + Frontend)
	@echo "==> fmt (backend)"
	@if [ -f "$(BACKEND_DIR)/go.mod" ]; then (cd $(BACKEND_DIR) && gofmt -w .); else echo "skip: no go.mod"; fi
	@echo "==> fmt (frontend)"
	@if [ -f "$(FRONTEND_DIR)/package.json" ]; then (cd $(FRONTEND_DIR) && npm run fmt || true); else echo "skip: no package.json"; fi

lint: ## Lint all (Go + Frontend)
	@echo "==> lint (backend)"
	@if [ -f "$(BACKEND_DIR)/go.mod" ]; then (cd $(BACKEND_DIR) && go vet ./...); else echo "skip: no go.mod"; fi
	@echo "==> lint (frontend)"
	@if [ -f "$(FRONTEND_DIR)/package.json" ]; then (cd $(FRONTEND_DIR) && npm run lint || true); else echo "skip: no package.json"; fi

test: ## Run tests (Go + Frontend)
	@echo "==> test (backend)"
	@if [ -f "$(BACKEND_DIR)/go.mod" ]; then (cd $(BACKEND_DIR) && go test ./...); else echo "skip: no go.mod"; fi
	@echo "==> test (frontend)"
	@if [ -f "$(FRONTEND_DIR)/package.json" ]; then (cd $(FRONTEND_DIR) && npm test || true); else echo "skip: no package.json"; fi

build: ## Build artifacts (backend + frontend)
	@echo "==> build (backend)"
	@if [ -f "$(BACKEND_DIR)/go.mod" ]; then (cd $(BACKEND_DIR) && go build ./...); else echo "skip: no go.mod"; fi
	@echo "==> build (frontend)"
	@if [ -f "$(FRONTEND_DIR)/package.json" ]; then (cd $(FRONTEND_DIR) && npm run build || true); else echo "skip: no package.json"; fi

run-backend: ## Run backend locally (expects apps/backend cmd/main.go later)
	@echo "==> run backend"
	@cd $(BACKEND_DIR) && go run ./cmd/server

k6-smoke: ## Run k6 smoke test (expects k6 script later)
	@echo "==> k6 smoke"
	@if command -v k6 >/dev/null 2>&1; then \
		if [ -f "$(LOAD_DIR)/smoke.js" ]; then (cd $(LOAD_DIR) && k6 run -u 1 -d 30s smoke.js); \
		else echo "skip: no $(LOAD_DIR)/smoke.js"; fi \
	else \
		echo "skip: k6 not installed"; \
	fi

ci: fmt lint test build k6-smoke ## Full local CI pipeline
	@echo "==> CI OK"
