package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"incident-replay/backend/internal/replay"
	"incident-replay/backend/internal/scenario"
)

const (
	defaultMaxRPS    = 200
	defaultRunsLimit = 20
	scenariosDir     = "./data/scenarios"
)

type Handler struct {
	runner *replay.Runner
}

func NewHandler(runner *replay.Runner) *Handler {
	return &Handler{runner: runner}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(withLogging)
	r.Use(withJSON)

	r.Get("/healthz", h.healthz)
	r.Handle("/metrics", promhttp.Handler())

	r.Post("/replay/start", h.replayStart)
	r.Post("/replay/stop", h.replayStop)
	r.Get("/replay/status", h.replayStatus)
	r.Get("/replay/runs", h.replayRuns)
	r.Get("/replay/report", h.replayReport)

	r.Post("/scenarios/upload", h.scenariosUpload)
	r.Get("/scenarios/list", h.scenariosList)

	return r
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) replayStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScenarioID    string  `json:"scenarioId"`
		TargetBaseURL string  `json:"targetBaseUrl"`
		RPS           int     `json:"rps"`
		DurationSec   int     `json:"durationSec"`
		Mode          string  `json:"mode"`
		Speed         float64 `json:"speed"`
		MaxDelayMs    int     `json:"maxDelayMs"`
		StartFromTs   string  `json:"startFromTs"`
		EndAtTs       string  `json:"endAtTs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	maxRPS := defaultMaxRPS
	if envMaxRPS := os.Getenv("MAX_RPS_PER_RUN"); envMaxRPS != "" {
		if val, err := strconv.Atoi(envMaxRPS); err == nil && val > 0 {
			maxRPS = val
		}
	}
	if req.RPS <= 0 || req.RPS > maxRPS {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_rps"})
		return
	}

	params := replay.StartParams{
		ScenarioID:    req.ScenarioID,
		TargetBaseURL: req.TargetBaseURL,
		RPS:           req.RPS,
		Duration:      time.Duration(req.DurationSec) * time.Second,
		Mode:          req.Mode,
		Speed:         req.Speed,
		MaxDelayMs:    int64(req.MaxDelayMs),
		StartFromTs:   req.StartFromTs,
		EndAtTs:       req.EndAtTs,
	}

	runID, err := h.runner.Start(params)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "too_many_concurrent_runs"):
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too_many_concurrent_runs"})
		case strings.Contains(msg, "required"), strings.Contains(msg, "must be"), strings.Contains(msg, "invalid"), strings.Contains(msg, "load scenario error"):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"runId":  runID,
		"status": "started",
	})
}

func (h *Handler) replayStop(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RunID string `json:"runId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.RunID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	if err := h.runner.Stop(req.RunID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"runId": req.RunID, "status": "stopped"})
}

func (h *Handler) replayStatus(w http.ResponseWriter, r *http.Request) {
	runID := strings.TrimSpace(r.URL.Query().Get("runId"))
	if runID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "runId is required"})
		return
	}
	status, err := h.runner.Status(runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) replayRuns(w http.ResponseWriter, r *http.Request) {
	limit := defaultRunsLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	runs, err := h.runner.ListRuns(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

func (h *Handler) replayReport(w http.ResponseWriter, r *http.Request) {
	runID := strings.TrimSpace(r.URL.Query().Get("runId"))
	if runID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "runId is required"})
		return
	}

	report, err := h.runner.LoadReport(runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "report_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *Handler) scenariosUpload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScenarioID string `json:"scenarioId"`
		JSONL      string `json:"jsonl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	if !scenario.ValidScenarioID(req.ScenarioID) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_scenario_id"})
		return
	}
	if strings.TrimSpace(req.JSONL) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "jsonl is required"})
		return
	}

	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	filename := filepath.Join(scenariosDir, req.ScenarioID+".jsonl")
	if err := os.WriteFile(filename, []byte(req.JSONL), 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":     "ok",
		"scenarioId": req.ScenarioID,
	})
}

func (h *Handler) scenariosList(w http.ResponseWriter, _ *http.Request) {
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusOK, []string{})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".jsonl" {
			continue
		}
		ids = append(ids, strings.TrimSuffix(name, ".jsonl"))
	}
	sort.Strings(ids)

	writeJSON(w, http.StatusOK, ids)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
