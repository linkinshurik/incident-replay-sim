package replay

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

type StartParams struct {
	ScenarioID    string
	TargetBaseURL string
	RPS           int
	Duration      time.Duration
}

type StatusResponse struct {
	RunID     string    `json:"runId"`
	State     string    `json:"state"`
	StartedAt time.Time `json:"startedAt"`
	Stats     Stats     `json:"stats"`
}

type Stats struct {
	Requests int64 `json:"requests"`
	Errors   int64 `json:"errors"`
	P95ms    int64 `json:"p95ms"`
}

type run struct {
	startedAt time.Time
	state     string
	stats     Stats
	stopCh    chan struct{}
}

type Runner struct {
	mu   sync.Mutex
	runs map[string]*run
}

func NewRunner() *Runner {
	return &Runner{runs: map[string]*run{}}
}

func (r *Runner) Start(p StartParams) (string, error) {
	if p.TargetBaseURL == "" {
		return "", errors.New("targetBaseUrl is required")
	}
	if p.RPS <= 0 {
		return "", errors.New("rps must be > 0")
	}
	if p.Duration <= 0 {
		return "", errors.New("durationSec must be > 0")
	}

	id := uuid.NewString()

	r.mu.Lock()
	r.runs[id] = &run{
		startedAt: time.Now().UTC(),
		state:     "running",
		stopCh:    make(chan struct{}),
	}
	r.mu.Unlock()

	// v0: тут поки не робимо реальний load, лише автозупинка по duration
	go func() {
		timer := time.NewTimer(p.Duration)
		defer timer.Stop()

		select {
		case <-timer.C:
			_ = r.Stop(id)
		case <-r.getStopCh(id):
			return
		}
	}()

	return id, nil
}

func (r *Runner) Stop(runID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rr, ok := r.runs[runID]
	if !ok {
		return errors.New("run not found")
	}
	if rr.state != "running" {
		return nil
	}
	rr.state = "stopped"
	close(rr.stopCh)
	return nil
}

func (r *Runner) Status(runID string) (StatusResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rr, ok := r.runs[runID]
	if !ok {
		return StatusResponse{}, errors.New("run not found")
	}

	return StatusResponse{
		RunID:     runID,
		State:     rr.state,
		StartedAt: rr.startedAt,
		Stats:     rr.stats,
	}, nil
}

func (r *Runner) getStopCh(runID string) <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rr, ok := r.runs[runID]; ok {
		return rr.stopCh
	}
	ch := make(chan struct{})
	close(ch)
	return ch
}
