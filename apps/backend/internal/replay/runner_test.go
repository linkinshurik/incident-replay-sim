package replay

import (
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
