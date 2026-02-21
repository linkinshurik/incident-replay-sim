package replay

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"incident-replay/backend/internal/scenario"
	"incident-replay/backend/internal/store"

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

const defaultLatencySamplesCap = 10000

var latencySamplesCap = loadLatencySamplesCap()

func loadLatencySamplesCap() int {
	if val, ok := os.LookupEnv("LATENCY_SAMPLES_CAP"); ok {
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			return v
		}
	}
	return defaultLatencySamplesCap
}

// Stats holds statistics data for a replay run
// p95ms is the 95th percentile latency in milliseconds
// We store latencies in a slice to compute p95

type Stats struct {
	Requests       int64   `json:"requests"`
	Errors         int64   `json:"errors"`
	LatencySamples []int64 // latency samples in ms
	latencyHead    int
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
		if len(s.LatencySamples) < latencySamplesCap {
			s.LatencySamples = append(s.LatencySamples, latencyMs)
		} else if latencySamplesCap > 0 {
			s.LatencySamples[s.latencyHead] = latencyMs
			s.latencyHead = (s.latencyHead + 1) % latencySamplesCap
		}
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

	idx := int(float64(len(samples)) * 0.95)
	if idx > 0 {
		idx--
	}
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

	store      *store.RunStore
	report     store.Report
	finishedAt *time.Time
	// semaphore release func for concurrency control
	releaseSlot func()
}

func NewRun(runID string, runStore *store.RunStore) *Run {
	return &Run{
		RunID:     runID,
		State:     StateRunning,
		StartedAt: time.Now().UTC(),
		Stats:     &Stats{},
		store:     runStore,
		report:    make(store.Report),
	}
}

// persist saves the current report state to store
func (r *Run) persist() {
	if r.store == nil {
		return
	}
	_ = r.store.Save(r.RunID, r.report)
}

// MarkStopped marks the run as stopped
func (r *Run) MarkStopped() {
	gauge := promReplayRunsActive

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.State == StateRunning {
		r.State = StateStopped
		if r.cancel != nil {
			r.cancel()
		}
		now := time.Now().UTC()
		r.finishedAt = &now

		// update report
		r.report["state"] = r.State
		r.report["finishedAt"] = r.finishedAt.Format(time.RFC3339)
		r.persist()

		if r.releaseSlot != nil {
			r.releaseSlot()
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
	now := time.Now().UTC()
	r.finishedAt = &now

	// update report
	r.report["state"] = r.State
	r.report["finishedAt"] = r.finishedAt.Format(time.RFC3339)
	if r.releaseSlot != nil {
		r.releaseSlot()
	}

	r.persist()

	promReplayRunsActive.Dec()
}

// AddSample adds a latency sample and error flag to stats
func (r *Run) AddSample(latencyMs int64, isError bool) {
	r.Stats.AddSample(latencyMs, isError)

	// Update report fields for stats
	r.mu.Lock()
	defer r.mu.Unlock()

	r.report["stats"] = map[string]interface{}{
		"requests": r.Stats.Requests,
		"errors":   r.Stats.Errors,
		"p95ms":    r.Stats.P95(),
	}
	r.persist()
}

// UpdateP95 recalculates the p95 for the run stats
func (r *Run) UpdateP95() {
	r.Stats.CalculateP95()

	// update report
	r.mu.Lock()
	defer r.mu.Unlock()

	r.report["stats"] = map[string]interface{}{
		"requests": r.Stats.Requests,
		"errors":   r.Stats.Errors,
		"p95ms":    r.Stats.P95(),
	}
	r.persist()
}

// StatusResponse returns the JSON-compatible status response
func (r *Run) StatusResponse() map[string]interface{} {
	resp := map[string]interface{}{
		"runId":     r.RunID,
		"state":     r.State,
		"startedAt": r.StartedAt.Format(time.RFC3339),
		"stats": map[string]interface{}{
			"requests": r.Stats.Requests,
			"errors":   r.Stats.Errors,
			"p95ms":    r.Stats.P95(),
		},
	}
	if r.finishedAt != nil {
		resp["finishedAt"] = r.finishedAt.Format(time.RFC3339)
	}
	return resp
}

// Runner manages active replay runs
// Stores runs in memory mapped by runID

type Runner struct {
	runs       map[string]*Run
	mu         sync.Mutex
	httpClient *http.Client
	runStore   *store.RunStore
	// concurrency limiting semaphore (channel) for max concurrent runs
	semaphore chan struct{}
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

func NewRunner() *Runner {
	maxConcurrent := 3 // default
	if val, ok := os.LookupEnv("MAX_CONCURRENT_RUNS"); ok {
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			maxConcurrent = v
		}
	}

	return &Runner{
		runs:       make(map[string]*Run),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		runStore:   store.NewRunStore(),
		semaphore:  make(chan struct{}, maxConcurrent),
	}
}

func parseTimestamp(ts string) (time.Time, error) {
	if ts == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, ts)
}

// TryAcquireSlot attempts to acquire a concurrency slot, returns true if acquired, false if no slots available
func (r *Runner) TryAcquireSlot() bool {
	select {
	case r.semaphore <- struct{}{}:
		return true
	default:
		return false
	}
}

// ReleaseSlot releases a previously acquired concurrency slot
func (r *Runner) ReleaseSlot() {
	select {
	case <-r.semaphore:
		// slot released
	default:
		// nothing to release
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

	// Validate timestamps format StartFromTs, EndAtTs if not empty
	startFrom, err := parseTimestamp(params.StartFromTs)
	if err != nil {
		return "", fmt.Errorf("invalid startFromTs: %w", err)
	}
	endAt, err := parseTimestamp(params.EndAtTs)
	if err != nil {
		return "", fmt.Errorf("invalid endAtTs: %w", err)
	}

	if !startFrom.IsZero() && !endAt.IsZero() && !(startFrom.Before(endAt) || startFrom.Equal(endAt)) {
		return "", errors.New("startFromTs must be before endAtTs")
	}

	// Acquire concurrency semaphore slot
	if !r.TryAcquireSlot() {
		return "", errors.New("too_many_concurrent_runs")
	}

	// Load scenario events
	rawEvents, err := scenario.LoadScenario(params.ScenarioID)
	if err != nil {
		r.ReleaseSlot()
		return "", fmt.Errorf("load scenario error: %w", err)
	}

	// In timestamp mode, we require events to have valid ts and sort them
	var events []scenario.Event
	if params.Mode == "timestamp" {
		return "", errors.New("timestamp mode is not implemented yet")
	} else {
		// For burst mode, use raw events
		events = rawEvents
	}

	runID := fmt.Sprintf("run-%d", time.Now().UTC().UnixNano())

	// update prometheus counter for requests
	promReplayRequestsTotal.Inc()

	// Create run and start the goroutine
	run := NewRun(runID, r.runStore)
	ctx, cancel := context.WithCancel(context.Background())
	run.cancel = cancel
	// assign release function
	run.releaseSlot = func() {
		r.ReleaseSlot()
	}

	r.mu.Lock()
	r.runs[runID] = run
	r.mu.Unlock()

	// Initial report state
	run.mu.Lock()
	run.report["runId"] = runID
	run.report["state"] = run.State
	run.report["startedAt"] = run.StartedAt.Format(time.RFC3339)
	run.report["stats"] = map[string]interface{}{
		"requests": 0,
		"errors":   0,
		"p95ms":    0,
	}
	run.mu.Unlock()
	run.persist()

	// update active runs gauge
	promReplayRunsActive.Inc()

	go r.runLoad(ctx, run, params, events)

	return runID, nil
}

// runLoad runs the replay load according to provided parameters
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
				// run was stopped explicitly; MarkStopped is called by Stop/MarkStopped caller
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
					run.AddSample(latencyMs, true)
					promReplayErrorsTotal.Inc()
					continue
				}

				_ = resp.Body.Close()

				isErr := resp.StatusCode >= 400
				run.AddSample(latencyMs, isErr)
				if isErr {
					promReplayErrorsTotal.Inc()
				}
			}
		}
	}
}

