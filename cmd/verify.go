package cmd

import (
	"fmt"

	"github.com/ryan/meowmail/pkg/security"
	"github.com/spf13/cobra"
)

var (
	verifyDomain string
)

var verifyCmd = &cobra.Command{
	Use:   "verify [domain]",
	Short: "Run security protocol checks (SPF, DMARC, MTA-STS, BIMI) against a domain",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		domain := args[0]
		fmt.Printf("Starting Security Verification for domain: %s\n\n", domain)

		// Run checks
		spfRes := security.VerifySPF(domain)
		dmarcRes := security.VerifyDMARC(domain)
		stsRes := security.VerifyMTASTS(domain)
		bimiRes := security.VerifyBIMI(domain, "default")

		results := []security.CheckResult{spfRes, dmarcRes, stsRes, bimiRes}

		passed := 0
		for _, res := range results {
			status := "❌ FAIL"
			if res.Passed {
				status = "✅ PASS"
				passed++
			}
			fmt.Printf("%s | [%s] - %s\n", status, res.Protocol, res.Message)
		}

		fmt.Printf("\nCompleted verification: %d passed, %d failed.\n", passed, len(results)-passed)
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}
