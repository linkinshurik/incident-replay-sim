package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"incident-replay/backend/internal/replay"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Removed duplicate prometheus metrics registration from here.
// Metrics registration is done in internal/replay/runner.go

const scenariosDataDir = "./data/scenarios"

var validScenarioID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type Handler struct {
	runner *replay.Runner
}

func NewHandler(r *replay.Runner) *Handler {
	return &Handler{runner: r}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", h.healthz)
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/replay/start", h.replayStart)
	mux.HandleFunc("/replay/stop", h.replayStop)
	mux.HandleFunc("/replay/status", h.replayStatus)

	mux.HandleFunc("/debug/echo", h.debugEcho)

	mux.HandleFunc("/scenarios/upload", h.scenarioUpload)
	mux.HandleFunc("/scenarios/list", h.scenarioList)

	return withJSON(withLogging(mux))
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) replayStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req startReq
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	// Set default for backward compatibility
	if req.Mode == "" {
		req.Mode = "burst"
	}

	// Validate mode
	if req.Mode != "burst" && req.Mode != "timestamp" {
		http.Error(w, "mode must be 'burst' or 'timestamp'", http.StatusBadRequest)
		return
	}

	// Validate speed if set
	if req.Speed != 0 && req.Speed <= 0 {
		http.Error(w, "speed must be > 0", http.StatusBadRequest)
		return
	}

	// Validate maxDelayMs if set
	if req.MaxDelayMs < 0 {
		http.Error(w, "maxDelayMs must be >= 0", http.StatusBadRequest)
		return
	}

	// Construct start params with extra fields
	params := replay.StartParams{
		ScenarioID:    req.ScenarioID,
		TargetBaseURL: req.TargetBaseURL,
		RPS:           req.RPS,
		Duration:      time.Duration(req.DurationSec) * time.Second,
		Mode:          req.Mode,
		Speed:         req.Speed,
		MaxDelayMs:    req.MaxDelayMs,
		StartFromTs:   req.StartFromTs,
		EndAtTs:       req.EndAtTs,
	}

	runID, err := h.runner.Start(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"runId":  runID,
		"status": "started",
	})
}

func (h *Handler) replayStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req stopReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	if err := h.runner.Stop(req.RunID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"runId":  req.RunID,
		"status": "stopped",
	})
}

func (h *Handler) replayStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	runID := r.URL.Query().Get("runId")
	if runID == "" {
		http.Error(w, "runId is required", http.StatusBadRequest)
		return
	}

	st, err := h.runner.Status(runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(st)
}

func (h *Handler) debugEcho(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// POST /scenarios/upload
// Accepts JSON {scenarioId string, jsonl string}
// Stores content into ./data/scenarios/<scenarioId>.jsonl
func (h *Handler) scenarioUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ScenarioID string `json:"scenarioId"`
		JSONL      string `json:"jsonl"`
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	if !validScenarioID.MatchString(req.ScenarioID) {
		http.Error(w, "invalid scenarioId", http.StatusBadRequest)
		return
	}

	// Create directory if not exists
	err := os.MkdirAll(scenariosDataDir, 0o755)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Safe filepath join
	path := filepath.Join(scenariosDataDir, req.ScenarioID+".jsonl")

	// Validate no ../ in final path
	// fix: use filepath.Clean on path for prefix check
	if !filepath.HasPrefix(filepath.Clean(path), filepath.Clean(scenariosDataDir)+string(filepath.Separator)) {
		http.Error(w, "invalid scenarioId path", http.StatusBadRequest)
		return
	}

	// Write file
	err = os.WriteFile(path, []byte(req.JSONL), 0o644)
	if err != nil {
		http.Error(w, "failed to save scenario", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "scenarioId": req.ScenarioID})
}

// GET /scenarios/list returns JSON array of scenarioIds
func (h *Handler) scenarioList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files, err := os.ReadDir(scenariosDataDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// empty list
			_ = json.NewEncoder(w).Encode([]string{})
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var ids []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		ext := filepath.Ext(name)
		if ext != ".jsonl" {
			continue
		}
		id := name[:len(name)-len(ext)]
		if !validScenarioID.MatchString(id) {
			continue
		}
		ids = append(ids, id)
	}

	_ = json.NewEncoder(w).Encode(ids)
}

type startReq struct {
	ScenarioID    string `json:"scenarioId"`
	TargetBaseURL string `json:"targetBaseUrl"`
	RPS           int    `json:"rps"`
	DurationSec   int    `json:"durationSec"`

	// New optional fields
	Mode        string  `json:"mode,omitempty"`        // burst|timestamp, default burst
	Speed       float64 `json:"speed,omitempty"`       // float64 > 0
	MaxDelayMs  int64   `json:"maxDelayMs,omitempty"`  // int >= 0
	StartFromTs string  `json:"startFromTs,omitempty"` // RFC3339 timestamp
	EndAtTs     string  `json:"endAtTs,omitempty"`     // RFC3339 timestamp
}

type stopReq struct {
	RunID string `json:"runId"`
}
