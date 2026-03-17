package security

import (
	"crypto/x509"

	"go.mozilla.org/pkcs7"
)

// EncryptSMIME encrypts data using a recipient's X.509 certificate for S/MIME compatibility
func EncryptSMIME(data []byte, recipientCert *x509.Certificate) ([]byte, error) {
	return pkcs7.Encrypt(data, []*x509.Certificate{recipientCert})
}

// ParseCertificate parses a DER-encoded X.509 certificate for the recipient
func ParseCertificate(derBytes []byte) (*x509.Certificate, error) {
	return x509.ParseCertificate(derBytes)
}
