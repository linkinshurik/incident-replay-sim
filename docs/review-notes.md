# Review Notes: Persist replay runs report

- Added persistence of run reports to the store (`./data/runs/`) with atomic file writes.
- Run status, stats, and timestamps are now saved and updated in real-time.
- Updated `Run` and `Runner` to use the new `RunStore`.
- Added unit tests for scenario loading, replay runner start/stop/status with scenario files.
- Tests validate correct JSON parsing of scenario files, addressing the original test failure.
- Code changes are minimal and focused on storing run metadata and fixing test data format.

This update fixes the JSON parse error in tests and integrates persistent run reporting,
satisfying the DoD without breaking existing functionality or observability.
