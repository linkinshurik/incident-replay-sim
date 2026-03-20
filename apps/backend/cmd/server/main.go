package main

import (
	"log"
	"net/http"
	"os"

	"github.com/linkinshurik/incident-replay/internal/httpapi"
	"github.com/linkinshurik/incident-replay/internal/replay"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	runner := replay.NewRunner()
	h := httpapi.NewHandler(runner)

	log.Printf("starting backend on %s", addr)
	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
