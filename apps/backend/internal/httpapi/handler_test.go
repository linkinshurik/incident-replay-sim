package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"incident-replay/backend/internal/replay"
)

func setupHandler(t *testing.T) *Handler {
	t.Helper()
	setupTestScenario(t)
	runner := replay.NewRunner()
	return NewHandler(runner)
}

func setupTestScenario(t *testing.T) {
	t.Helper()
	const dir = "./data/scenarios"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create scenarios dir: %v", err)
	}
	path := filepath.Join(dir, "testscenario.jsonl")
	content := `{"ts":"2021-01-01T00:00:00Z","type":"http","method":"GET","path":"/healthz"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(path)
	})
}

func TestHealthz_OKWhenDirsWritable(t *testing.T) {
	tmpDir := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	h := setupHandler(t)

	r := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	h.healthz(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body failed: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", body["status"])
	}

	// ensure dirs were created
	if _, err := os.Stat("./data/scenarios"); err != nil {
		t.Fatalf("expected scenarios dir to exist: %v", err)
	}
	if _, err := os.Stat("./data/runs"); err != nil {
		t.Fatalf("expected runs dir to exist: %v", err)
	}
}

func TestHealthz_DegradedWhenScenariosDirNotWritable(t *testing.T) {
	// create a temp dir and replace data/scenarios with a file so MkdirAll works but CreateTemp fails
	tmpDir := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	if err := os.MkdirAll("data", 0o755); err != nil {
		t.Fatalf("mkdir data failed: %v", err)
	}

	// create a file where scenarios dir should be; MkdirAll will succeed, CreateTemp will fail
	if err := os.WriteFile("data/scenarios", []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	h := setupHandler(t)

	r := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	h.healthz(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", res.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body failed: %v", err)
	}
	if body["status"] != "degraded" {
		t.Fatalf("expected status degraded, got %s", body["status"])
	}
	if body["error"] == "" {
		t.Fatalf("expected non-empty error in degraded response")
	}
}

func TestReplayStart_RPSValidation(t *testing.T) {
	h := setupHandler(t)

	// Test cases for invalid RPS
	tests := []struct {
		name     string
		rps      int
		wantErr  bool
		wantCode int
		wantBody string
	}{
		{"zero rps", 0, true, 400, `{"error":"invalid_rps"}`},
		{"negative rps", -1, true, 400, `{"error":"invalid_rps"}`},
		{"rps too large", 201, true, 400, `{"error":"invalid_rps"}`},
		{"rps equal max", 200, false, 200, ""},
		{"rps valid middle", 100, false, 200, ""},
	}

	// Save and restore env for MAX_RPS_PER_RUN
	orig := os.Getenv("MAX_RPS_PER_RUN")
	defer os.Setenv("MAX_RPS_PER_RUN", orig)

	err := os.Setenv("MAX_RPS_PER_RUN", "200")
	if err != nil {
		t.Fatalf("failed to set env: %v", err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"scenarioId":    "testscenario",
				"targetBaseUrl": "http://example.com",
				"rps":           tc.rps,
				"durationSec":   1,
			}
			buf := new(bytes.Buffer)
			err := json.NewEncoder(buf).Encode(reqBody)
			if err != nil {
				t.Fatalf("encode json failed: %v", err)
			}

			r := httptest.NewRequest("POST", "/replay/start", buf)
			w := httptest.NewRecorder()

			h.replayStart(w, r)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tc.wantCode {
				t.Fatalf("unexpected status code %d, want %d", res.StatusCode, tc.wantCode)
			}

			if tc.wantErr {
				var resp map[string]string
				err := json.NewDecoder(res.Body).Decode(&resp)
				if err != nil {
					t.Fatalf("failed to decode json error body: %v", err)
				}
				if resp["error"] != "invalid_rps" {
					t.Fatalf("expected error 'invalid_rps', got '%s'", resp["error"])
				}
			} else {
				// expect runId and status in response
				var resp map[string]string
				err := json.NewDecoder(res.Body).Decode(&resp)
				if err != nil {
					t.Fatalf("failed to decode json success body: %v", err)
				}
				if resp["status"] != "started" {
					t.Fatalf("expected status 'started', got '%s'", resp["status"])
				}
				if resp["runId"] == "" {
					t.Fatalf("expected non-empty runId")
				}
			}
		})
	}
}

func TestReplayStart_RPSValidation_EnvOverride(t *testing.T) {
	h := setupHandler(t)

	oldMaxRPS := os.Getenv("MAX_RPS_PER_RUN")
	defer os.Setenv("MAX_RPS_PER_RUN", oldMaxRPS)
	os.Setenv("MAX_RPS_PER_RUN", "50")

	// Reload maxRPSPerRun from environment
	maxRPS := 0
	val := os.Getenv("MAX_RPS_PER_RUN")
	if val != "" {
		v, err := strconv.Atoi(val)
		if err == nil && v > 0 {
			maxRPS = v
		} else {
			maxRPS = 200
		}
	} else {
		maxRPS = 200
	}

	if maxRPS != 50 {
		t.Fatalf("expected maxRPS to be 50, got %d", maxRPS)
	}

	// rps 51 should be invalid
	reqBody := map[string]interface{}{
		"scenarioId":    "testscenario",
		"targetBaseUrl": "http://example.com",
		"rps":           51,
		"durationSec":   1,
	}
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		t.Fatalf("encode json failed: %v", err)
	}

	r := httptest.NewRequest("POST", "/replay/start", buf)
	w := httptest.NewRecorder()

	h.replayStart(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != 400 {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}

	var resp map[string]string
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("decode json failed: %v", err)
	}

	if resp["error"] != "invalid_rps" {
		t.Fatalf("expected error invalid_rps, got %s", resp["error"])
	}
}

func TestReplayStart_ShuttingDownReturns503(t *testing.T) {
	setupTestScenario(t)
	runner := replay.NewRunner()
	runner.BeginShutdown()
	h := NewHandler(runner)

	reqBody := map[string]interface{}{
		"scenarioId":    "testscenario",
		"targetBaseUrl": "http://example.com",
		"rps":           1,
		"durationSec":   1,
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
		t.Fatalf("encode json failed: %v", err)
	}

	r := httptest.NewRequest("POST", "/replay/start", buf)
	w := httptest.NewRecorder()
	h.replayStart(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != 503 {
		t.Fatalf("expected status 503, got %d", res.StatusCode)
	}

	var resp map[string]string
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json failed: %v", err)
	}
	if resp["error"] != "shutting_down" {
		t.Fatalf("expected error shutting_down, got %s", resp["error"])
	}
}
