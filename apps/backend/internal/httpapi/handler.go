package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"incident-replay/backend/internal/replay"
)

const (
	defaultMaxRPS = 200
)

// StartHandler handles /replay/start POST requests.
func StartHandler(runner *replay.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
			return
		}

		// Validate RPS
		maxRPS := defaultMaxRPS
		if envMaxRPS := os.Getenv("MAX_RPS_PER_RUN"); envMaxRPS != "" {
			if val, err := strconv.Atoi(envMaxRPS); err == nil && val > 0 {
				maxRPS = val
			}
		}
		if req.RPS <= 0 || req.RPS > maxRPS {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_rps"})
			return
		}

		params := replay.StartParams{
			ScenarioID:    req.ScenarioID,
			TargetBaseURL: req.TargetBaseURL,
			RPS:           req.RPS,
			DurationSec:   req.DurationSec,
			Mode:          req.Mode,
			Speed:         req.Speed,
			MaxDelayMs:    req.MaxDelayMs,
			StartFromTs:   req.StartFromTs,
			EndAtTs:       req.EndAtTs,
		}

		runID, err := runner.Start(params)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
			return
		}

		resp := map[string]string{
			"runId":  runID,
			"status": "started",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

// Other handlers omitted for brevity
