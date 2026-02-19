package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"incident-replay/backend/internal/replay"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Removed duplicate prometheus metrics registration from here.
// Metrics registration is done in internal/replay/runner.go

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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	runID, err := h.runner.Start(replay.StartParams{
		ScenarioID:    req.ScenarioID,
		TargetBaseURL: req.TargetBaseURL,
		RPS:           req.RPS,
		Duration:      time.Duration(req.DurationSec) * time.Second,
	})
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

type startReq struct {
	ScenarioID    string `json:"scenarioId"`
	TargetBaseURL string `json:"targetBaseUrl"`
	RPS           int    `json:"rps"`
	DurationSec   int    `json:"durationSec"`
}

type stopReq struct {
	RunID string `json:"runId"`
}
