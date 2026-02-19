package replay

import (
	"sync"
	"testing"
	"time"
)

func TestStartValidation(t *testing.T) {
	r := NewRunner()

	if _, err := r.Start(StartParams{TargetBaseURL: "", RPS: 1, Duration: time.Second}); err == nil {
		t.Fatalf("expected error for empty target")
	}
	if _, err := r.Start(StartParams{TargetBaseURL: "http://x", RPS: 0, Duration: time.Second}); err == nil {
		t.Fatalf("expected error for rps")
	}
	if _, err := r.Start(StartParams{TargetBaseURL: "http://x", RPS: 1, Duration: 0}); err == nil {
		t.Fatalf("expected error for duration")
	}
}

func TestStartStopStatus(t *testing.T) {
	r := NewRunner()

	id, err := r.Start(StartParams{TargetBaseURL: "http://x", RPS: 1, Duration: time.Second})
	if err != nil {
		t.Fatalf("start err: %v", err)
	}

	st, err := r.Status(id)
	if err != nil || st.State != "running" {
		t.Fatalf("status expected running, got %+v err=%v", st, err)
	}

	if err := r.Stop(id); err != nil {
		t.Fatalf("stop err: %v", err)
	}

	st, _ = r.Status(id)
	if st.State != "stopped" {
		t.Fatalf("expected stopped, got %s", st.State)
	}
}

func TestStatsAddSampleConcurrent(t *testing.T) {
	run := NewRun("test-run")

	wg := sync.WaitGroup{}
	n := 1000

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			latencyMs := int64(i % 100)
			isError := i%10 == 0
			run.AddSample(latencyMs, isError)
		}(i)
	}
	wg.Wait()

	if run.Stats.Requests != int64(n) {
		t.Fatalf("expected %d requests, got %d", n, run.Stats.Requests)
	}
	expectedErrors := int64(n / 10)
	if run.Stats.Errors != expectedErrors {
		t.Fatalf("expected %d errors, got %d", expectedErrors, run.Stats.Errors)
	}

	// Check latency samples count
	if len(run.Stats.LatencySamples) != n-int(expectedErrors) {
		t.Fatalf("expected %d latency samples, got %d", n-int(expectedErrors), len(run.Stats.LatencySamples))
	}
}

func TestCalculateP95(t *testing.T) {

	stats := &Stats{}

	// Add 100 samples 1..100 ms
	for i := int64(1); i <= 100; i++ {
		stats.AddSample(i, false)
	}

	stats.CalculateP95()

	if stats.p95ms != 95 {
		t.Fatalf("expected p95 95, got %d", stats.p95ms)
	}

	// Add some errors, should not affect latency samples
	for i := 0; i < 10; i++ {
		stats.AddSample(0, true)
	}

	stats.CalculateP95()

	if stats.p95ms != 95 {
		t.Fatalf("expected p95 95 after errors, got %d", stats.p95ms)
	}
}
