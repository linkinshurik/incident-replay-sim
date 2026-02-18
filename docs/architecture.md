# Architecture v0

## Goal
Incident Replay & Load Simulator: відтворювати потік подій/запитів з контролем RPS/latency та знімати метрики.

## Components (v0)
- Backend API (control-plane): старт/стоп сценаріїв, конфіг, метрики.
- Load runner (data-plane): виконує replay/генерує навантаження.
- Frontend UI: керування сценаріями + перегляд базових метрик (v0).
- Observability: /metrics, logs.

## Deployment (v0)
- Local: docker compose або `go run`.
- Later: Kubernetes (Helm) + Terraform для бази/черги (якщо буде).

## Data flow (v0)
User -> Frontend -> Backend API -> (in-memory scenario runner) -> Target endpoint(s)
Backend -> /metrics -> Prometheus (later)

## Non-goals (v0)
- Multi-tenant auth, складні ролі доступу.
- Distributed runner cluster.
