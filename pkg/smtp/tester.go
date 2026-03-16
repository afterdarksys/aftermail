package smtp

import (
	"net"
	"net/smtp"
	"time"
)

// GrammarTestConfig holds the configuration for a grammar test run
type GrammarTestConfig struct {
	Host     string
	Port     string
	Timeout  time.Duration
}

// TestResult represents the outcome of a single grammar test case
type TestResult struct {
	Name    string
	Passed  bool
	Message string
}

// RunGrammarTestSuite executes a battery of standard and malformed SMTP commands
func RunGrammarTestSuite(config GrammarTestConfig) []TestResult {
	var results []TestResult
	addr := net.JoinHostPort(config.Host, config.Port)

	// Test 1: Identify if server accepts connection
	conn, err := net.DialTimeout("tcp", addr, config.Timeout)
	if err != nil {
		results = append(results, TestResult{Name: "Connection", Passed: false, Message: err.Error()})
		return results
	}
	conn.Close()
	results = append(results, TestResult{Name: "Connection", Passed: true, Message: "Connected successfully"})

	// Setup SMTP client for proper testing
	client, err := smtp.Dial(addr)
	if err != nil {
		results = append(results, TestResult{Name: "SMTP Handshake", Passed: false, Message: err.Error()})
		return results
	}
	defer client.Quit()

	// Test 2: Standard EHLO
	err = client.Hello("aftermail.test")
	if err != nil {
		results = append(results, TestResult{Name: "EHLO Compliance", Passed: false, Message: err.Error()})
	} else {
		results = append(results, TestResult{Name: "EHLO Compliance", Passed: true, Message: "EHLO accepted"})
	}

	// Note: Fully robust grammar fuzzing would require raw TCP writes to bypass net/smtp safety checks.
	// We will implement manual fuzzing logic later in Phase 6.

	return results
}
