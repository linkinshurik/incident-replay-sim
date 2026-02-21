package replay

import (
	"context"
	"sync"
	"testing"
	"time"

	"incident-replay/backend/internal/store"
)

// existing tests omitted for brevity

func TestRunnerStopAllMarksFailed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	memStore := store.NewInMemory()

	r := NewRunner(memStore)

	// create a run and mark it as running
	runID := "run-stopall-1"
	req := RunRequest{
		RunID: runID,
	}

	// simulate a running run in the runner's internal state
	r.mu.Lock()
	r.runs[runID] = &Run{
		ID:        runID,
		State:     StateRunning,
		StartedAt: time.Now().Add(-1 * time.Minute),
	}
	r.mu.Unlock()

	// ensure nothing in store yet
	if _, err := memStore.GetReport(ctx, runID); err == nil {
		t.Fatalf("expected no report in store before StopAll, but found one")
	}

	// call StopAll with a reason
	reason := "shutdown"
	r.StopAll(reason)

	// check in-memory state
	r.mu.Lock()
	run, ok := r.runs[runID]
	r.mu.Unlock()
	if !ok {
		t.Fatalf("expected run %s to exist in runner after StopAll", runID)
	}
	if run.State != StateFailed {
		t.Fatalf("expected run state %s, got %s", StateFailed, run.State)
	}
	if run.FinishedAt.IsZero() {
		t.Fatalf("expected FinishedAt to be set after StopAll")
	}

	// check persisted report
	report, err := memStore.GetReport(ctx, runID)
	if err != nil {
		t.Fatalf("expected report to be persisted after StopAll, got error: %v", err)
	}
	if report.State != StateFailed {
		t.Fatalf("expected persisted report state %s, got %s", StateFailed, report.State)
	}
	if report.FinishedAt.IsZero() {
		t.Fatalf("expected persisted report FinishedAt to be set")
	}
	if report.Error == "" {
		t.Fatalf("expected persisted report to contain error reason, got empty string")
	}
}

// ensure concurrent safety when StopAll is called while runs are added
func TestRunnerStopAllConcurrent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	memStore := store.NewInMemory()
	r := NewRunner(memStore)

	var wg sync.WaitGroup

	// start goroutine that adds runs
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			id := "concurrent-run-" + time.Now().Format("150405.000000")
			r.mu.Lock()
			r.runs[id] = &Run{
				ID:        id,
				State:     StateRunning,
				StartedAt: time.Now(),
			}
			r.mu.Unlock()
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// give the goroutine a moment to start
	time.Sleep(5 * time.Millisecond)

	// call StopAll while runs might be added
	r.StopAll("shutdown")

	wg.Wait()

	// verify that any run present is not in running state
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, run := range r.runs {
		if run.State == StateRunning {
			// allow races where a run may have been added just after StopAll;
			// such runs won't have a persisted report yet, but must be non-running
			t.Fatalf("run %s left in running state after StopAll", id)
		}
	}

	// additionally, ensure persisted reports (if any) are not running
	reports, err := memStore.ListReports(ctx, 100)
	if err != nil {
		t.Fatalf("ListReports error: %v", err)
	}
	for _, rep := range reports {
		if rep.State == StateRunning {
			t.Fatalf("persisted report %s left in running state after StopAll", rep.ID)
		}
	}
}
