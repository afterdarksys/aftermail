package cmd

import (
	"fmt"
	"time"

	"github.com/afterdarksys/aftermail/pkg/imap"
	"github.com/afterdarksys/aftermail/pkg/pop3"
	"github.com/afterdarksys/aftermail/pkg/smtp"
	"github.com/spf13/cobra"
)

var (
	testSmtpHost string
	testSmtpPort string
	testTimeout  time.Duration
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run protocol grammar and compliance tests",
}

var testSmtpCmd = &cobra.Command{
	Use:   "smtp",
	Short: "Run SMTP protocol grammar tests against a server",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Starting SMTP Grammar Test Suite against %s:%s...\n\n", testSmtpHost, testSmtpPort)

		config := smtp.GrammarTestConfig{
			Host:    testSmtpHost,
			Port:    testSmtpPort,
			Timeout: testTimeout,
		}

		results := smtp.RunGrammarTestSuite(config)

		passed := 0
		for _, res := range results {
			status := "❌ FAIL"
			if res.Passed {
				status = "✅ PASS"
				passed++
			}
			fmt.Printf("%s | [%s] - %s\n", status, res.Name, res.Message)
		}
		
		fmt.Printf("\nCompleted %d tests: %d passed, %d failed.\n", len(results), passed, len(results)-passed)
	},
}

var (
	testImapHost   string
	testImapPort   string
	testImapUseTLS bool
)


var testImapCmd = &cobra.Command{
	Use:   "imap",
	Short: "Run IMAP protocol grammar tests against a server",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Starting IMAP Grammar Test Suite against %s:%s (TLS=%v)...\n\n", testImapHost, testImapPort, testImapUseTLS)

		config := imap.GrammarTestConfig{
			Host:    testImapHost,
			Port:    testImapPort,
			UseTLS:  testImapUseTLS,
			Timeout: testTimeout,
		}

		results := imap.RunGrammarTestSuite(config)

		passed := 0
		for _, res := range results {
			status := "❌ FAIL"
			if res.Passed {
				status = "✅ PASS"
				passed++
			}
			fmt.Printf("%s | [%s] - %s\n", status, res.Name, res.Message)
		}
		
		fmt.Printf("\nCompleted %d tests: %d passed, %d failed.\n", len(results), passed, len(results)-passed)
	},
}

var (
	testPop3Host   string
	testPop3Port   string
	testPop3UseTLS bool
)

var testPop3Cmd = &cobra.Command{
	Use:   "pop3",
	Short: "Run POP3 legacy protocol grammar tests against a server",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Starting POP3 Grammar Test Suite against %s:%s (TLS=%v)...\n\n", testPop3Host, testPop3Port, testPop3UseTLS)

		config := pop3.GrammarTestConfig{
			Host:    testPop3Host,
			Port:    testPop3Port,
			UseTLS:  testPop3UseTLS,
			Timeout: testTimeout,
		}

		results := pop3.RunGrammarTestSuite(config)

		passed := 0
		for _, res := range results {
			status := "❌ FAIL"
			if res.Passed {
				status = "✅ PASS"
				passed++
			}
			fmt.Printf("%s | [%s] - %s\n", status, res.Name, res.Message)
		}
		
		fmt.Printf("\nCompleted %d tests: %d passed, %d failed.\n", len(results), passed, len(results)-passed)
	},
}


func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.AddCommand(testSmtpCmd)
	testCmd.AddCommand(testImapCmd)
	testCmd.AddCommand(testPop3Cmd)
	
	testSmtpCmd.Flags().StringVar(&testSmtpHost, "host", "localhost", "Target SMTP Host")
	testSmtpCmd.Flags().StringVar(&testSmtpPort, "port", "25", "Target SMTP Port")
	testSmtpCmd.Flags().DurationVar(&testTimeout, "timeout", 10*time.Second, "Connection timeout")

	testImapCmd.Flags().StringVar(&testImapHost, "host", "localhost", "Target IMAP Host")
	testImapCmd.Flags().StringVar(&testImapPort, "port", "143", "Target IMAP Port")
	testImapCmd.Flags().BoolVar(&testImapUseTLS, "tls", false, "Use TLS (usually port 993)")
	testImapCmd.Flags().DurationVar(&testTimeout, "timeout", 10*time.Second, "Connection timeout")

	testPop3Cmd.Flags().StringVar(&testPop3Host, "host", "localhost", "Target POP3 Host")
	testPop3Cmd.Flags().StringVar(&testPop3Port, "port", "110", "Target POP3 Port")
	testPop3Cmd.Flags().BoolVar(&testPop3UseTLS, "tls", false, "Use TLS (usually port 995)")
	testPop3Cmd.Flags().DurationVar(&testTimeout, "timeout", 10*time.Second, "Connection timeout")
}
