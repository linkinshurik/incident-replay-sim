# Incident Replay & Load Simulator - Product Documentation

## Goal
Develop a tool to replay incidents and simulate load by reproducing a stream of events or HTTP requests with controllable request rate (RPS) and latency. The system should collect and expose metrics for observability.

## Scope
- Backend API to control replay scenarios: start, stop, status.
- Load runner for generating replayed load based on event streams.
- Frontend UI for managing scenarios and viewing basic metrics.
- Observability features including metrics endpoint and structured logs.
- Deployment support for local environment (Docker Compose, go run).

## Non-Scope
- Multi-tenant authentication or complex role-based access controls.
- Distributed runner clusters or horizontally scalable runners in v0.

## User Stories

1. **As a user, I want to start a replay scenario by specifying scenario ID, target URL, RPS, and duration, so that I can simulate load based on a recorded incident.**

2. **As a user, I want to stop an ongoing replay run to immediately halt the load simulation when needed.**

3. **As a user, I want to check the status of a running or stopped replay including statistics like total requests, errors, and latency percentiles, so I can monitor load test progress.**

4. **As a developer, I want the backend API to expose `/healthz` and `/metrics` endpoints for health checks and monitoring integration.**

5. **As an operator, I want to deploy the system locally using Docker Compose or direct Go commands for easy testing and development.**
