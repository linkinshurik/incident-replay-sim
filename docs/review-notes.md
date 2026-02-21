# Review Notes: Update Root Makefile with Docker Commands

- Added `docker-up` and `docker-down` targets to root Makefile.
- `docker-up` runs `docker compose up --build`, starting all services with build.
- `docker-down` runs `docker compose down -v`, stopping services and removing volumes.
- Existing targets remained intact and unchanged.
- Commands are documented with helpful comments in the Makefile.
- Changes align with DoD: no broken build, lint, tests; functionality is additive and non-breaking.
- No documentation outside Makefile changes needed as commands are straightforward and typical for Docker Compose usage.
