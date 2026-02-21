# Review Notes: Add observability v2 metrics for replay runs

- Implemented new Prometheus metrics in `internal/replay`: `replay_runs_total{mode}`, `replay_runs_failed_total{mode,reason}`, `replay_run_duration_seconds{mode}` histogram, and `replay_active_runs` gauge, keeping existing per-request metrics for backward compatibility.
- Wired metrics updates into the replay lifecycle: incrementing runs_total on start, updating active_runs gauge on start/stop/fail, and recording duration plus failed_total with a normalized `mode` label and low-cardinality `reason`.
- Adjusted `Run` and `Runner` to track mode per run and to centralize completion handling so durations are measured once per run without introducing high-cardinality labels.
- Updated `Stop`, `StopAll`, `Status`, and `ListRuns` logic to align with the new Runner/Run responsibilities while preserving API behavior and persistence via `RunStore`.
- Added targeted tests in `internal/replay/runner_test.go` to cover shutdown flag behavior, StopAll failure persistence, and metric recording on run completion, while keeping existing tests green under `make ci`.
- CI log in `.agent.last_ci.txt` shows `make ci` passing end-to-end (fmt, lint, test, build, k6-smoke), satisfying DoD requirements for this change set.