package daemonapi

import (
	"encoding/json"
	"net/http"
	"github.com/ryan/meowmail/pkg/storage"
)

// Server handles local client requests to the meowmaild daemon cache
type Server struct {
	db *storage.DB
}

// NewServer initializes the local loopback API Handlers
func NewServer(db *storage.DB) *Server {
	return &Server{db: db}
}

// RegisterRoutes binds HTTP paths to handlers
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/folders", s.handleGetFolders)
	mux.HandleFunc("/api/messages", s.handleGetMessages)
}

func (s *Server) handleGetFolders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Mock response fetching folders from DB abstraction
	json.NewEncoder(w).Encode(map[string]interface{}{
		"folders": []string{"Inbox", "Sent", "Trash", "Important"},
	})
}

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Mock response fetching messages for a folder
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"id": 1,
				"sender": "boss@company.com",
				"subject": "Important update",
				"preview": "Please read this...",
				"protocol": "imap",
			},
		},
	})
}
