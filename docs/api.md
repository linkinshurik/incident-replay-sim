# API v0

Base URL: `/`

## Health
GET `/healthz`
- 200 OK
- body: `{ "status": "ok" }`

## Metrics
GET `/metrics`
- 200 OK
- Prometheus text format (може бути заглушка на старті)

## Replay control
POST `/replay/start`
Request JSON:
{
  "scenarioId": "string",
  "targetBaseUrl": "string",
  "rps": 10,
  "durationSec": 60
}

Response:
- 200 OK
{
  "runId": "string",
  "status": "started"
}

POST `/replay/stop`
Request JSON:
{
  "runId": "string"
}

Response:
- 200 OK
{
  "runId": "string",
  "status": "stopped"
}

GET `/replay/status?runId=...`
Response:
- 200 OK
{
  "runId": "string",
  "state": "running|stopped|failed",
  "startedAt": "RFC3339",
  "stats": {
    "requests": 0,
    "errors": 0,
    "p95ms": 0
  }
}
