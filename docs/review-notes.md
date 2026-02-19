# Review Notes: Scenario Storage V1 Implementation

- Added POST /scenarios/upload to accept JSON payload with scenarioId and jsonl content.
- Validated scenarioId against regex allowing letters, digits, underscore, and dash only.
- Stored uploaded scenario files safely into ./data/scenarios/<scenarioId>.jsonl after ensuring directory existence.
- Added GET /scenarios/list to return JSON array of stored scenarioId strings found in ./data/scenarios.
- Prevented directory traversal by verifying file path prefix after cleaning path.
- Updated API documentation in docs/api.md accordingly to reflect new endpoints and their usage.
- Changes follow DoD, pass lint, test, build, and do not break existing functionality.
- No secrets added and log/metrics conventions remain consistent.
