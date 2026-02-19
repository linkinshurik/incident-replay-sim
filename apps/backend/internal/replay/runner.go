package replay

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"incident-replay/backend/internal/scenario"

	"github.com/prometheus/client_golang/prometheus"
)

// ReplayState represents the current state of a replay run
//go:generate stringer -type=ReplayState

type ReplayState string

const (
	StateRunning ReplayState = "running"
	StateStopped ReplayState = "stopped"
	StateFailed  ReplayState = "failed"
)

// Stats holds statistics data for a replay run
// p95ms is the 95th percentile latency in milliseconds
// We store latencies in a slice to compute p95

type Stats struct {
	Requests       int64   `json:"requests"`
	Errors         int64   `json:"errors"`
	LatencySamples []int64 // latency samples in ms
	p95ms          int64
	mu             sync.Mutex
}

// AddSample adds a latency sample (in milliseconds) and updates counters.
func (s *Stats) AddSample(latencyMs int64, isError bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Requests++
	if isError {
		s.Errors++
	}
	// Only store samples of successful requests for latency calc
	if !isError {
		s.LatencySamples = append(s.LatencySamples, latencyMs)
	}
}

// CalculateP95 calculates the 95th percentile latency from stored samples
func (s *Stats) CalculateP95() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.LatencySamples) == 0 {
		s.p95ms = 0
		return
	}

	// Copy and sort to find p95
	samples := make([]int64, len(s.LatencySamples))
	copy(samples, s.LatencySamples)
	sort.Slice(samples, func(i, j int) bool {
		return samples[i] < samples[j]
	})

	idx := int(float64(len(samples))*0.95) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(samples) {
		idx = len(samples) - 1
	}
	s.p95ms = samples[idx]
}

// P95 returns the last computed p95 latency
func (s *Stats) P95() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.p95ms
}

// Run represents a single replay execution
// Stores state, start time, and stats

type Run struct {
	RunID     string
	State     ReplayState
	StartedAt time.Time
	Stats     *Stats
	mu        sync.Mutex
	// control
	cancel context.CancelFunc
}

func NewRun(runID string) *Run {
	return &Run{
		RunID:     runID,
		State:     StateRunning,
		StartedAt: time.Now().UTC(),
		Stats:     &Stats{},
	}
}

// MarkStopped marks the run as stopped
func (r *Run) MarkStopped() {
	// Mark as stopped only if running
	// Cancel the context if any
	// Decrement active runs gauge on stop
	gauge := promReplayRunsActive

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.State == StateRunning {
		r.State = StateStopped
		if r.cancel != nil {
			r.cancel()
		}
		gauge.Dec()
	}
}

// MarkFailed marks the run as failed
func (r *Run) MarkFailed() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.State = StateFailed
	if r.cancel != nil {
		r.cancel()
	}
	promReplayRunsActive.Dec()
}

// AddSample adds a latency sample and error flag to stats
func (r *Run) AddSample(latencyMs int64, isError bool) {
	r.Stats.AddSample(latencyMs, isError)
}

// UpdateP95 recalculates the p95 for the run stats
func (r *Run) UpdateP95() {
	r.Stats.CalculateP95()
}

// StatusResponse returns the JSON-compatible status response
func (r *Run) StatusResponse() map[string]interface{} {
	return map[string]interface{}{
		"runId":     r.RunID,
		"state":     r.State,
		"startedAt": r.StartedAt.Format(time.RFC3339),
		"stats": map[string]interface{}{
			"requests": r.Stats.Requests,
			"errors":   r.Stats.Errors,
			"p95ms":    r.Stats.P95(),
		},
	}
}

// Runner manages active replay runs
// Stores runs in memory mapped by runID

type Runner struct {
	runs       map[string]*Run
	mu         sync.Mutex
	httpClient *http.Client
}

