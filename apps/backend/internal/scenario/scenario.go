package scenario

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Event represents a single scenario event
// Only limited fields are supported per docs/events.md
// weight defaults to 1 if not set

type Event struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	Weight  int               `json:"weight,omitempty"`
}

// weightedPool helps selecting events by weight
// We implement prefix sums for efficient selection if needed in future
// For now we just build expanded slice according to weight for simplicity

type weightedPool struct {
	events []Event
}

// newWeightedPool creates a weightedPool from given events, expanding entries by weight
func newWeightedPool(events []Event) *weightedPool {
	var expanded []Event
	for _, ev := range events {
		w := ev.Weight
		if w <= 0 {
			w = 1
		}
		for i := 0; i < w; i++ {
			expanded = append(expanded, ev)
		}
	}
	return &weightedPool{events: expanded}
}

// Events returns slice of events expanded by weight

func (p *weightedPool) Events() []Event {
	return p.events
}

// LoadScenario loads scenario events from file by scenarioId
// Loads file from ./data/scenarios/<scenarioId>.jsonl
// Parses JSONL lines to Event, supports method,path,headers,body,weight
// Returns expanded events by weight or error

func LoadScenario(scenarioId string) ([]Event, error) {
	const scenariosDir = "./data/scenarios"
	path := filepath.Join(scenariosDir, scenarioId+".jsonl")

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open scenario file: %w", err)
	}
	defer file.Close()

	var events []Event
	rdr := bufio.NewReader(file)

	lineNum := 0
	for {
		line, err := rdr.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("read error: %w", err)
		}
		line = strings.TrimSpace(line)
		if line != "" {
			var ev map[string]interface{}
			if err := json.Unmarshal([]byte(line), &ev); err != nil {
				return nil, fmt.Errorf("json parse error line %d: %w", lineNum+1, err)
			}

			// Validate type field to be "http" (from docs/events.md example)
			typeVal, ok := ev["type"]
			if !ok || typeVal != "http" {
				return nil, fmt.Errorf("invalid or missing event type at line %d", lineNum+1)
			}

			// Parse fields with basic type assertions
			method, _ := ev["method"].(string)
			path, _ := ev["path"].(string)

			// headers is optional map[string]string
			headersMap := make(map[string]string)
			if rawHeaders, ok := ev["headers"].(map[string]interface{}); ok {
				for k, v := range rawHeaders {
					if vs, ok := v.(string); ok {
						headersMap[k] = vs
					}
				}
			}

			// body can be string
			body := ""
			if b, ok := ev["body"].(string); ok {
				body = b
			}

			// weight integer >0 or default 1
			weight := 1
			if w, ok := ev["weight"].(float64); ok {
				weight = int(w)
				if weight < 1 {
					weight = 1
				}
			}

			events = append(events, Event{
				Method:  method,
				Path:    path,
				Headers: headersMap,
				Body:    body,
				Weight:  weight,
			})
		}

		lineNum++
		if errors.Is(err, io.EOF) {
			break
		}
	}

	if len(events) == 0 {
		return nil, errors.New("no valid events found")
	}

	// Build weighted pool to expand by weight
	pool := newWeightedPool(events)
	return pool.Events(), nil
}

// Extra: helper function for safer scenarioId validation (not part of spec but useful for tests)
//
// ValidScenarioID checks if scenarioId matches safe pattern

func ValidScenarioID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}
