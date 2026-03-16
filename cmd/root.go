package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ryan/meowmail/internal/gui"
)

var rootCmd = &cobra.Command{
	Use:   "meowmail",
	Short: "Meowmail: The making e-mail purr again protocol debugger.",
	Long: `Meowmail is a comprehensive e-mail protocol testing, 
interactive debugging, and security verification tool. 
Supports SMTP, POP3, IMAP, DKIM, DMARC, SPF verification, and more.`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no arguments/flags are provided (default run), launch Fyne GUI.
		gui.StartGUI()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
