package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Interact with the local meowmaild service",
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if the meowmaild background service is running",
	Run: func(cmd *cobra.Command, args []string) {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://127.0.0.1:4460/api/status")
		if err != nil {
			fmt.Printf("❌ meowmaild is NOT reachable: %v\n", err)
			return
		}
		defer resp.Body.Close()
		
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("✅ meowmaild is online!\n%s\n", string(body))
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
}
