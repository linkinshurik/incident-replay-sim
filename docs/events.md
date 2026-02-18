# Events v0 (Replay input)

## Event format (JSONL)
Один рядок = один event.

{
  "ts": "RFC3339",
  "type": "http",
  "method": "GET|POST|PUT|DELETE",
  "path": "/api/...",
  "headers": { "k": "v" },
  "body": "base64-or-plain-string",
  "weight": 1
}

## Notes
- `weight` використовується для частоти (1..N).
- На v0 допускається: тільки method+path без body/headers.
