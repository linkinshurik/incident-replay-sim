package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	// Channel to listen for signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Channel to wait server shutdown
	done := make(chan struct{})

	go func() {
		<-sigCh
		log.Printf("shutdown started")

		// stop all running replays first so they are marked failed and persisted
		runner.StopAll("server_shutdown")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("server shutdown error: %v", err)
		} else {
			log.Printf("shutdown completed")
		}

		close(done)
	}()

	log.Printf("backend listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}

	// Wait for shutdown to complete
	<-done
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
