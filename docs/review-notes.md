# Review Notes: JSONL Parser and Weighted Scenario Events

- Added new package `apps/backend/internal/scenario` implementing JSONL parser for scenario events as specified in `docs/events.md`.
- Supports parsing of method, path, headers, body, and weight fields with validation of event type == "http".
- Weight is used to expand event occurrences via a weighted pool implementation (simple expansion by replication).
- Exposes function `LoadScenario(scenarioId string) ([]Event, error)` to load and expand scenario events.
- Includes unit tests covering valid parse cases, invalid JSON handling, empty file, missing file, and scenarioId validation.
- Changes meet DoD requirements: code builds, passes formatting, linting, tests, and smoke tests.
- No security issues or secrets introduced; scenario loading is file-based under controlled directory.
- Documentation updates reflect correct event format and API surface; no breaking changes noted.