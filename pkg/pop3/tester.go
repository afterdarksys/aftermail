package pop3

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/knadh/go-pop3"
)

// GrammarTestConfig holds the configuration for POP3 grammar testing
type GrammarTestConfig struct {
	Host     string
	Port     string
	UseTLS   bool
	Timeout  time.Duration
}

// TestResult represents the outcome of a single grammar test case
type TestResult struct {
	Name    string
	Passed  bool
	Message string
}

// RunGrammarTestSuite runs standard POP3 checks.
func RunGrammarTestSuite(config GrammarTestConfig) []TestResult {
	var results []TestResult
	addr := net.JoinHostPort(config.Host, config.Port)

	// Test 1: Identify if server accepts connection
	tcpConn, err := net.DialTimeout("tcp", addr, config.Timeout)
	if err != nil {
		results = append(results, TestResult{Name: "Connection", Passed: false, Message: err.Error()})
		return results
	}
	tcpConn.Close()
	results = append(results, TestResult{Name: "Connection", Passed: true, Message: "Connected successfully"})

	portInt := 110
	fmt.Sscanf(config.Port, "%d", &portInt)

	opts := pop3.Opt{
		Host:          config.Host,
		Port:          portInt,
		DialTimeout:   config.Timeout,
		TLSEnabled:    config.UseTLS,
		TLSSkipVerify: true, // We are a protocol debugger, skip cert verification
	}

	// Setup POP3 client
	client := pop3.New(opts)
	conn, err := client.NewConn()
	if err != nil {
		results = append(results, TestResult{Name: "POP3 Handshake", Passed: false, Message: err.Error()})
		return results
	}
	defer func() {
		if err := conn.Quit(); err != nil {
			log.Printf("Error quitting POP3: %v\n", err)
		}
	}()
	
	results = append(results, TestResult{Name: "POP3 Handshake", Passed: true, Message: "Connected and negotiated protocol"})

	// STAT relies on Auth usually, it may fail, we just want to check grammar response basically. 
	count, size, err := conn.Stat()
	if err != nil {
		// STAT might fail if not authenticated, but we want to see what error it returns (grammar check)
		results = append(results, TestResult{Name: "POP3 STAT (Unauth)", Passed: false, Message: err.Error()})
	} else {
		results = append(results, TestResult{Name: "POP3 STAT", Passed: true, Message: "STAT successful (msgs: " + fmt.Sprint(count) + ", size: " + fmt.Sprint(size) + ")"})
	}

	return results
}
