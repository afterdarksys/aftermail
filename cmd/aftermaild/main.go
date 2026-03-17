package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/afterdarksys/aftermail/internal/daemonapi"
	"github.com/afterdarksys/aftermail/pkg/accounts"
	"github.com/afterdarksys/aftermail/pkg/rules"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

func main() {
	log.Println("Starting aftermaild - Background Mail Sync Service")

	// Setup local database
	db, err := storage.InitDB("file:aftermaild.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer db.Close()

	apiServer := &daemonapi.Server{
		DB:   db,
		Port: 4460,
	}

	go func() {
		log.Println("Listening for local GUI/CLI APIs on http://127.0.0.1:4460")
		if err := apiServer.StartServer(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("API server error: %v", err)
		}
	}()

	// TODO: Initialize SQLite Database
	// Initialize Starlark Rule Engine
	log.Println("[Rules] Initializing MailScript (Starlark) Engine...")
	// We load a mock rule to ensure the engine boots up successfully
	testRule := "def evaluate():\n    accept()\n"
	err = rules.ExecuteEngine(testRule, &rules.MessageContext{Headers: map[string]string{}})
	if err != nil {
		log.Printf("[Rules] Failed to boot engine: %v\n", err)
	} else {
		log.Println("[Rules] MailScript Engine loaded successfully.")
	}

	// TODO: Start Account Poller routines (IMAP, POP3, AMP)

	// Wait for interrupt
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down aftermaild...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close all gRPC connection pool connections
	if err := accounts.CloseAllConnections(); err != nil {
		log.Printf("Error closing gRPC connections: %v\n", err)
	}

	// NOTE: daemonapi.Server doesn't expose Shutdown currently, so we rely on the process exiting
	log.Printf("Server shutdown signal received via ctx: %v\n", ctx.Err())
}
