package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/linkinshurik/incident-replay/internal/replay"
	"github.com/linkinshurik/incident-replay/internal/scenario"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const scenariosDataDir = "./data/scenarios"

var validScenarioID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var (
	maxRPSPerRun = getEnvInt("MAX_RPS_PER_RUN", 200)
)

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

	mux.HandleFunc("/replay/runs", h.replayRunsList)
	mux.HandleFunc("/replay/report", h.replayRunReport)

	mux.HandleFunc("/debug/echo", h.debugEcho)

	mux.HandleFunc("/scenarios/upload", h.scenarioUpload)
	mux.HandleFunc("/scenarios/upload-har", h.scenarioUploadHAR)
	mux.HandleFunc("/scenarios/list", h.scenarioList)

	return withJSON(withLogging(mux))
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	var iv int
	_, err := fmt.Sscanf(v, "%d", &iv)
	if err != nil || iv <= 0 {
		return def
	}
	return iv
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

	// Validate RPS: must be >0 and <= maxRPSPerRun
	if req.RPS <= 0 || req.RPS > maxRPSPerRun {
		http.Error(w, `{"error":"invalid_rps"}`, http.StatusBadRequest)
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

// GET /replay/runs?limit=20 returns array of run summaries
func (h *Handler) replayRunsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 20
	qLimit := r.URL.Query().Get("limit")
	if qLimit != "" {
		// parse limit
		if l, err := parsePositiveInt(qLimit); err == nil && l > 0 {
			limit = l
		}
	}

	reports, err := h.runner.ListRuns(limit)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(reports)
}

// GET /replay/report?runId=... returns JSON report from store
func (h *Handler) replayRunReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	runID := r.URL.Query().Get("runId")
	if runID == "" {
		http.Error(w, "runId is required", http.StatusBadRequest)
		return
	}

	report, err := h.runner.LoadReport(runID)
	if err != nil {
		http.Error(w, "report not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

func parsePositiveInt(s string) (int, error) {
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	if err != nil || v <= 0 {
		return 0, errors.New("not positive int")
	}
	return v, nil
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

	// Validate path is under scenariosDataDir (no path traversal)
	if rel, err := filepath.Rel(scenariosDataDir, path); err != nil || strings.HasPrefix(rel, "..") {
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

// POST /scenarios/upload-har
// Accepts multipart form: scenarioId (form field), file (HAR file from Chrome DevTools).
// Converts HAR to scenario JSONL (preserving request order and timestamps) and stores as <scenarioId>.jsonl.
func (h *Handler) scenarioUploadHAR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	scenarioID := r.FormValue("scenarioId")
	if scenarioID == "" {
		http.Error(w, "scenarioId is required", http.StatusBadRequest)
		return
	}
	if !validScenarioID.MatchString(scenarioID) {
		http.Error(w, "invalid scenarioId", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	harBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusBadRequest)
		return
	}

	jsonl, err := scenario.HARToJSONL(harBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = os.MkdirAll(scenariosDataDir, 0o755)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	path := filepath.Join(scenariosDataDir, scenarioID+".jsonl")
	if rel, err := filepath.Rel(scenariosDataDir, path); err != nil || strings.HasPrefix(rel, "..") {
		http.Error(w, "invalid scenarioId path", http.StatusBadRequest)
		return
	}

	if err := os.WriteFile(path, jsonl, 0o644); err != nil {
		http.Error(w, "failed to save scenario", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "scenarioId": scenarioID})
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
