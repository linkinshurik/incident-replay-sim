package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"incident-replay/backend/internal/replay"
)

func setupHandler() *Handler {
	runner := replay.NewRunner()
	return NewHandler(runner)
}

func TestReplayStart_RPSValidation(t *testing.T) {
	h := setupHandler()

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
	h := setupHandler()

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
