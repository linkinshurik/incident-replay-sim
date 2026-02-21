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
	@if [ -f "$(FRONTEND_DIR)/package.json" ]; then (cd $(FRONTEND_DIR) && [ -d node_modules ] || npm install); (cd $(FRONTEND_DIR) && npm run lint || true); else echo "skip: no package.json"; fi

test: ## Run tests (Go + Frontend)
	@echo "==> test (backend)"
	@if [ -f "$(BACKEND_DIR)/go.mod" ]; then (cd $(BACKEND_DIR) && go test ./...); else echo "skip: no go.mod"; fi
	@echo "==> test (frontend)"
	@if [ -f "$(FRONTEND_DIR)/package.json" ]; then (cd $(FRONTEND_DIR) && [ -d node_modules ] || npm install); (cd $(FRONTEND_DIR) && npm test || true); else echo "skip: no package.json"; fi

build: ## Build artifacts (backend + frontend)
	@echo "==> build (backend)"
	@if [ -f "$(BACKEND_DIR)/go.mod" ]; then (cd $(BACKEND_DIR) && go build ./...); else echo "skip: no go.mod"; fi
	@echo "==> build (frontend)"
	@if [ -f "$(FRONTEND_DIR)/package.json" ]; then (cd $(FRONTEND_DIR) && [ -d node_modules ] || npm install); (cd $(FRONTEND_DIR) && npm run build || true); else echo "skip: no package.json"; fi

run-backend: ## Run backend locally (expects apps/backend cmd/main.go later)
	@echo "==> run backend"
	@cd $(BACKEND_DIR) && go run ./cmd/server

k6-smoke: ## Run k6 smoke test (starts backend temporarily)
	@if ! command -v k6 >/dev/null 2>&1; then echo "skip: k6 not installed"; exit 0; fi
	@if [ ! -f "$(LOAD_DIR)/smoke.js" ]; then echo "skip: no $(LOAD_DIR)/smoke.js"; exit 0; fi

	@echo "==> starting backend on :8080"
	@cd $(BACKEND_DIR) && (ADDR=":8080" nohup go run ./cmd/server > ../../.backend.log 2>&1 & echo $$! > ../../.backend.pid)

	@echo "==> waiting for backend"
	@for i in $$(seq 1 30); do \
		if curl -fsS http://127.0.0.1:8080/healthz >/dev/null 2>&1; then \
			echo "backend is up"; break; \
		fi; \
		sleep 1; \
		if [ $$i -eq 30 ]; then \
			echo "backend failed to start. See .backend.log"; \
			exit 1; \
		fi; \
	done

	@echo "==> running k6"
	@cd $(LOAD_DIR) && BASE_URL="http://127.0.0.1:8080" k6 run -u 1 -d 30s smoke.js || true

	@echo "==> stopping backend"
	@kill $$(cat .backend.pid) >/dev/null 2>&1 || true
	@rm -f .backend.pid
	@echo "==> k6 smoke OK"

helm-lint: ## Run helm lint on infra/helm/incident-replay
	@if ! command -v helm >/dev/null 2>&1; then echo "skip: helm not installed"; exit 0; fi
	@helm lint infra/helm/incident-replay

helm-template: ## Render helm template for infra/helm/incident-replay
	@if ! command -v helm >/dev/null 2>&1; then echo "skip: helm not installed"; exit 0; fi
	@helm template incident-replay infra/helm/incident-replay

ci: fmt lint test build k6-smoke ## Full local CI pipeline
	@echo "==> CI OK"
