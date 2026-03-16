package cmd

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/afterdarksys/aftermail/pkg/tlsconn"
	"github.com/spf13/cobra"
)

var (
	useTLS bool
	timeout time.Duration
)

var sessionCmd = &cobra.Command{
	Use:   "session [host:port]",
	Short: "Start a raw TLS or TCP session with a remote server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host := args[0]
		
		fmt.Printf("Connecting to %s (TLS: %v)...\n", host, useTLS)
		
		session, err := tlsconn.Connect(host, useTLS, timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer session.Close()
		
		fmt.Println("Connected. Type commands below. Press Ctrl+C to exit.")
		
		// Start a goroutine to read from server and print to stdout
		go func() {
			for {
				line, err := session.ReadLine()
				if err != nil {
					fmt.Fprintf(os.Stderr, "\nConnection closed by remote host: %v\n", err)
					os.Exit(0)
				}
				fmt.Print("< " + line)
			}
		}()
		
		// Read from stdin and send to server
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := scanner.Text()
			if input != "" {
				err := session.WriteLine(input)
				if err != nil {
					fmt.Fprintf(os.Stderr, "\nError writing to server: %v\n", err)
					os.Exit(1)
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(sessionCmd)
	sessionCmd.Flags().BoolVarP(&useTLS, "tls", "t", false, "Connect using TLS")
	sessionCmd.Flags().DurationVarP(&timeout, "timeout", "w", 10*time.Second, "Connection timeout")
}
