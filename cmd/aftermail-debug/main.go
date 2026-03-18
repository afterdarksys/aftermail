package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/chzyer/readline"
)

const (
	daemonURL = "http://127.0.0.1:4460"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	if len(os.Args) > 1 {
		// Command mode
		handleCommand(os.Args[1:])
		return
	}

	// Interactive REPL mode
	startREPL()
}

func handleCommand(args []string) {
	cmd := args[0]

	switch cmd {
	case "shell", "repl", "debug":
		startREPL()
	case "ping":
		pingDaemon()
	case "status":
		getDaemonStatus()
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printHelp()
	}
}

func startREPL() {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[35maftermail-debug>\033[0m ",
		HistoryFile:     os.ExpandEnv("$HOME/.aftermail_debug_history"),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    completer,
	})
	if err != nil {
		// Fallback to basic input if readline fails
		startBasicREPL()
		return
	}
	defer rl.Close()

	printBanner()
	fmt.Println("Type 'help' for available commands, 'exit' to quit")
	fmt.Println()

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if line == "exit" || line == "quit" {
			fmt.Println("Goodbye! 🐱")
			break
		}

		executeREPLCommand(line)
	}
}

var completer = readline.NewPrefixCompleter(
	readline.PcItem("help"),
	readline.PcItem("ping"),
	readline.PcItem("status"),
	readline.PcItem("accounts"),
	readline.PcItem("sync"),
	readline.PcItem("debug",
		readline.PcItem("cache"),
		readline.PcItem("connections"),
		readline.PcItem("goroutines"),
		readline.PcItem("memory"),
		readline.PcItem("db"),
		readline.PcItem("config"),
		readline.PcItem("env"),
	),
	readline.PcItem("logs"),
	readline.PcItem("stats"),
	readline.PcItem("clear"),
	readline.PcItem("version"),
	readline.PcItem("exit"),
)

func startBasicREPL() {
	scanner := bufio.NewScanner(os.Stdin)

	printBanner()
	fmt.Println("Type 'help' for available commands, 'exit' to quit")
	fmt.Println()

	for {
		fmt.Print("\033[35maftermail-debug>\033[0m ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if line == "exit" || line == "quit" {
			fmt.Println("Goodbye! 🐱")
			break
		}

		executeREPLCommand(line)
	}
}

func executeREPLCommand(line string) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "help", "?":
		printREPLHelp()

	case "ping":
		pingDaemon()

	case "status", "info":
		getDaemonStatus()

	case "accounts":
		listAccounts()

	case "sync":
		if len(args) > 0 {
			syncAccount(args[0])
		} else {
			fmt.Println("Usage: sync <account-id>")
		}

	case "debug":
		if len(args) > 0 {
			debugCommand(args)
		} else {
			printDebugHelp()
		}

	case "logs":
		count := "50"
		if len(args) > 0 {
			count = args[0]
		}
		showLogs(count)

	case "stats":
		showStats()

	case "test":
		if len(args) > 0 {
			runTest(args[0])
		} else {
			listTests()
		}

	case "monitor":
		startMonitor()

	case "trace":
		if len(args) > 0 {
			enableTrace(args[0])
		} else {
			fmt.Println("Usage: trace <component>")
		}

	case "clear", "cls":
		fmt.Print("\033[H\033[2J")

	case "version":
		fmt.Printf("aftermail-debug %s (commit: %s)\n", version, commit)
		fmt.Printf("Go: %s\n", runtime.Version())

	default:
		fmt.Printf("Unknown command: %s (type 'help' for commands)\n", cmd)
	}
}

func printBanner() {
	banner := `
    ___       ___      ____             ______                      __
   /   |  ___/ / /__  / __ \__  _____  / ____/___  ____  _________  / ___
  / /| | / __  / _ \/ / / / / / / __ \/ /   / __ \/ __ \/ ___/ __ \/ / _ \
 / ___ |/ /_/ /  __/ /_/ / /_/ / /_/ / /___/ /_/ / / / (__  ) /_/ / /  __/
/_/  |_|\__,_/\___/_____/\__,_/\__, /\____/\____/_/ /_/____/\____/_/\___/
                              /____/

    🐱 AfterMail Debug Console - Daemon Debugging & Diagnostics
`
	fmt.Println(banner)
}

