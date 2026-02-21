package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"incident-replay/backend/internal/replay"
	"incident-replay/backend/internal/scenario"
	"incident-replay/backend/internal/store"
)

// Handler aggregates all HTTP handlers for the backend API.
type Handler struct {
	mux       *http.ServeMux
	scenarios *scenario.Service
	runs      *replay.Runner
	store     store.RunStore
}

// NewHandler constructs a new Handler and wires all routes.
func NewHandler(scenarios *scenario.Service, runs *replay.Runner, store store.RunStore) *Handler {
	h := &Handler{
		mux:       http.NewServeMux(),
		scenarios: scenarios,
		runs:      runs,
		store:     store,
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
// If not writable, returns 503 with {"status":"degraded","error":"..."}.
// Otherwise returns 200 with {"status":"ok"}.
func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	base := "data"
	scenariosDir := filepath.Join(base, "scenarios")
	runsDir := filepath.Join(base, "runs")

	if err := ensureWritableDir(r.Context(), scenariosDir); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "degraded",
			"error":  fmt.Sprintf("scenarios dir not writable: %v", err),
		})
		return
	}

	if err := ensureWritableDir(r.Context(), runsDir); err != nil {
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
// If the directory does not exist, it attempts to create it with 0755 perms.
// It returns an error if the directory cannot be created or written to.
func ensureWritableDir(ctx context.Context, path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}

	// Quick writable check: create and remove a temp file in the directory.
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

	if err := h.scenarios.Save(r.Context(), req.ScenarioID, []byte(req.JSONL)); err != nil {
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

	list, err := h.scenarios.List(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
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

func (h *Handler) handleReplayStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req startReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	opts := replay.StartOptions{
		ScenarioID:  req.ScenarioID,
		TargetBase:  req.TargetBase,
		RPS:         req.RPS,
		DurationSec: req.DurationSec,
		Mode:        req.Mode,
		Speed:       req.Speed,
		MaxDelayMs:  req.MaxDelayMs,
		StartFromTs: req.StartFromTs,
		EndAtTs:     req.EndAtTs,
	}

	runID, err := h.runs.Start(r.Context(), opts)
	if err != nil {
		if errors.Is(err, replay.ErrInvalidOptions) {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
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

	if err := h.runs.Stop(r.Context(), req.RunID); err != nil {
		if errors.Is(err, replay.ErrRunNotFound) {
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

	status, err := h.runs.Status(r.Context(), runID)
	if err != nil {
		if errors.Is(err, replay.ErrRunNotFound) {
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

	list, err := h.runs.List(r.Context())
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

	report, err := h.store.Load(r.Context(), runID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(report)
}
