package main

import (
	"log"
	"net/http"
	"os"

	"incident-replay/backend/internal/httpapi"
	"incident-replay/backend/internal/replay"
	"incident-replay/backend/internal/scenario"
	"incident-replay/backend/internal/store"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	scenarioSvc := scenario.NewService("data/scenarios")
	runStore := store.NewFileStore("data/runs")
	runner := replay.NewRunner(scenarioSvc, runStore)

	h := httpapi.NewHandler(scenarioSvc, runner, runStore)

	log.Printf("starting backend on %s", addr)
	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
