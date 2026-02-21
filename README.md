# Incident Replay & Load Simulator

## Overview
This repository contains the Incident Replay & Load Simulator application with backend and frontend components.

## Running with Docker Compose

To start backend and frontend services locally with Docker Compose, run:

```sh
docker-compose up --build
```

- Backend API server will be accessible at: `http://localhost:8080`
- Frontend UI will be accessible at: `http://localhost:3000`

The frontend proxies API requests (e.g., `/replay`, `/scenarios`, `/healthz`) to the backend automatically to avoid CORS issues.

## Data Persistence

The services share a local volume `./data` mounted inside the backend container at `/data` to store scenarios and replay runs.

## Makefile commands

You can also manage builds and tests using the Makefile:

```sh
make build   # build backend and frontend
make run-backend  # run backend locally
make test    # run backend and frontend tests
```

## Notes

- Frontend serves static files on port 3000 and proxies API to backend at port 8080.
- Backend listens on port 8080 and manages scenario storage in /data.
- Modify `docker-compose.yml` for advanced configuration and scaling.
