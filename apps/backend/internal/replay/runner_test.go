package replay

import (
	"os"
	"path/filepath"
	"testing"
)

func withTempWD(t *testing.T) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir failed: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore wd failed: %v", err)
		}
	})
}

func TestRunnerStopAllMarksFailedAndPersists(t *testing.T) {
	withTempWD(t)

	r := NewRunner()
	runID := "run-stopall-1"
	run := NewRun(runID, r.runStore)

	r.mu.Lock()
	r.runs[runID] = run
	r.mu.Unlock()

	r.StopAll("shutdown")

	if run.State != StateFailed {
		t.Fatalf("expected state %q, got %q", StateFailed, run.State)
	}
	if run.finishedAt == nil {
		t.Fatalf("expected finishedAt to be set")
	}

	report, err := r.runStore.Load(runID)
	if err != nil {
		t.Fatalf("expected persisted report, got error: %v", err)
	}
	if got, ok := report["state"].(string); !ok || got != string(StateFailed) {
		t.Fatalf("expected persisted state %q, got %#v", StateFailed, report["state"])
	}

	reportPath := filepath.Join("data", "runs", runID+".json")
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("expected report file %s to exist: %v", reportPath, err)
	}
}

func TestRunnerShutdownFlag(t *testing.T) {
	r := NewRunner()
	if r.IsShuttingDown() {
		t.Fatalf("expected shutdown flag false by default")
	}

	r.BeginShutdown()

	if !r.IsShuttingDown() {
		t.Fatalf("expected shutdown flag true after BeginShutdown")
	}
}
