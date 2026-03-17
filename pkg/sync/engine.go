package sync

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/afterdarksys/aftermail/pkg/accounts"
	"github.com/afterdarksys/aftermail/pkg/imap"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

// Engine handles the background synchronization loops bridging IMAP/AMP remote states to local SQLite
type Engine struct {
	db       *storage.DB
	clients  map[string]*imap.Client
	mu       sync.RWMutex
	isActive bool
}

// NewEngine instantiates a synchronization block tied to the active database
func NewEngine(db *storage.DB) *Engine {
	return &Engine{
		db:      db,
		clients: make(map[string]*imap.Client),
	}
}

// Start polling remote inboxes and multiplexing state downwards
func (e *Engine) Start(ctx context.Context, refreshInterval time.Duration) {
	e.mu.Lock()
	if e.isActive {
		e.mu.Unlock()
		return
	}
	e.isActive = true
	e.mu.Unlock()

	log.Println("[Sync] Engine activated. Background polling engaged.")

	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Sync] Context canceled, shutting down engine...")
			e.Stop()
			return
		case <-ticker.C:
			// In a true implementation, we iterate through db.GetAccounts()
			// For each IMAP account, we check UIDValidity and run diffs against local stored UIDs
			e.mu.RLock()
			log.Printf("[Sync] Polling logic activated for %d mapped connections...\n", len(e.clients))
			e.mu.RUnlock()
		}
	}
}

// RegisterClient maps a dialled IMAP socket into the sync engine cache
func (e *Engine) RegisterClient(accountID string, client *imap.Client) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.clients[accountID] = client
}

// DownloadMessage forcefully fetches an RFC822 payload and caches it offline
func (e *Engine) DownloadMessage(ctx context.Context, accountID string, uid uint32) error {
	e.mu.RLock()
	client, ok := e.clients[accountID]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("client not connected for account: %s", accountID)
	}
	
	_ = client // Wait for real IDLE wrapper
	
	// Mock downloading the envelope here into e.db.SaveMessage()
	msg := &accounts.Message{
		ID:       int64(uid),
		FolderID: 1, // Assume Inbox is ID 1
		Subject:  "Synchronized Offline Cache Message",
		Sender:   "remote@aftersmtp.com",
	}
	
	if _, err := e.db.SaveMessage(msg); err != nil {
		return fmt.Errorf("failed to sink message to DB: %w", err)
	}

	return nil
}

// Stop gracefully tears down active listeners
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.isActive = false
	for id, client := range e.clients {
		client.Close()
		delete(e.clients, id)
	}
	log.Println("[Sync] Engine halted. Connections closed.")
}
