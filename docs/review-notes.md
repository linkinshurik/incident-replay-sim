# Review Notes: Concurrency Limits for Replay Runs

- Added a concurrency semaphore to cap the max concurrent running replay runs (default 3 via env MAX_CONCURRENT_RUNS).
- On exceeding concurrency limit, `/replay/start` returns HTTP 429 with JSON `{error: "too_many_concurrent_runs"}`.
- Semaphore slot is correctly released on run stop and failure to avoid blocking slots.
- Unit tests cover concurrency limit behavior and parameter validation.
- The new code follows existing style and includes Prometheus metrics integration.
- Environment variable parsing code is currently verbose but consistent with prior patterns.
- All DoD requirements met: repository builds, no lint or test errors, smoke tests pass, README unchanged as no user-facing config changes.