// Prometheus metrics for replay
var (
	promReplayRequestsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "replay_requests_total",
			Help: "Total number of replay requests started.",
		},
	)
	promReplayErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "replay_errors_total",
			Help: "Total number of errors occurred during replay.",
		},
	)
	promReplayRunsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "replay_runs_active",
			Help: "Current number of active replay runs.",
		},
	)
	promReplayRequestDurationMs = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "replay_request_duration_ms",
			Help:    "Histogram of replay request durations in milliseconds.",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10),
		},
	)
)

func init() {
	prometheus.MustRegister(promReplayRequestsTotal)
	prometheus.MustRegister(promReplayErrorsTotal)
	prometheus.MustRegister(promReplayRunsActive)
	prometheus.MustRegister(promReplayRequestDurationMs)
}

type StartParams struct {
	ScenarioID    string
	TargetBaseURL string
	RPS           int
	Duration      time.Duration

	// New optional fields
	Mode        string  // burst|timestamp
	Speed       float64 // >0
	MaxDelayMs  int64   // >=0
	StartFromTs string  // RFC3339 timestamp
	EndAtTs     string  // RFC3339 timestamp
}

type StatusStats struct {
	Requests int64 `json:"requests"`
	Errors   int64 `json:"errors"`
	P95ms    int64 `json:"p95ms"`
}

type Status struct {
	RunID     string      `json:"runId"`
	State     ReplayState `json:"state"`
	StartedAt string      `json:"startedAt"`
	Stats     StatusStats `json:"stats"`
}

