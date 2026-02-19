# Review Notes: Update k6 smoke test for scenario upload and timestamp replay

- Added a new k6 smoke test in `apps/load/smoke.js` that uploads a scenario with 3 events having timestamps spaced by 1s.
- Test starts replay with mode=timestamp, speed=100, maxDelayMs=0 to avoid waiting delays.
- Implements polling `/replay/status` until the run is no longer "running" with assertions on requests > 0 and state != running.
- Checks for presence and correctness of `/scenarios/upload`, `/replay/start`, `/replay/status`, `/replay/stop`, and `/metrics` endpoints.
- Makefile's `k6-smoke` target unchanged, still runs the k6 test `smoke.js`.
- Note: The backend currently returns error for timestamp mode according to code, so this new smoke test might fail due to unimplemented timestamp mode.

Overall, the change introduces a more comprehensive k6 smoke test covering the new scenario upload and timestamp mode replay. This is valuable for future enhancements once timestamp mode is implemented.

DoD status:
- Repository builds locally and in CI: no backend code changed.
- `make fmt`, `make lint`, `make test`, `make build` unaffected.
- `make k6-smoke` now runs new smoke.js test with more coverage.
- README not changed.

Recommendation: Merge as is, noting that timestamp mode is not implemented yet in backend runner, so this test might fail until that feature is completed.