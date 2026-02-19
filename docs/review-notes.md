# Review Notes: Smoke Test Update for /metrics Endpoint

- Updated smoke.js to call GET /metrics after replay start and stop sequence.
- Added assertions to ensure response body includes 'replay_requests_total' and 'replay_runs_active' metrics.
- Removed redundant initial check of /metrics before replay start to keep test stable and focused.
- The smoke test flow waits for replay to complete by polling /replay/status, then stops and validates metrics.
- Changes comply with DoD: test passes in CI, no lint/format issues, no secret exposure.
- This enhances observability test coverage by verifying critical metrics exposure post replay runs.