func NewRunner() *Runner {
	return &Runner{
		runs:       make(map[string]*Run),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (r *Runner) Start(params StartParams) (string, error) {
	if strings.TrimSpace(params.TargetBaseURL) == "" {
		return "", errors.New("targetBaseUrl is required")
	}
	if params.RPS <= 0 {
		return "", errors.New("rps must be > 0")
	}
	if params.Duration <= 0 {
		return "", errors.New("duration must be > 0")
	}
	if strings.TrimSpace(params.ScenarioID) == "" {
		return "", errors.New("scenarioId is required")
	}

	if params.Mode == "" {
		params.Mode = "burst"
	}

	// Validate mode
	if params.Mode != "burst" && params.Mode != "timestamp" {
		return "", errors.New("mode must be 'burst' or 'timestamp'")
	}

	if params.Speed != 0 && params.Speed <= 0 {
		return "", errors.New("speed must be > 0")
	}

	if params.MaxDelayMs < 0 {
		return "", errors.New("maxDelayMs must be >= 0")
	}

	// TODO: Validate timestamps format StartFromTs, EndAtTs if not empty

	// Load scenario events
	events, err := scenario.LoadScenario(params.ScenarioID)
	if err != nil {
		return "", fmt.Errorf("load scenario error: %w", err)
	}

	runID := fmt.Sprintf("run-%d", time.Now().UTC().UnixNano())

	// update prometheus counter for requests
	promReplayRequestsTotal.Inc()

	// Create run and start the goroutine
	run := NewRun(runID)
	ctx, cancel := context.WithCancel(context.Background())
	run.cancel = cancel

	r.mu.Lock()
	r.runs[runID] = run
	r.mu.Unlock()

	// update active runs gauge
	promReplayRunsActive.Inc()

	go r.runLoad(ctx, run, params, events)

	return runID, nil
}

func (r *Runner) runLoad(ctx context.Context, run *Run, params StartParams, events []scenario.Event) {
	baseURL := strings.TrimRight(params.TargetBaseURL, "/")

	if len(events) == 0 {
		// No events to run, mark failed and return
		run.MarkFailed()
		return
	}

	rps := params.RPS
	dur := params.Duration

	if params.Mode == "burst" {
		// Burst mode: send requests spaced by interval = 1/rps
		interval := time.Second / time.Duration(rps) // spacing between requests
		if interval == 0 {
			interval = time.Millisecond * 1
		}

		end := time.Now().Add(dur)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		randSrc := rand.NewSource(time.Now().UnixNano())
		randGen := rand.New(randSrc)

		eventsLen := len(events)

		for {
			select {
			case <-ctx.Done():
				run.MarkStopped()
				return
			case now := <-ticker.C:
				if now.After(end) {
					// Time exceeded stop run
					run.MarkStopped()
					return
				}

				start := time.Now()

				// choose event weighted
				ev := events[randGen.Intn(eventsLen)]

				url := baseURL + ev.Path

				var req *http.Request
				var err error

				if ev.Body != "" {
					req, err = http.NewRequest(ev.Method, url, strings.NewReader(ev.Body))
				} else {
					req, err = http.NewRequest(ev.Method, url, nil)
				}
				if err != nil {
					latencyMs := time.Since(start).Milliseconds()
					run.AddSample(latencyMs, true)
					promReplayErrorsTotal.Inc()
					continue
				}

				// add headers
				for k, v := range ev.Headers {
					req.Header.Set(k, v)
				}

				resp, err := r.httpClient.Do(req)
				latencyMs := time.Since(start).Milliseconds()

				promReplayRequestDurationMs.Observe(float64(latencyMs))

				if err != nil {
					// Record error sample
					run.AddSample(0, true)
					promReplayErrorsTotal.Inc()
					continue
				}

				_ = resp.Body.Close()

				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					// Non 2xx status
					run.AddSample(latencyMs, true)
					promReplayErrorsTotal.Inc()
					continue
				}

				// success
				run.AddSample(latencyMs, false)
			}
		}
	} else if params.Mode == "timestamp" {
		// TODO: Implement timestamp mode respecting StartFromTs, EndAtTs, Speed, MaxDelayMs
		// For now, fail fallback
		run.MarkFailed()
		return
	} else {
		// Unknown mode
		run.MarkFailed()
		return
	}
}

func (r *Runner) Stop(runID string) error {
	if strings.TrimSpace(runID) == "" {
		return errors.New("runId is required")
	}
	if ok := r.StopRun(runID); !ok {
		return errors.New("run not found")
	}
	return nil
}

func (r *Runner) Status(runID string) (Status, error) {
	if strings.TrimSpace(runID) == "" {
		return Status{}, errors.New("runId is required")
	}

	run, ok := r.GetRun(runID)
	if !ok {
		return Status{}, errors.New("run not found")
	}

	// Calculate latest p95
	// Defensive: run.Stats.CalculateP95()
	run.UpdateP95()

	// Snapshot values
	run.Stats.mu.Lock()
	requests := run.Stats.Requests
	errorsCount := run.Stats.Errors
	p95 := run.Stats.p95ms
	run.Stats.mu.Unlock()

	return Status{
		RunID:     run.RunID,
		State:     run.State,
		StartedAt: run.StartedAt.Format(time.RFC3339),
		Stats: StatusStats{
			Requests: requests,
			Errors:   errorsCount,
			P95ms:    p95,
		},
	}, nil
}

// StartRun creates and stores a new run
func (r *Runner) StartRun(runID string) *Run {
	r.mu.Lock()
	defer r.mu.Unlock()
	run := NewRun(runID)
	r.runs[runID] = run
	return run
}

// StopRun marks a run as stopped and cancels its context
func (r *Runner) StopRun(runID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[runID]
	if !ok {
		return false
	}
	run.MarkStopped()

	// update active runs gauge
	promReplayRunsActive.Dec()

	return true
}

// GetRun fetches a run by ID
func (r *Runner) GetRun(runID string) (*Run, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[runID]
	return run, ok
}

// Simulate recording a latency sample or error (for usage)
// This is just an example; actual runner would call AddSample
func (r *Runner) RecordSample(runID string, latencyMs int64, isError bool) {
	if run, ok := r.GetRun(runID); ok {
		run.AddSample(latencyMs, isError)
	}
}

// PeriodicUpdate updates p95 for all runs - should be called periodically
func (r *Runner) PeriodicUpdate() {
	r.mu.Lock()
	runs := make([]*Run, 0, len(r.runs))
	for _, run := range r.runs {
		runs = append(runs, run)
	}
	r.mu.Unlock()

	for _, run := range runs {
		run.UpdateP95()
	}
}
