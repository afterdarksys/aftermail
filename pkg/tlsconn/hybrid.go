package tlsconn

import (
	"crypto/tls"
	"log"
)

// ConfigurePostQuantum dynamically modifies the standardized `tls.Config`
// pushing `X25519Kyber768Draft00` (CurveID) into the primary preferred parameters.
func ConfigurePostQuantum(baseConfig *tls.Config) *tls.Config {
	log.Printf("[TLS] Enforcing Post-Quantum constraints mapping X25519Kyber768Draft00 negotiations...")
	
	if baseConfig == nil {
		baseConfig = &tls.Config{}
	}

	// Strictly enforce TLS 1.3 exclusively as prior specifications do not safely constrain PQC parameters
	baseConfig.MinVersion = tls.VersionTLS13
	
	// In Go 1.23+, modern Post-Quantum cipher algorithms (CurvePQC) like Kyber drafts 
	// are explicitly enabled via `tls.X25519Kyber768Draft00` natively within `tls.CurvePreferences`.
	// For compilation compatibility pre-1.23, we pass down the direct enumerator constants.
	
	// CurveID: 0x6399 (draft X25519Kyber768)
	const X25519Kyber768Draft00 tls.CurveID = 0x6399
	
	baseConfig.CurvePreferences = []tls.CurveID{
		X25519Kyber768Draft00,
		tls.X25519,
		tls.CurveP256,
	}

	return baseConfig
}
