package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/afterdarksys/aftermail/pkg/security"
	"github.com/spf13/cobra"
)

var (
	emailFile string
)

var inspectCmd = &cobra.Command{
	Use:   "inspect [email_file]",
	Short: "Inspect a raw .eml file for DKIM and ARC signatures",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		emailPath := args[0]
		data, err := ioutil.ReadFile(emailPath)
		if err != nil {
			fmt.Printf("❌ FAIL | Failed to read email file: %v\n", err)
			return
		}

		fmt.Printf("Analyzing Email: %s\n\n", emailPath)

		dkimRes := security.VerifyDKIM(data)
		arcRes := security.VerifyARC(data)

		results := []security.CheckResult{dkimRes, arcRes}

		for _, res := range results {
			status := "❌ FAIL"
			if res.Passed {
				status = "✅ PASS"
			}
			fmt.Printf("%s | [%s] - %s\n", status, res.Protocol, res.Message)
		}
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
