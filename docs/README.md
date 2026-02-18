# Incident Replay & Load Simulator

## Overview
This repository implements a tool for incident replay and load simulation. It allows to reproduce a stream of events or requests with control over request rate (RPS) and latency, while collecting metrics.

## Architecture (v0)
- **Backend API (control-plane)**: manages lifecycle of scenarios (start/stop), configuration, and exposes metrics.
- **Load runner (data-plane)**: executes replay and generates load toward target endpoints.
- **Frontend UI**: interface to manage scenarios and view basic metrics.
- **Observability**: exposes `/metrics` and structured logs.

## Deployment
- Locally: via Docker Compose or running Go applications directly.
- Future plans: Kubernetes deployment with Helm and Terraform for infrastructure.

## Data flow
User interacts with Frontend UI → Backend API → in-memory scenario runner → target endpoints.
Backend exposes metrics for Prometheus scraping.

## Important URLs
- Health: `/healthz`
- Metrics: `/metrics`
- Replay control endpoints: `/replay/start`, `/replay/stop`, `/replay/status`

For more detailed docs please see other markdown files in `docs/` folder.