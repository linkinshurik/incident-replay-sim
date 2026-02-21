# Review Notes: Helm Chart Skeleton for Incident Replay

- Added Helm chart under `infra/helm/incident-replay` with skeleton files: `Chart.yaml`, `values.yaml`, and templates for backend/frontend deployments, services, helpers, and optional ingress.
- Backend container exposes port 8080 with properly configured readiness and liveness HTTP probes to `/healthz`.
- Frontend container exposes port 80 with similar readiness/liveness probes.
- Images for backend and frontend are configurable via `values.yaml` with repository, tag, and pullPolicy.
- Service types are configurable and default to `ClusterIP`.
- Ingress is optional and disabled by default; if enabled it routes to frontend service.
- Templates follow best practices including label usage and helper definitions for naming.
- Minimal configuration added as per DoD; no secrets/tokens embedded.
- No changes required; changes meet Definition of Done requirements and integrate well with existing docs and CI.
