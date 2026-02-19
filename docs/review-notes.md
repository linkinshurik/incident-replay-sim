# Review Notes: Extend POST /replay/start API with new optional fields

- Added `mode` (burst|timestamp), `speed`, `maxDelayMs`, `startFromTs`, and `endAtTs` optional fields to `/replay/start` API and internal runner.
- Input validations for `speed` (>0) and `maxDelayMs` (>=0) performed at HTTP handler and runner start.
- Backward compatibility maintained: default `mode` is `burst` if omitted.
- Runner implements `burst` mode playback as before; `timestamp` mode is stubbed with immediate failure (to be implemented).
- Prometheus metrics correctly increment/decrement on run start and stop.
- Scenario file handling remains safe and validated.
- Updated `docs/api.md` reflects new parameters and validation rules.
- Tests cover input validation, start/stop/status functionality, and HTTP request replay correctness.

Overall, changes meet Definition of Done and project standards.