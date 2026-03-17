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
		} else if strings.HasPrefix(action, "add_header:") {
			parts := strings.SplitN(strings.TrimPrefix(action, "add_header:"), ":", 2)
			if len(parts) == 2 {
				log.Printf("[Rules] Added header %s: %s to message %s", parts[0], parts[1], originalMsg.Subject)
				// In actual implementation, modify the message headers here
			}
		} else if strings.HasPrefix(action, "divert_to:") {
			email := strings.TrimPrefix(action, "divert_to:")
			log.Printf("[Rules] Diverting message %s to %s", originalMsg.Subject, email)
			// In actual implementation, redirect the message to the specified email
		} else if strings.HasPrefix(action, "screen_to:") {
			email := strings.TrimPrefix(action, "screen_to:")
			log.Printf("[Rules] Screening message %s to %s", originalMsg.Subject, email)
			// In actual implementation, screen/filter the message to the specified email
		} else if strings.HasPrefix(action, "skip_malware_check:") {
			sender := strings.TrimPrefix(action, "skip_malware_check:")
			log.Printf("[Rules] Skipping malware check for sender %s", sender)
			// In actual implementation, bypass malware scanning for this sender
		} else if strings.HasPrefix(action, "skip_spam_check:") {
			sender := strings.TrimPrefix(action, "skip_spam_check:")
			log.Printf("[Rules] Skipping spam check for sender %s", sender)
			// In actual implementation, bypass spam filtering for this sender
		} else if strings.HasPrefix(action, "skip_whitelist_check:") {
			ip := strings.TrimPrefix(action, "skip_whitelist_check:")
			log.Printf("[Rules] Skipping whitelist check for IP %s", ip)
			// In actual implementation, bypass whitelist check for this IP
		} else if strings.HasPrefix(action, "force_second_pass:") {
			mailserver := strings.TrimPrefix(action, "force_second_pass:")
			log.Printf("[Rules] Forcing second pass through mailserver %s", mailserver)
			// In actual implementation, route to another server for additional processing
		} else if strings.HasPrefix(action, "set_dlp:") {
			parts := strings.SplitN(strings.TrimPrefix(action, "set_dlp:"), ":", 2)
			if len(parts) == 2 {
				log.Printf("[Rules] Setting DLP policy: mode=%s, target=%s", parts[0], parts[1])
				// In actual implementation, apply DLP policy
			}
		} else if strings.HasPrefix(action, "skip_dlp:") {
			parts := strings.SplitN(strings.TrimPrefix(action, "skip_dlp:"), ":", 2)
			if len(parts) == 2 {
				log.Printf("[Rules] Skipping DLP check: mode=%s, target=%s", parts[0], parts[1])
				// In actual implementation, bypass DLP for this target
			}
		} else if action == "quarantine" {
			log.Printf("[Rules] Quarantining message %s", originalMsg.Subject)
			// In actual implementation, move message to quarantine folder
		} else if action == "add_to_digest" {
			log.Printf("[Rules] Adding message %s to next digest", originalMsg.Subject)
			// In actual implementation, add to digest queue
		} else if action == "drop" {
			log.Printf("[Rules] Dropping message %s", originalMsg.Subject)
			// In actual implementation, forcefully drop the message
		} else if action == "bounce" {
			log.Printf("[Rules] Bouncing message %s back to sender", originalMsg.Subject)
			// In actual implementation, send bounce notification to sender
		} else if strings.HasPrefix(action, "smtp_error:") {
			code := strings.TrimPrefix(action, "smtp_error:")
			log.Printf("[Rules] Replying with SMTP error code %s for message %s", code, originalMsg.Subject)
			// In actual implementation, send SMTP error response
		} else if strings.HasPrefix(action, "smtp_dsn:") {
			dsn := strings.TrimPrefix(action, "smtp_dsn:")
			log.Printf("[Rules] Replying with SMTP DSN %s for message %s", dsn, originalMsg.Subject)
			// In actual implementation, send SMTP Delivery Status Notification
		} else if strings.HasPrefix(action, "log:") {
			logMsg := strings.TrimPrefix(action, "log:")
			log.Printf("[Rules] Log entry: %s", logMsg)
		} else if strings.HasPrefix(action, "set_filter_rules:") {
			rule := strings.TrimPrefix(action, "set_filter_rules:")
			log.Printf("[Rules] Setting content filter rules: %s", rule)
			// In actual implementation, apply content filter rules
		} else if strings.HasPrefix(action, "rbl_check:") {
			parts := strings.SplitN(strings.TrimPrefix(action, "rbl_check:"), ":", 2)
			if len(parts) == 2 {
				log.Printf("[Rules] RBL check for IP %s against %s", parts[0], parts[1])
				// In actual implementation, perform RBL lookup
			}
		} else if strings.HasPrefix(action, "mx_rbl_check:") {
			parts := strings.SplitN(strings.TrimPrefix(action, "mx_rbl_check:"), ":", 2)
			if len(parts) == 2 {
				log.Printf("[Rules] MX RBL check for domain %s against %s", parts[0], parts[1])
				// In actual implementation, check MX records against RBL
			}
		} else if strings.HasPrefix(action, "domain_resolve:") {
			parts := strings.SplitN(strings.TrimPrefix(action, "domain_resolve:"), ":", 2)
			if len(parts) == 2 {
				log.Printf("[Rules] Domain resolution for %s (verify: %s)", parts[0], parts[1])
				// In actual implementation, resolve and verify domain
			}
		}
	}
	return nil
}