// StartParams holds parameters for starting a replay run
type StartParams struct {
	ScenarioID    string
	TargetBaseURL string
	RPS           int
	Duration      time.Duration
	Mode          string
	Speed         float64
	MaxDelayMs    int64
	StartFromTs   string
	EndAtTs       string
}

// Status returns the current status for a run
func (r *Runner) Status(runID string) (*Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	run, ok := r.runs[runID]
	if !ok {
		return nil, fmt.Errorf("run not found")
	}
	return run, nil
}

// Stop stops a running run by ID
func (r *Runner) Stop(runID string) error {
	r.mu.Lock()
	run, ok := r.runs[runID]
	r.mu.Unlock()

	if !ok {
		return fmt.Errorf("run not found")
	}

	run.MarkStopped()
	return nil
}

// StopAll stops all running runs and marks them as failed with the given reason.
// The reason is not currently persisted but can be used in future report extensions.
func (r *Runner) StopAll(reason string) {
	_ = reason // reserved for future use

	r.mu.Lock()
	runsCopy := make([]*Run, 0, len(r.runs))
	for _, run := range r.runs {
		runsCopy = append(runsCopy, run)
	}
	r.mu.Unlock()

	for _, run := range runsCopy {
		run.MarkFailed()
	}
}

// ListRuns returns up to limit run reports from store
func (r *Runner) ListRuns(limit int) ([]map[string]interface{}, error) {
	ids, err := r.runStore.List(limit)
	if err != nil {
		return nil, err
	}

	var runs []map[string]interface{}
	for _, id := range ids {
		rep, err := r.runStore.Load(id)
		if err != nil {
			continue
		}
		if rep != nil {
			// ensure required fields, or skip if missing
			if _, ok := rep["runId"]; !ok {
				rep["runId"] = id
			}
			runs = append(runs, rep)
		}
	}
	return runs, nil
}

// LoadReport loads a report for a given runID from store
func (r *Runner) LoadReport(runID string) (store.Report, error) {
	return r.runStore.Load(runID)
}
