package replay

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

func TestStartValidation(t *testing.T) {
	r := NewRunner()

	if _, err := r.Start(StartParams{TargetBaseURL: "", RPS: 1, Duration: time.Second, ScenarioID: "valid"}); err == nil {
		t.Fatalf("expected error for empty target")
	}
	if _, err := r.Start(StartParams{TargetBaseURL: "http://x", RPS: 0, Duration: time.Second, ScenarioID: "valid"}); err == nil {
		t.Fatalf("expected error for rps")
	}
	if _, err := r.Start(StartParams{TargetBaseURL: "http://x", RPS: 1, Duration: 0, ScenarioID: "valid"}); err == nil {
		t.Fatalf("expected error for duration")
	}
	if _, err := r.Start(StartParams{TargetBaseURL: "http://x", RPS: 1, Duration: time.Second, ScenarioID: ""}); err == nil {
		t.Fatalf("expected error for empty scenarioId")
	}
}

func TestStartStopStatus(t *testing.T) {
	r := NewRunner()

	// Prepare a simple scenario file
	const scenarioID = "testrun"
	data := `{"type":"http","method":"GET","path":"/test","weight":1}` + "\n"
	// Create scenario file
	f, err := writeScenarioFile(t, scenarioID, data)
	defer func() {
		_ = f.Close()
		_ = removeScenarioFile(t, scenarioID)
	}()

	if err != nil {
		t.Fatalf("error creating scenario file: %v", err)
	}

	id, err := r.Start(StartParams{TargetBaseURL: "http://127.0.0.1", RPS: 1, Duration: time.Second, ScenarioID: scenarioID})
	if err != nil {
		t.Fatalf("start err: %v", err)
	}

	st, err := r.Status(id)
	if err != nil || st.State != StateRunning {
		t.Fatalf("status expected running, got %+v err=%v", st, err)
	}

	if err := r.Stop(id); err != nil {
		t.Fatalf("stop err: %v", err)
	}

	st, _ = r.Status(id)
	if st.State != StateStopped {
		t.Fatalf("expected stopped, got %s", st.State)
	}
}

func writeScenarioFile(t *testing.T, scenarioID, content string) (*os.File, error) {
	fpath := "./data/scenarios/" + scenarioID + ".jsonl"
	err := os.MkdirAll("./data/scenarios", 0o755)
	if err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	f, err := os.Create(fpath)
	if err != nil {
		t.Fatalf("create file failed: %v", err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	return f, err
}

func removeScenarioFile(t *testing.T, scenarioID string) error {
	fpath := "./data/scenarios/" + scenarioID + ".jsonl"
	return os.Remove(fpath)
}

func TestStatsAddSampleConcurrent(t *testing.T) {
	run := NewRun("test-run", nil)

	wg := sync.WaitGroup{}
	n := 1000

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			latencyMs := int64(i % 100)
			isError := i%10 == 0
			run.AddSample(latencyMs, isError)
		}(i)
	}
	wg.Wait()

	if run.Stats.Requests != int64(n) {
		t.Fatalf("expected %d requests, got %d", n, run.Stats.Requests)
	}
	expectedErrors := int64(n / 10)
	if run.Stats.Errors != expectedErrors {
		t.Fatalf("expected %d errors, got %d", expectedErrors, run.Stats.Errors)
	}

	// Check latency samples count
	if len(run.Stats.LatencySamples) != n-int(expectedErrors) {
		t.Fatalf("expected %d latency samples, got %d", n-int(expectedErrors), len(run.Stats.LatencySamples))
	}
}

func TestCalculateP95(t *testing.T) {

	stats := &Stats{}

	// Add 100 samples 1..100 ms
	for i := int64(1); i <= 100; i++ {
		stats.AddSample(i, false)
	}

	stats.CalculateP95()

	if stats.p95ms != 95 {
		t.Fatalf("expected p95 95, got %d", stats.p95ms)
	}

	// Add some errors, should not affect latency samples
	for i := 0; i < 10; i++ {
		stats.AddSample(0, true)
	}

	stats.CalculateP95()

	if stats.p95ms != 95 {
		t.Fatalf("expected p95 95 after errors, got %d", stats.p95ms)
	}
}

func TestReplayRunnerWithHttpRequests(t *testing.T) {
	// Setup an httptest server to verify requests
	hits := make(map[string]int)
	mu := sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits["method:"+r.Method+" path:"+r.URL.Path]++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	// Prepare a scenario file with multiple events different methods and paths
	const scenarioID = "httptestrun"
	content := `{"type":"http","method":"GET","path":"/getpath","weight":3}` + "\n" +
		`{"type":"http","method":"POST","path":"/postpath","weight":2}` + "\n"
	f, err := writeScenarioFile(t, scenarioID, content)
	if err != nil {
		t.Fatalf("failed to write scenario file: %v", err)
	}
	defer func() {
		_ = f.Close()
		_ = removeScenarioFile(t, scenarioID)
	}()

	r := NewRunner()

	rps := 10
	dur := 2 * time.Second

	runID, err := r.Start(StartParams{
		ScenarioID:    scenarioID,
		TargetBaseURL: server.URL,
		RPS:           rps,
		Duration:      dur,
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for the run to finish
	time.Sleep(dur + time.Second)

	st, err := r.Status(runID)
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}

	if st.State != StateStopped {
		t.Errorf("expected run state stopped, got %s", st.State)
	}

	// Check if hits map has expected methods and paths
	mu.Lock()
	defer mu.Unlock()

	if hits["method:GET path:/getpath"] == 0 {
		t.Errorf("expected hits on GET /getpath")
	}
	if hits["method:POST path:/postpath"] == 0 {
		t.Errorf("expected hits on POST /postpath")
	}

	// Check that number of requests roughly matches RPS and duration
	totalHits := 0
	for _, v := range hits {
		totalHits += v
	}
	if totalHits < int(int64(rps)*int64(dur.Seconds())/2) { // expect at least half of planned requests
		t.Errorf("expected total hits at least %d, got %d", int(int64(rps)*int64(dur.Seconds())/2), totalHits)
	}
}