func printHelp() {
	help := `AfterMail Debug Console - Daemon Interaction & Debugging Tool

USAGE:
    aftermail-debug [command]

COMMANDS:
    shell, repl, debug   Start interactive debug console (default)
    ping                 Check if daemon is running
    status               Get daemon status
    help                 Show this help message

INTERACTIVE MODE:
    Run 'aftermail-debug' without arguments to enter interactive shell
    The shell provides debugging and diagnostic capabilities

EXAMPLES:
    aftermail-debug              # Start interactive console
    aftermail-debug ping         # Check daemon health
`
	fmt.Println(help)
}

func printREPLHelp() {
	help := `
🔧 AFTERMAIL DEBUG CONSOLE COMMANDS

Daemon Control:
  ping                    - Check daemon connectivity and response time
  status, info            - Get daemon status, uptime, and health
  logs [n]                - Show last n log entries (default: 50)
  stats                   - Show performance statistics
  monitor                 - Start real-time monitoring dashboard

Debug Utilities:
  debug cache             - Show cache statistics and hit rates
  debug connections       - List active gRPC/IMAP/SMTP connections
  debug goroutines        - Show goroutine count and stack traces
  debug memory            - Display detailed memory usage
  debug db                - Database diagnostics and table statistics
  debug config            - Show current configuration
  debug env               - Show environment variables

Account Operations:
  accounts                - List all configured accounts with details
  sync <account-id>       - Trigger manual sync for specific account

Testing:
  test list               - List available integration tests
  test <name>             - Run specific test
  trace <component>       - Enable tracing for component (imap, smtp, grpc)

Utilities:
  clear, cls              - Clear screen
  version                 - Show version and build info
  help, ?                 - Show this help
  exit, quit              - Exit console

TIP: Use Tab completion for commands and arguments!
`
	fmt.Println(help)
}

func printDebugHelp() {
	help := `
🔍 DEBUG UTILITIES:

  debug cache          - Cache statistics, hit rates, memory usage
  debug connections    - Active connections (gRPC, IMAP, SMTP, WebSocket)
  debug goroutines     - Goroutine count, stack traces, leak detection
  debug memory         - Heap, stack, GC stats, memory breakdown
  debug db             - Database size, table counts, query performance
  debug config         - Full configuration dump (sanitized)
  debug env            - Environment variables affecting daemon
`
	fmt.Println(help)
}

