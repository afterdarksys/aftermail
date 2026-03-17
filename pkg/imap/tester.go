package imap

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/emersion/go-imap/client"
)

// GrammarTestConfig holds the configuration for IMAP grammar testing
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

// RunGrammarTestSuite runs standard IMAP checks.
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

	// Setup IMAP client
	var c *client.Client
	if config.UseTLS {
		// Use DialTLS to attempt an implicit TLS connection (usually port 993)
		c, err = client.DialTLS(addr, nil) // Insecure skip verify would go here in prod for debug tool
	} else {
		c, err = client.Dial(addr)
	}

	if err != nil {
		results = append(results, TestResult{Name: "IMAP Handshake", Passed: false, Message: err.Error()})
		return results
	}
	defer func() {
		if err := c.Logout(); err != nil {
			log.Printf("Error logging out of IMAP: %v\n", err)
		}
	}()
	
	results = append(results, TestResult{Name: "IMAP Handshake", Passed: true, Message: "Connected and negotiated protocol"})

	// Test 2: CAPABILITY command
	caps, err := c.Capability()
	if err != nil {
		results = append(results, TestResult{Name: "IMAP Capability", Passed: false, Message: err.Error()})
	} else {
		// Just noting we got capabilities
		results = append(results, TestResult{Name: "IMAP Capability", Passed: true, Message: fmt.Sprintf("%d capabilities advertised", len(caps))})
	}

	return results
}
