package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ryan/meowmail/pkg/amp"
	"github.com/spf13/cobra"
)

var (
	ampGateway string
	ampDID     string
	ampTarget  string
	ampPayload string
)

var ampCmd = &cobra.Command{
	Use:   "amp",
	Short: "Interact with the AfterSMTP Advanced Message Protocol layer",
}

var ampSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Dispatch an AMP message payload to a target DID",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Initializing AMP Client for DID: %s via %s\n", ampDID, ampGateway)
		
		client := amp.NewClient(ampGateway, ampDID, "mock-priv-key")
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		fmt.Printf("Encrypting and dispatching payload to %s...\n", ampTarget)
		err := client.SendMessage(ctx, ampTarget, []byte(ampPayload))
		if err != nil {
			fmt.Printf("❌ FAIL | AMP Dispatch Error: %v\n", err)
		} else {
			fmt.Println("✅ PASS | Message successfully routed over Substrate/QUIC overlay.")
		}
	},
}

func init() {
	rootCmd.AddCommand(ampCmd)
	ampCmd.AddCommand(ampSendCmd)

	ampSendCmd.Flags().StringVar(&ampGateway, "gateway", "tls://gateway.aftersmtp.local:8443", "AfterSMTP Entry node")
	ampSendCmd.Flags().StringVar(&ampDID, "did", "did:after:ryan", "Sender DID Identity")
	ampSendCmd.Flags().StringVar(&ampTarget, "target", "did:after:recipient", "Target DID Identity")
	ampSendCmd.Flags().StringVar(&ampPayload, "payload", "{}", "JSON AMP Payload")
}
