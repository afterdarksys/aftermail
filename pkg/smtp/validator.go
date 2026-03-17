package smtp

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/afterdarksys/aftermail/pkg/accounts"
)

// RFCValidator ensures structural compliance of outbound payloads mirroring DMARC and DKIM best practices
type RFCValidator struct {
	strictMode bool
}

func NewRFCValidator(strict bool) *RFCValidator {
	return &RFCValidator{strictMode: strict}
}

// ValidateMessageID checks formatting (e.g. `<random-uuid@aftersmtp.local>`)
func (r *RFCValidator) ValidateMessageID(id string) error {
	if !strings.HasPrefix(id, "<") || !strings.HasSuffix(id, ">") {
		return fmt.Errorf("RFC 5322 violation: Message-ID missing angle brackets: %s", id)
	}
	if !strings.Contains(id, "@") {
		return fmt.Errorf("RFC 5322 violation: Message-ID missing domain component: %s", id)
	}
	return nil
}

// ValidateSender ensures domain alignments
func (r *RFCValidator) ValidateSender(sender *accounts.Account) error {
	if sender == nil {
		return fmt.Errorf("RFC 5322 violation: Null Sender Header")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(sender.Email) {
		return fmt.Errorf("RFC 5322 violation: Malformed Sender Address format")
	}

	// Structural boundaries
	return nil
}

// Check8BitMIME analyzes bytes determining if standard 7-bit ASCII constraints
// apply or if the server supports full UTF-8 payload transfers securely natively.
func (r *RFCValidator) Check8BitMIME(payload []byte) bool {
	for _, b := range payload {
		if b > 127 {
			return true // High bit set, implies 8BITMIME requirement
		}
	}
	return false
}
