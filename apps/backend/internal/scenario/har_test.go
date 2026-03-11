package scenario

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHARToJSONL(t *testing.T) {
	// Build HAR programmatically to avoid JSON escaping errors (spec: root must be "log")
	har := map[string]interface{}{
		"log": map[string]interface{}{
			"version": "1.2",
			"creator": map[string]interface{}{"name": "Chrome"},
			"entries": []interface{}{
				map[string]interface{}{
					"startedDateTime": "2024-01-15T10:00:00.000Z",
					"request": map[string]interface{}{
						"method":  "GET",
						"url":     "https://api.example.com/api/users",
						"headers": []interface{}{},
					},
				},
				map[string]interface{}{
					"startedDateTime": "2024-01-15T10:00:01.500Z",
					"request": map[string]interface{}{
						"method":  "POST",
						"url":     "https://api.example.com/api/orders?q=1",
						"headers": []interface{}{map[string]interface{}{"name": "Content-Type", "value": "application/json"}},
						"postData": map[string]interface{}{"text": "{\"id\":1}"},
					},
				},
			},
		},
	}
	harBytes, _ := json.Marshal(har)

	jsonl, err := HARToJSONL(harBytes)
	if err != nil {
		t.Fatalf("HARToJSONL: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(jsonl)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], `"path":"/api/users"`) {
		t.Errorf("first line should contain path /api/users: %s", lines[0])
	}
	if !strings.Contains(lines[1], `"path":"/api/orders?q=1"`) {
		t.Errorf("second line should contain path with query: %s", lines[1])
	}
	if !strings.Contains(lines[0], `"ts":"2024-01-15T10:00:00.000Z"`) {
		t.Errorf("first line should contain ts: %s", lines[0])
	}
}

func TestHARToJSONL_Empty(t *testing.T) {
	harBytes := []byte(`{"log":{"version":"1.2","creator":{},"entries":[]}}`)
	_, err := HARToJSONL(harBytes)
	if err == nil {
		t.Fatal("expected error for empty entries")
	}
}
