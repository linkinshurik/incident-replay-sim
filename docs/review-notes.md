# Review Notes: Frontend Dockerfile Addition

- Added a multi-stage Dockerfile in `apps/frontend/Dockerfile`:
  - Stage 1 uses `node:18-alpine` to run `npm ci` and build the React app.
  - Stage 2 uses `nginx:stable-alpine` to serve the built app from `/usr/share/nginx/html`.
- Added `apps/frontend/nginx.conf` with minimal nginx configuration:
  - Serves static files with fallback to `index.html` for SPA routing.
  - Proxies `/replay/`, `/scenarios/`, `/healthz`, and `/metrics` endpoints to the backend service at `http://backend:8080`.
  - Proxy headers are set to preserve client info.
- This setup avoids CORS issues by proxying API requests through nginx.
- The changes are minimal, focused, and align with the task description and DoD.
- No issues detected; frontend containerization and proxying properly configured.