func pingDaemon() {
	fmt.Print("📡 Pinging daemon... ")

	resp, err := http.Get(daemonURL + "/health")
	if err != nil {
		fmt.Printf("\n❌ Daemon not responding: %v\n", err)
		fmt.Println("💡 Is aftermaild running? Start it with: aftermaild")
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✅ Online!\n")
	fmt.Printf("Response: %s\n", string(body))
}

func getDaemonStatus() {
	resp, err := http.Get(daemonURL + "/api/status")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var status map[string]interface{}
	if err := json.Unmarshal(body, &status); err != nil {
		fmt.Printf("Response: %s\n", string(body))
		return
	}

	fmt.Println("\n📊 Daemon Status:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for key, val := range status {
		fmt.Printf("  %-20s: %v\n", key, val)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

func listAccounts() {
	resp, err := http.Get(daemonURL + "/api/accounts")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var accounts []map[string]interface{}
	if err := json.Unmarshal(body, &accounts); err != nil {
		fmt.Printf("Response: %s\n", string(body))
		return
	}

	fmt.Println("\n📧 Configured Accounts:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for i, acc := range accounts {
		enabled := "✓"
		if !acc["enabled"].(bool) {
			enabled = "✗"
		}
		fmt.Printf("  [%d] %s %v (%v)\n", i+1, enabled, acc["name"], acc["type"])
		fmt.Printf("      📮 Email: %v\n", acc["email"])
		if lastSync, ok := acc["last_synced_at"]; ok && lastSync != nil {
			fmt.Printf("      🔄 Last Sync: %v\n", lastSync)
		}
		fmt.Println()
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

func syncAccount(accountID string) {
	fmt.Printf("🔄 Triggering sync for account %s...\n", accountID)

	data := map[string]string{"account_id": accountID}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(daemonURL+"/api/sync", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✅ Response: %s\n", string(body))
}

func debugCommand(args []string) {
	subcommand := args[0]

	switch subcommand {
	case "help":
		printDebugHelp()
	case "cache":
		debugCache()
	case "connections", "conn":
		debugConnections()
	case "goroutines", "go":
		debugGoroutines()
	case "memory", "mem":
		debugMemory()
	case "db", "database":
		debugDatabase()
	case "config", "cfg":
		debugConfig()
	case "env":
		debugEnv()
	default:
		fmt.Printf("Unknown debug subcommand: %s\n", subcommand)
		printDebugHelp()
	}
}

func debugCache() {
	resp, err := http.Get(daemonURL + "/debug/cache")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("\n📦 Cache Statistics:\n%s\n", string(body))
}

func debugConnections() {
	resp, err := http.Get(daemonURL + "/debug/connections")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("\n🔌 Active Connections:\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n%s\n", string(body))
}

func debugGoroutines() {
	resp, err := http.Get(daemonURL + "/debug/goroutines")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("\n🔄 Goroutines:\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n%s\n", string(body))
}

func debugMemory() {
	resp, err := http.Get(daemonURL + "/debug/memory")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("\n💾 Memory Usage:\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n%s\n", string(body))
}

func debugDatabase() {
	resp, err := http.Get(daemonURL + "/debug/db")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("\n🗄️  Database Diagnostics:\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n%s\n", string(body))
}

func debugConfig() {
	resp, err := http.Get(daemonURL + "/debug/config")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("\n⚙️  Configuration:\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n%s\n", string(body))
}

func debugEnv() {
	fmt.Println("\n🌍 Environment Variables:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	count := 0
	for _, env := range os.Environ() {
		if strings.Contains(strings.ToUpper(env), "AFTERMAIL") ||
		   strings.Contains(strings.ToUpper(env), "MAIL") ||
		   strings.Contains(env, "SMTP") ||
		   strings.Contains(env, "IMAP") {
			fmt.Printf("  %s\n", env)
			count++
		}
	}
	if count == 0 {
		fmt.Println("  (no mail-related environment variables found)")
	}
	fmt.Println()
}

func showLogs(count string) {
	resp, err := http.Get(fmt.Sprintf("%s/api/logs?count=%s", daemonURL, count))
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("\n📜 Recent Logs (last %s):\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n%s\n", count, string(body))
}

func showStats() {
	resp, err := http.Get(daemonURL + "/api/stats")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("\n📈 Performance Statistics:\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n%s\n", string(body))
}

func listTests() {
	fmt.Println("\n🧪 Available Integration Tests:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	tests := []string{
		"imap-connect    - Test IMAP connection and authentication",
		"smtp-send       - Test SMTP message sending",
		"oauth-refresh   - Test OAuth token refresh flow",
		"grpc-gateway    - Test gRPC gateway connection",
		"db-integrity    - Test database integrity and constraints",
		"cache-perf      - Test cache performance and hit rates",
	}
	for _, test := range tests {
		fmt.Printf("  • %s\n", test)
	}
	fmt.Println("\nUsage: test <name>")
	fmt.Println()
}

func runTest(testName string) {
	fmt.Printf("🧪 Running test: %s...\n", testName)

	data := map[string]string{"test": testName}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(daemonURL+"/api/test", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("%s\n", string(body))
}

func startMonitor() {
	fmt.Println("🔴 Real-time Monitoring (Ctrl+C to stop)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("(Monitoring dashboard not yet implemented)")
	fmt.Println("Will show: goroutines, memory, connections, sync status")
}

func enableTrace(component string) {
	fmt.Printf("🔍 Enabling trace for: %s\n", component)

	data := map[string]string{"component": component, "enable": "true"}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(daemonURL+"/debug/trace", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✅ %s\n", string(body))
}
