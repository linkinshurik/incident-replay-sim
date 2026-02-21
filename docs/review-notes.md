# Review Notes: Backend Dockerfile Addition

- Added a multi-stage Dockerfile for the Go backend, building a static Linux amd64 binary.
- Uses `golang:1.25-alpine` as a build image with git install for dependencies.
- Runtime image is minimal Alpine 3.18 with a non-root user `app`.
- Creates `/data/scenarios` and `/data/runs` directories owned by the non-root user.
- Exposes port 8080 and uses environment variables `ADDR` and `DATA_DIR`.
- Entrypoint runs the built binary directly.

This addition meets the DoD criteria for minimal viable Docker support with security best practices.
No changes needed.