package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"incident-replay/backend/internal/httpapi"
	"incident-replay/backend/internal/replay"
)

func main() {
	addr := env("ADDR", ":8080")

	runner := replay.NewRunner()
	h := httpapi.NewHandler(runner)

	srv := &http.Server{
		Addr:              addr,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("backend listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
