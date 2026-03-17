package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/afterdarksys/aftermail/internal/daemonapi"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Interact with the local aftermaild service",
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the headless 127.0.0.1:4460 REST/gRPC hybrid daemon",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🚀 Booting AfterMail Background Daemon (aftermaild)...")
		
		// In a real startup sequence, we'd initialize the SQLite DB connection here
		// db := storage.InitDB("aftermail.db")
		server := &daemonapi.Server{
			DB:   nil, // Mocked for daemon CLI isolation
			Port: 4460,
		}

		if err := server.StartServer(); err != nil {
			fmt.Printf("❌ Fatal error starting daemon: %v\n", err)
		}
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if the aftermaild background service is running",
	Run: func(cmd *cobra.Command, args []string) {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://127.0.0.1:4460/api/status")
		if err != nil {
			fmt.Printf("❌ aftermaild is NOT reachable: %v\n", err)
			return
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("✅ aftermaild is online!\n%s\n", string(body))
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
}
