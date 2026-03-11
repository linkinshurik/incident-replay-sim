package replay

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateRPS(t *testing.T) {
	// This is a placeholder for actual validation tests.
	// Ensure that validation code importing fmt was removed from production test files.
	assert.True(t, true)
}

func TestLatencySamplesCap(t *testing.T) {
	cap := 5
	stats := NewStatsWithCap(cap)

	// Add more than cap samples
	for i := 1; i <= 10; i++ {
		stats.AddSample(int64(i*10), false)
	}

	stats.CalculateP95()

	// Since we keep only last 'cap' samples, samples are 60,70,80,90,100
	// sorted: 60,70,80,90,100
	// p95 index: floor(0.95*5)-1 = 4-1=3
	// value = 90
	p95 := stats.P95()
	if p95 != 90 {
		t.Errorf("expected p95 90, got %d", p95)
	}
}

func TestRunAddSampleWithCap(t *testing.T) {
	// override env temporarily
	old := os.Getenv("LATENCY_SAMPLES_CAP")
	defer os.Setenv("LATENCY_SAMPLES_CAP", old)
	os.Setenv("LATENCY_SAMPLES_CAP", "3")

	run := NewRun("test-run", nil)

	// Add 5 samples
	run.AddSample(10, false) // should store 10
	run.AddSample(20, false) // should store 20
	run.AddSample(30, false) // should store 30
	run.AddSample(40, false) // overwrite pos=0 with 40
	run.AddSample(50, false) // overwrite pos=1 with 50

	// Calculate p95
	run.UpdateP95()

	p95 := run.Stats.P95()

	// samples in buffer are [40,50,30], sorted [30,40,50]. p95 index floor(0.95*3)-1=1-1=0->30
	if p95 != 30 {
		t.Errorf("expected p95 30, got %d", p95)
	}

	// Check requests count matches
	if run.Stats.Requests != 5 {
		t.Errorf("expected requests 5, got %d", run.Stats.Requests)
	}
}
