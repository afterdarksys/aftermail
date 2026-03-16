package security

import (
	"bytes"
	"fmt"
	"github.com/emersion/go-msgauth/dkim"
)

// VerifyDKIM parses and verifies a raw email file containing DKIM headers
func VerifyDKIM(rawEmailData []byte) CheckResult {
	reader := bytes.NewReader(rawEmailData)
	
	verifications, err := dkim.Verify(reader)
	if err != nil {
		return CheckResult{"DKIM", false, fmt.Sprintf("Failed to parse email or missing DKIM: %v", err)}
	}

	if len(verifications) == 0 {
		return CheckResult{"DKIM", false, "No DKIM signatures found"}
	}

	for _, v := range verifications {
		if v.Err != nil {
			return CheckResult{"DKIM", false, fmt.Sprintf("Domain %s failed validation: %v", v.Domain, v.Err)}
		}
	}

	return CheckResult{"DKIM", true, fmt.Sprintf("Passed validation for %d signatures.", len(verifications))}
}
