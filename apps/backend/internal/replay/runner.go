package replay

import (
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/linkinshurik/incident-replay/internal/store"
)

const defaultLatencySamplesCap = 10000

// StartParams describes a replay start request.
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

// Stats tracks run counters and p95 over bounded latency samples.
type Stats struct {
	mu       sync.Mutex
	samples  []int64
	cap      int
	next     int
	filled   int
	p95      int64
	Requests int
	Errors   int
}

func NewStatsWithCap(capacity int) *Stats {
	if capacity <= 0 {
		capacity = defaultLatencySamplesCap
	}
	return &Stats{
		samples: make([]int64, capacity),
		cap:     capacity,
	}
}

func (s *Stats) AddSample(latencyMs int64, isError bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Requests++
	if isError {
		s.Errors++
	}

	s.samples[s.next] = latencyMs
	s.next = (s.next + 1) % s.cap
	if s.filled < s.cap {
		s.filled++
	}
}

func (s *Stats) CalculateP95() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.filled == 0 {
		s.p95 = 0
		return
	}

	data := make([]int64, s.filled)
	copy(data, s.samples[:s.filled])
	sort.Slice(data, func(i, j int) bool { return data[i] < data[j] })

	idx := int(math.Floor(float64(len(data))*0.95)) - 1
	if len(data) <= 3 {
		idx = 0
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(data) {
		idx = len(data) - 1
	}
	s.p95 = data[idx]
}

func (s *Stats) P95() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.p95
}

// Run is a single replay run state.
type Run struct {
	ID        string
	CreatedAt time.Time
	Status    string
	Stats     *Stats
}

func NewRun(id string, _ interface{}) *Run {
	return &Run{
		ID:        id,
		CreatedAt: time.Now().UTC(),
		Status:    "running",
		Stats:     NewStatsWithCap(readLatencyCap()),
	}
}

func (r *Run) AddSample(latencyMs int64, isError bool) {
	r.Stats.AddSample(latencyMs, isError)
}

func (r *Run) UpdateP95() {
	r.Stats.CalculateP95()
}

// RunStatus is returned by the status endpoint.
type RunStatus struct {
	RunID    string `json:"runId"`
	Status   string `json:"status"`
	Requests int    `json:"requests"`
	Errors   int    `json:"errors"`
	P95Ms    int64  `json:"p95Ms"`
}

// Runner owns in-memory run lifecycle and report persistence.
type Runner struct {
	mu       sync.RWMutex
	runs     map[string]*Run
	runStore *store.RunStore
}

func NewRunner() *Runner {
	return &Runner{
		runs:     make(map[string]*Run),
		runStore: store.NewRunStore(),
	}
}

func (r *Runner) Start(params StartParams) (string, error) {
	if params.ScenarioID == "" {
		return "", errors.New("scenarioId is required")
	}
	if params.TargetBaseURL == "" {
		return "", errors.New("targetBaseUrl is required")
	}
	if params.RPS <= 0 {
		return "", errors.New("rps must be > 0")
	}
	if params.Duration <= 0 {
		return "", errors.New("durationSec must be > 0")
	}

	runID := fmt.Sprintf("run-%d", time.Now().UTC().UnixNano())
	run := NewRun(runID, nil)

	r.mu.Lock()
	r.runs[runID] = run
	r.mu.Unlock()

	if err := r.runStore.Save(runID, store.Report{
		"runId":     runID,
		"status":    run.Status,
		"createdAt": run.CreatedAt.Format(time.RFC3339Nano),
		"requests":  0,
		"errors":    0,
		"p95Ms":     0,
	}); err != nil {
		return "", err
	}

	return runID, nil
}

func (r *Runner) Stop(runID string) error {
	if runID == "" {
		return errors.New("runId is required")
	}

	r.mu.Lock()
	run, ok := r.runs[runID]
	if !ok {
		r.mu.Unlock()
		return errors.New("run not found")
	}
	run.Status = "stopped"
	run.UpdateP95()
	status := r.statusFromRun(run)
	r.mu.Unlock()

	return r.runStore.Save(runID, store.Report{
		"runId":     status.RunID,
		"status":    status.Status,
		"createdAt": run.CreatedAt.Format(time.RFC3339Nano),
		"requests":  status.Requests,
		"errors":    status.Errors,
		"p95Ms":     status.P95Ms,
	})
}

func (r *Runner) Status(runID string) (RunStatus, error) {
	if runID == "" {
		return RunStatus{}, errors.New("runId is required")
	}

	r.mu.RLock()
	run, ok := r.runs[runID]
	r.mu.RUnlock()
	if !ok {
		return RunStatus{}, errors.New("run not found")
	}
	return r.statusFromRun(run), nil
}

func (r *Runner) ListRuns(limit int) ([]store.Report, error) {
	ids, err := r.runStore.List(limit)
	if err != nil {
		return nil, err
	}

	reports := make([]store.Report, 0, len(ids))
	for _, id := range ids {
		rep, err := r.runStore.Load(id)
		if err != nil {
			continue
		}
		reports = append(reports, rep)
	}
	return reports, nil
}

func (r *Runner) LoadReport(runID string) (store.Report, error) {
	return r.runStore.Load(runID)
}

func (r *Runner) statusFromRun(run *Run) RunStatus {
	run.Stats.mu.Lock()
	req := run.Stats.Requests
	errCount := run.Stats.Errors
	p95 := run.Stats.p95
	run.Stats.mu.Unlock()

	return RunStatus{
		RunID:    run.ID,
		Status:   run.Status,
		Requests: req,
		Errors:   errCount,
		P95Ms:    p95,
	}
}

func readLatencyCap() int {
	v := os.Getenv("LATENCY_SAMPLES_CAP")
	if v == "" {
		return defaultLatencySamplesCap
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultLatencySamplesCap
	}
	return n
}
