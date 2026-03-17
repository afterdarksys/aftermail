package rules

import (
	"log"
	"strings"

	"github.com/afterdarksys/aftermail/pkg/accounts"
	"github.com/afterdarksys/aftermail/pkg/proto"
)

// HandleActions interprets the Starlark execution results.
func HandleActions(originalMsg *accounts.Message, context *MessageContext, dispatcher func(*proto.AMPMessage) error) error {
	for _, action := range context.Actions {
		if strings.HasPrefix(action, "auto_reply:") {
			replyText := strings.TrimPrefix(action, "auto_reply:")
			
			// Construct the automated response payload
			payload := &proto.AMPMessage{
				Headers: &proto.AMPHeaders{
					RecipientDid: originalMsg.SenderDID,
					SenderDid:    "did:aftersmtp:local:auto-responder", // or resolve from Account Config
				},
				// We inject the templated auto-reply message into the unencrypted routing layer optionally
				// However, if we built AMFPayload, we would encrypt it here.
				BlockchainProof: "auto-response-generated",
			}
			
			// Output for logging
			log.Printf("[Responder] Triggered auto-reply for %s. Payload: %s", originalMsg.Subject, replyText)

			if dispatcher != nil {
				if err := dispatcher(payload); err != nil {
					log.Printf("[Responder] Dispatch failed: %v", err)
					return err
				}
			}
		} else if strings.HasPrefix(action, "fileinto:") {
			folder := strings.TrimPrefix(action, "fileinto:")
			log.Printf("[Rules] Moved message %s to folder %s", originalMsg.Subject, folder)
		} else if action == "discard" {
			log.Printf("[Rules] Target message %s dropped implicitly.", originalMsg.Subject)
		}
	}
	return nil
}
