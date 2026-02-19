package scenario

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadScenario_ValidSimple(t *testing.T) {
	tmpDir := "./data/scenarios"
	os.MkdirAll(tmpDir, 0o755)
	filename := filepath.Join(tmpDir, "testscenario.jsonl")
	content := `{"ts":"2021-01-01T00:00:00Z","type":"http","method":"GET","path":"/api/foo","weight":2}
{"ts":"2021-01-01T00:00:01Z","type":"http","method":"POST","path":"/api/bar","weight":3}`
	err := os.WriteFile(filename, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to write scenario file: %v", err)
	}
	defer os.Remove(filename)

	events, err := LoadScenario("testscenario")
	if err != nil {
		t.Fatalf("LoadScenario error: %v", err)
	}

	// Expect expanded by weight: 2 + 3 = 5
	if len(events) != 5 {
		t.Fatalf("expected 5 events expanded by weight, got %d", len(events))
	}

	// Check first 2 events method GET
	for i := 0; i < 2; i++ {
		if events[i].Method != "GET" {
			t.Errorf("expected method GET at index %d, got %s", i, events[i].Method)
		}
	}

	// Check next 3 events method POST
	for i := 2; i < 5; i++ {
		if events[i].Method != "POST" {
			t.Errorf("expected method POST at index %d, got %s", i, events[i].Method)
		}
	}
}

func TestLoadScenario_InvalidJSON(t *testing.T) {
	tmpDir := "./data/scenarios"
	os.MkdirAll(tmpDir, 0o755)
	filename := filepath.Join(tmpDir, "invalid.jsonl")
	content := `invalid json line`
	_ = os.WriteFile(filename, []byte(content), 0o644)
	defer os.Remove(filename)

	_, err := LoadScenario("invalid")
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestLoadScenario_Empty(t *testing.T) {
	tmpDir := "./data/scenarios"
	os.MkdirAll(tmpDir, 0o755)
	filename := filepath.Join(tmpDir, "empty.jsonl")
	content := `


` // empty lines
	_ = os.WriteFile(filename, []byte(content), 0o644)
	defer os.Remove(filename)

	_, err := LoadScenario("empty")
	if err == nil {
		t.Fatal("expected error for empty events")
	}
}

func TestLoadScenario_MissingFile(t *testing.T) {
	_, err := LoadScenario("notexists")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestValidScenarioID(t *testing.T) {
	valids := []string{"abc123", "ABC_def-456"}
	invalids := []string{"abc def", "abc/def", "", "abc$", "abc@"}

	for _, v := range valids {
		if !ValidScenarioID(v) {
			t.Errorf("expected valid but got invalid for %s", v)
		}
	}

	for _, v := range invalids {
		if ValidScenarioID(v) {
			t.Errorf("expected invalid but got valid for %s", v)
		}
	}
}
