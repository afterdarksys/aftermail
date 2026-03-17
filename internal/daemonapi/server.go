package daemonapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/afterdarksys/aftermail/pkg/storage"
)

// Server encapsulates the daemon REST/Web dashboard router
type Server struct {
	DB   *storage.DB
	Port int
}

// StartServer launches a background REST/gRPC hybrid bound to localhost
func (s *Server) StartServer() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.Port)
	log.Printf("[DaemonAPI] Binding headless dashboard and REST hooks to %s...", addr)

	mux := http.NewServeMux()

	// Web App Dashboard Route
	mux.HandleFunc("/", s.handleDashboard)
	
	// REST JSON Routes for SDKs and Browser Extension
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/inbox", s.handleInbox)

	// Profiling Endpoints for Performance Analysis
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return httpServer.ListenAndServe()
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	html := `<!DOCTYPE html>
<html>
<head>
	<title>AfterMail Web Dashboard</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; padding: 2rem; background: #0f172a; color: #f8fafc; }
		h1 { color: #38bdf8; }
		.card { background: #1e293b; padding: 1.5rem; border-radius: 8px; border: 1px solid #334155; max-width: 600px; margin-top: 2rem; }
	</style>
</head>
<body>
	<h1>📡 AfterMail Headless Daemon</h1>
	<p>The local daemon is actively routing Web3 AMF payloads and traditional IMAP/SMTP streams.</p>
	<div class="card">
		<h3>System Status: <span style="color: #4ade80;">Active</span></h3>
		<p>Navigate to /api/v1/status for native SDK integration loops.</p>
	</div>
</body>
</html>`
	w.Write([]byte(html))
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := map[string]interface{}{
		"status": "online",
		"uptime": time.Now().Format(time.RFC3339),
		"version": "1.2.0-headless",
	}
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleInbox(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.DB == nil {
		http.Error(w, `{"error": "database uninitialized"}`, 500)
		return
	}
	messages, err := s.DB.ListMessages()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), 500)
		return
	}
	json.NewEncoder(w).Encode(messages)
}
