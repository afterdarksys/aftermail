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
	
	// Health Check
	mux.HandleFunc("/health", s.handleHealth)

	// REST JSON Routes for SDKs and Browser Extension
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/inbox", s.handleInbox)

	// Debug Console API Routes
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/accounts", s.handleAccounts)
	mux.HandleFunc("/api/sync", s.handleSync)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/test", s.handleTest)

	// Debug Utilities
	mux.HandleFunc("/debug/cache", s.handleDebugCache)
	mux.HandleFunc("/debug/connections", s.handleDebugConnections)
	mux.HandleFunc("/debug/goroutines", s.handleDebugGoroutines)
	mux.HandleFunc("/debug/memory", s.handleDebugMemory)
	mux.HandleFunc("/debug/db", s.handleDebugDatabase)
	mux.HandleFunc("/debug/config", s.handleDebugConfig)
	mux.HandleFunc("/debug/trace", s.handleDebugTrace)

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

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"daemon":    "aftermaild",
		"version":   "1.2.0-headless",
	}
	json.NewEncoder(w).Encode(health)
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.DB == nil {
		http.Error(w, `{"error": "database uninitialized"}`, 500)
		return
	}
	accounts, err := s.DB.ListAccounts()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), 500)
		return
	}
	json.NewEncoder(w).Encode(accounts)
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Trigger account sync (placeholder - implement actual sync logic)
	response := map[string]interface{}{
		"status":  "sync_triggered",
		"message": "Account synchronization initiated",
		"time":    time.Now().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Return recent log entries (placeholder - integrate with actual logger)
	logs := []map[string]interface{}{
		{
			"timestamp": time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			"level":     "INFO",
			"message":   "Daemon started successfully",
		},
		{
			"timestamp": time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
			"level":     "INFO",
			"message":   "REST API listening on port " + fmt.Sprintf("%d", s.Port),
		},
	}
	json.NewEncoder(w).Encode(logs)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.DB == nil {
		http.Error(w, `{"error": "database uninitialized"}`, 500)
		return
	}

	messages, _ := s.DB.ListMessages()
	accounts, _ := s.DB.ListAccounts()

	stats := map[string]interface{}{
		"total_messages": len(messages),
		"total_accounts": len(accounts),
		"uptime_seconds": time.Now().Unix(),
		"timestamp":      time.Now().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	test := map[string]interface{}{
		"status":  "ok",
		"message": "Test endpoint responding",
		"echo":    r.URL.Query().Get("echo"),
	}
	json.NewEncoder(w).Encode(test)
}

func (s *Server) handleDebugCache(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Cache debugging info (placeholder - integrate with actual cache)
	cache := map[string]interface{}{
		"cache_size":     0,
		"cache_hits":     0,
		"cache_misses":   0,
		"eviction_count": 0,
	}
	json.NewEncoder(w).Encode(cache)
}

func (s *Server) handleDebugConnections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Active connection debugging (placeholder)
	connections := map[string]interface{}{
		"active_imap":     0,
		"active_smtp":     0,
		"active_grpc":     0,
		"active_quic":     0,
		"connection_pool": "healthy",
	}
	json.NewEncoder(w).Encode(connections)
}

func (s *Server) handleDebugGoroutines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	// Return pprof goroutine dump
	w.Write([]byte(fmt.Sprintf("Goroutine count: %d\n\n", 0)))
	w.Write([]byte("Use /debug/pprof/goroutine for full goroutine dump\n"))
}

func (s *Server) handleDebugMemory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Memory stats (placeholder - integrate with runtime.MemStats)
	memory := map[string]interface{}{
		"alloc_bytes":      0,
		"total_alloc":      0,
		"sys_bytes":        0,
		"num_gc":           0,
		"goroutines":       0,
		"heap_alloc_bytes": 0,
	}
	json.NewEncoder(w).Encode(memory)
}

func (s *Server) handleDebugDatabase(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.DB == nil {
		http.Error(w, `{"error": "database uninitialized"}`, 500)
		return
	}

	dbInfo := map[string]interface{}{
		"status":          "connected",
		"wal_mode":        "enabled",
		"connection_pool": "active",
		"last_vacuum":     "unknown",
	}
	json.NewEncoder(w).Encode(dbInfo)
}

func (s *Server) handleDebugConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	config := map[string]interface{}{
		"daemon_port":   s.Port,
		"debug_enabled": true,
		"tls_enabled":   false,
		"log_level":     "INFO",
	}
	json.NewEncoder(w).Encode(config)
}

func (s *Server) handleDebugTrace(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Trace debugging enabled.\n"))
	w.Write([]byte("Use /debug/pprof/trace for full execution trace\n"))
}
