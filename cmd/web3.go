package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/afterdarksys/aftermail/pkg/web3mail"
	"github.com/spf13/cobra"
)

var (
	web3Endpoint string
	web3Wallet   string
)

var web3Cmd = &cobra.Command{
	Use:   "web3",
	Short: "Interact with the Mailblocks.io Proof-of-Stake email layer",
}

var web3GetCmd = &cobra.Command{
	Use:   "quarantined",
	Short: "List staked emails awaiting your review via IPFS",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Fetching Staked Emails for wallet: %s via %s\n\n", web3Wallet, web3Endpoint)
		
		client := web3mail.NewClient(web3Endpoint)
		
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		emails, err := client.GetQuarantined(ctx, web3Wallet)
		if err != nil {
			fmt.Printf("❌ FAIL | Web3 Fetch Error: %v\n", err)
			return
		}

		if len(emails) == 0 {
			fmt.Println("No staked emails found on IPFS for this address.")
			return
		}

		for _, e := range emails {
			fmt.Printf("ID: %s | Sender: %s | Stake: %.2f ETH | IPFS CID: %s\n", e.ID, e.Sender, e.StakeAmount, e.IPFSCID)
		}
	},
}

func init() {
	rootCmd.AddCommand(web3Cmd)
	web3Cmd.AddCommand(web3GetCmd)

	web3GetCmd.Flags().StringVar(&web3Endpoint, "endpoint", "http://localhost:8080", "Mailblocks.io API Node")
	web3GetCmd.Flags().StringVar(&web3Wallet, "wallet", "0x000000000000000", "Your receipt wallet address")
}
