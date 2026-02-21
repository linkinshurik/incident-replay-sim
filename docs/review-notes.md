# Review Notes: Root Docker Compose and README Update

- Added a root docker-compose.yml defining `backend` and `frontend` services, with ports and volumes as specified.
- Backend service builds from `apps/backend`, exposes port 8080, and mounts local `./data` volume to container `/data`.
- Frontend service builds from `apps/frontend`, exposes port 3000 mapped to container port 80, and depends on backend starting first.
- Frontend proxies API requests to backend internally, avoiding CORS issues; no extraneous volume mounts added for frontend.
- Updated top-level README.md with instructions on running services via `docker-compose`, exposed ports, data persistence, and relevant make commands.
- Changes meet DoD requirements: repo builds, tests, lint, k6 smoke pass; README adequately documents running via docker-compose.
- No issues detected; changes are minimal, clear, and align with product goals.