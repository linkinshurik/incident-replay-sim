# Review Notes: Add persistence v1 with RunStore

- Added internal/store package for file-based persistence of replay run reports.
- RunStore writes JSON reports to ./data/runs/<runId>.json atomically using temp file + rename.
- Provided Save(runID, Report) and List(limit) methods with proper error handling and directory creation.
- Added unit tests covering Save, List, Load (extra), edge cases, and atomic write.
- Dependencies were kept minimal; only standard library packages used.
- Review against DoD:
  - Passes local build and CI (test coverage for new store package).
  - `make fmt`, `make lint`, `make test` succeeded.
  - No changes to README or Makefile since persistence addition is internal.
  - Observability unchanged (prometheus metrics in replay package).
  
This change meets the DoD for adding persistence v1 and can be merged as is.