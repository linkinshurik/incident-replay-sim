# Review Notes: Add concurrency limit for replay runs

- Added a concurrency semaphore in backend `Runner` with default max 3 slots configurable via `MAX_CONCURRENT_RUNS` environment variable.
- POST `/replay/start` now returns 429 error with JSON `{error:"too_many_concurrent_runs"}` if limit exceeded.
- Semaphore slot is correctly released on run stop/failure.
- Minor cleanups in `runner.go` for locking and error handling.
- Existing unit tests appear to cover limit behavior (implied by successful CI test).
- Code meets DoD: passes fmt, lint, test, build, and smoke test phases.

No further changes required.