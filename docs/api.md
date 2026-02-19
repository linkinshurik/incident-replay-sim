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
  "durationSec": 60,
  "mode": "burst|timestamp",       # optional, default "burst"
  "speed": 1.0,                   # optional, > 0
  "maxDelayMs": 0,                # optional, >= 0
  "startFromTs": "RFC3339",    # optional, start timestamp
  "endAtTs": "RFC3339"          # optional, end timestamp
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

## Scenario Storage

POST `/scenarios/upload`
Accept JSON:
{
  "scenarioId": "string",
  "jsonl": "string"
}
- `scenarioId` must be safe, only letters, digits, underscore, or dash
- Stores the content into `./data/scenarios/<scenarioId>.jsonl`

Response:
- 200 OK
{
  "status": "ok",
  "scenarioId": "string"
}

GET `/scenarios/list`
Returns a list of stored `scenarioId`s

Response:
- 200 OK
[
  "scenario1",
  "scenario2"
]
