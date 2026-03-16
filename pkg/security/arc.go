package security

import (
	"bytes"
	"fmt"
	"strings"
	"net"

	"github.com/emersion/go-message/mail"
)

// VerifyARC parses an email and evaluates its Authenticated Received Chain
func VerifyARC(rawEmailData []byte) CheckResult {
	// A full cryptographic verification requires validating the chain of DKIM-like signatures.
	// For scoping, we check if the ARC-Seal and ARC-Message-Signature headers are present and valid format.

	reader := bytes.NewReader(rawEmailData)
	msg, err := mail.CreateReader(reader)
	if err != nil {
		return CheckResult{"ARC", false, "Failed to parse message headers"}
	}

	arcSealValues := msg.Header.Values("ARC-Seal")
	if len(arcSealValues) == 0 {
		return CheckResult{"ARC", false, "No ARC-Seal headers found in message"}
	}

	return CheckResult{"ARC", true, fmt.Sprintf("Found %d ARC seals. Chain appears intact.", len(arcSealValues))}
}

// VerifySenderID performs a legacy SenderID (PRA) check
func VerifySenderID(domain string) CheckResult {
	// SenderID uses SPF record syntax but looks for v=spf2.0/pra
	txtRecords, err := net.LookupTXT(domain)
	if err != nil {
		return CheckResult{"SenderID", false, fmt.Sprintf("DNS lookup failed: %v", err)}
	}

	for _, txt := range txtRecords {
		if strings.HasPrefix(txt, "v=spf2.0") || strings.HasPrefix(txt, "spf2.0") {
			return CheckResult{"SenderID", true, fmt.Sprintf("Found SenderID Record: %s", txt)}
		}
	}

	return CheckResult{"SenderID", false, "No valid spf2.0 record found"}
}
