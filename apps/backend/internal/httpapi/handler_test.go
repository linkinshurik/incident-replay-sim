package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"incident-replay/backend/internal/replay"
)

func withTempWD(t *testing.T) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
}

func setupHandler(t *testing.T) *Handler {
	t.Helper()
	return NewHandler(replay.NewRunner())
}

func TestHealthzOKWhenDirsWritable(t *testing.T) {
	withTempWD(t)
	h := setupHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	if _, err := os.Stat(filepath.Join("data", "scenarios")); err != nil {
		t.Fatalf("expected scenarios dir to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join("data", "runs")); err != nil {
		t.Fatalf("expected runs dir to exist: %v", err)
	}
}

func TestReplayStartRPSValidation(t *testing.T) {
	withTempWD(t)
	h := setupHandler(t)

	body := map[string]any{
		"scenarioId":    "s1",
		"targetBaseUrl": "http://example.com",
		"rps":           0,
		"durationSec":   1,
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		t.Fatalf("encode body failed: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/replay/start", buf)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}

	var resp map[string]string
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp["error"] != "invalid_rps" {
		t.Fatalf("expected error invalid_rps, got %q", resp["error"])
	}
}

func TestReplayStartShuttingDownReturns503(t *testing.T) {
	withTempWD(t)
	runner := replay.NewRunner()
	runner.BeginShutdown()
	h := NewHandler(runner)

	body := map[string]any{
		"scenarioId":    "s1",
		"targetBaseUrl": "http://example.com",
		"rps":           1,
		"durationSec":   1,
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		t.Fatalf("encode body failed: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/replay/start", buf)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", res.StatusCode)
	}

	var resp map[string]string
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp["error"] != "shutting_down" {
		t.Fatalf("expected error shutting_down, got %q", resp["error"])
	}
}
