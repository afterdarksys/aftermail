package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ryan/meowmail/internal/daemonapi"
	"github.com/ryan/meowmail/pkg/storage"
)

func main() {
	log.Println("Starting meowmaild - Background Mail Sync Service")

	// Setup local database
	db, err := storage.InitDB("file:meowmaild.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer db.Close()

	// Setup local loopback API
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status": "running", "uptime": "ok"}`)
	})

	apiServer := daemonapi.NewServer(db)
	apiServer.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    "127.0.0.1:4460",
		Handler: mux,
	}

	go func() {
		log.Println("Listening for local GUI/CLI APIs on http://127.0.0.1:4460")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("API server error: %v", err)
		}
	}()

	// TODO: Initialize SQLite Database
	// TODO: Initialize Starlark Rule Engine
	// TODO: Start Account Poller routines (IMAP, POP3, AMP)

	// Wait for interrupt
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down meowmaild...")
	
	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v\n", err)
	}
}
