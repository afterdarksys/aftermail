package security

import (
	"fmt"
	"net"
	"strings"
)

// CheckResult holds the status and message from a verification check
type CheckResult struct {
	Protocol string
	Passed   bool
	Message  string
}

// VerifySPF performs a basic DNS check for SPF records on a domain
func VerifySPF(domain string) CheckResult {
	txtRecords, err := net.LookupTXT(domain)
	if err != nil {
		return CheckResult{"SPF", false, fmt.Sprintf("DNS lookup failed: %v", err)}
	}

	for _, txt := range txtRecords {
		if strings.HasPrefix(txt, "v=spf1") {
			return CheckResult{"SPF", true, fmt.Sprintf("Found SPF Record: %s", txt)}
		}
	}

	return CheckResult{"SPF", false, "No valid v=spf1 record found"}
}

// VerifyDMARC performs a basic DNS check for DMARC policies
func VerifyDMARC(domain string) CheckResult {
	dmarcDomain := "_dmarc." + domain
	txtRecords, err := net.LookupTXT(dmarcDomain)
	if err != nil {
		return CheckResult{"DMARC", false, fmt.Sprintf("DNS lookup failed: %v", err)}
	}

	for _, txt := range txtRecords {
		if strings.HasPrefix(txt, "v=DMARC1") {
			return CheckResult{"DMARC", true, fmt.Sprintf("Found DMARC Record: %s", txt)}
		}
	}

	return CheckResult{"DMARC", false, "No valid v=DMARC1 record found"}
}

// VerifyMTASTS performs a DNS check to detect MTA-STS deployment
func VerifyMTASTS(domain string) CheckResult {
	stsDomain := "_mta-sts." + domain
	txtRecords, err := net.LookupTXT(stsDomain)
	if err != nil {
		return CheckResult{"MTA-STS", false, fmt.Sprintf("DNS lookup failed: %v", err)}
	}

	for _, txt := range txtRecords {
		if strings.HasPrefix(txt, "v=STSv1") {
			return CheckResult{"MTA-STS", true, fmt.Sprintf("Found MTA-STS Record: %s", txt)}
		}
	}

	return CheckResult{"MTA-STS", false, "No valid v=STSv1 record found"}
}

// VerifyBIMI performs a DNS check to detect BIMI assertions
func VerifyBIMI(domain string, selector string) CheckResult {
	if selector == "" {
		selector = "default"
	}
	bimiDomain := selector + "._bimi." + domain
	txtRecords, err := net.LookupTXT(bimiDomain)
	if err != nil {
		return CheckResult{"BIMI", false, fmt.Sprintf("DNS lookup failed: %v", err)}
	}

	for _, txt := range txtRecords {
		if strings.HasPrefix(txt, "v=BIMI1") {
			return CheckResult{"BIMI", true, fmt.Sprintf("Found BIMI Record: %s", txt)}
		}
	}

	return CheckResult{"BIMI", false, "No valid v=BIMI1 record found"}
}
