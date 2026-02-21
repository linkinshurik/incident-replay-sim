package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"incident-replay/backend/internal/replay"
	"incident-replay/backend/internal/scenario"
)

const (
	scenariosDirDefault = "data/scenarios"
	runsDirDefault      = "data/runs"
	defaultRunsLimit    = 20
	defaultMaxRPSPerRun = 200
)

// Handler aggregates all HTTP handlers for the backend API.
type Handler struct {
	mux  *http.ServeMux
	runs *replay.Runner
}

// NewHandler constructs a new Handler and wires all routes.
func NewHandler(runs *replay.Runner) *Handler {
	h := &Handler{
		mux:  http.NewServeMux(),
		runs: runs,
	}

	h.routes()
	return h
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) routes() {
	h.mux.HandleFunc("/healthz", h.handleHealthz)

	h.mux.HandleFunc("/scenarios/upload", h.handleScenariosUpload)
	h.mux.HandleFunc("/scenarios/list", h.handleScenariosList)

	h.mux.HandleFunc("/replay/start", h.handleReplayStart)
	h.mux.HandleFunc("/replay/stop", h.handleReplayStop)
	h.mux.HandleFunc("/replay/status", h.handleReplayStatus)
	h.mux.HandleFunc("/replay/runs", h.handleReplayRuns)
	h.mux.HandleFunc("/replay/report", h.handleReplayReport)
}

// handleHealthz checks that ./data/scenarios and ./data/runs exist and are writable.
func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	if err := ensureWritableDir(r.Context(), scenariosDirDefault); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "degraded",
			"error":  fmt.Sprintf("scenarios dir not writable: %v", err),
		})
		return
	}

	if err := ensureWritableDir(r.Context(), runsDirDefault); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "degraded",
			"error":  fmt.Sprintf("runs dir not writable: %v", err),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ensureWritableDir verifies that path exists as a directory and is writable.
func ensureWritableDir(ctx context.Context, path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}

	f, err := os.CreateTemp(path, ".healthcheck-*")
	if err != nil {
		return err
	}
	name := f.Name()
	cerr := f.Close()
	rerr := os.Remove(name)

	if cerr != nil {
		return cerr
	}
	if rerr != nil {
		return rerr
	}

	return ctx.Err()
}

// --- Scenarios ---

type uploadScenarioRequest struct {
	ScenarioID string `json:"scenarioId"`
	JSONL      string `json:"jsonl"`
}

func (h *Handler) handleScenariosUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req uploadScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}
	if !scenario.ValidScenarioID(req.ScenarioID) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid scenarioId"})
		return
	}
	if strings.TrimSpace(req.JSONL) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "jsonl is required"})
		return
	}

	if err := os.MkdirAll(scenariosDirDefault, 0o755); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	tmp := filepath.Join(scenariosDirDefault, req.ScenarioID+".jsonl.tmp")
	finalPath := filepath.Join(scenariosDirDefault, req.ScenarioID+".jsonl")
	if err := os.WriteFile(tmp, []byte(req.JSONL), 0o644); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if err := os.Rename(tmp, finalPath); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":     "ok",
		"scenarioId": req.ScenarioID,
	})
}

func (h *Handler) handleScenariosList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	entries, err := os.ReadDir(scenariosDirDefault)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]string{})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	items := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".jsonl" {
			continue
		}
		items = append(items, strings.TrimSuffix(name, ".jsonl"))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

// --- Replay ---

type startReplayRequest struct {
	ScenarioID  string  `json:"scenarioId"`
	TargetBase  string  `json:"targetBaseUrl"`
	RPS         int     `json:"rps"`
	DurationSec int     `json:"durationSec"`
	Mode        string  `json:"mode"`
	Speed       float64 `json:"speed"`
	MaxDelayMs  int     `json:"maxDelayMs"`
	StartFromTs string  `json:"startFromTs"`
	EndAtTs     string  `json:"endAtTs"`
}

type startReplayResponse struct {
	RunID  string `json:"runId"`
	Status string `json:"status"`
}

func maxRPSPerRun() int {
	if raw := os.Getenv("MAX_RPS_PER_RUN"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err == nil && v > 0 {
			return v
		}
	}
	return defaultMaxRPSPerRun
}

func (h *Handler) handleReplayStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if h.runs.IsShuttingDown() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "shutting_down"})
		return
	}

	var req startReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	if req.RPS <= 0 || req.RPS > maxRPSPerRun() {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_rps"})
		return
	}

	params := replay.StartParams{
		ScenarioID:    req.ScenarioID,
		TargetBaseURL: req.TargetBase,
		RPS:           req.RPS,
		Duration:      time.Duration(req.DurationSec) * time.Second,
		Mode:          req.Mode,
		Speed:         req.Speed,
		MaxDelayMs:    int64(req.MaxDelayMs),
		StartFromTs:   req.StartFromTs,
		EndAtTs:       req.EndAtTs,
	}

	runID, err := h.runs.Start(params)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(startReplayResponse{RunID: runID, Status: "started"})
}

type stopReplayRequest struct {
	RunID string `json:"runId"`
}

type stopReplayResponse struct {
	RunID  string `json:"runId"`
	Status string `json:"status"`
}

func (h *Handler) handleReplayStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req stopReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	if err := h.runs.Stop(req.RunID); err != nil {
		if strings.Contains(err.Error(), "run not found") {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stopReplayResponse{RunID: req.RunID, Status: "stopped"})
}

func (h *Handler) handleReplayStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	runID := r.URL.Query().Get("runId")
	if runID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "runId is required"})
		return
	}

	status, err := h.runs.Status(runID)
	if err != nil {
		if strings.Contains(err.Error(), "run not found") {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (h *Handler) handleReplayRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	limit := defaultRunsLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid limit"})
			return
		}
		limit = v
	}

	list, err := h.runs.ListRuns(limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (h *Handler) handleReplayReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	runID := r.URL.Query().Get("runId")
	if runID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "runId is required"})
		return
	}

	report, err := h.runs.LoadReport(runID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}
