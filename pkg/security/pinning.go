package security

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log"
)

// PinManager handles public key hashing to guarantee TLS handshakes map to trusted backends natively
type PinManager struct {
	PinnedPKP map[string][]string // Map of hosts to a slice of SHA256 base64/hex encoded SPKIs
}

// NewPinManager initializes an empty structure ready for trust injection
func NewPinManager() *PinManager {
	return &PinManager{
		PinnedPKP: make(map[string][]string),
	}
}

// AddPin injects a trusted leaf or intermediate thumbprint locally
func (p *PinManager) AddPin(host, hashHex string) {
	p.PinnedPKP[host] = append(p.PinnedPKP[host], hashHex)
	log.Printf("[Pinning] Bound expected thumbprint %s for host %s", hashHex, host)
}

// VerifyConnection integrates into tls.Config{VerifyConnection} hooks natively
func (p *PinManager) VerifyConnection(host string, certs []*x509.Certificate) error {
	expectedPins, ok := p.PinnedPKP[host]
	if !ok || len(expectedPins) == 0 {
		// If pinning isn't strictly enforced for this host, bypass
		return nil
	}

	if len(certs) == 0 {
		return fmt.Errorf("no peer certificates presented for pinning validation")
	}

	for _, cert := range certs {
		// Calculate SHA256 over the SubjectPublicKeyInfo (SPKI)
		hash := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
		hashStr := hex.EncodeToString(hash[:])

		// Cross-check against active pins
		for _, pin := range expectedPins {
			if hashStr == pin {
				log.Printf("[Pinning] Validation succeeded for %s", host)
				return nil
			}
		}
	}

	return fmt.Errorf("certificate pinning validation failed for %s: no matching public key extracted from the chain", host)
}
