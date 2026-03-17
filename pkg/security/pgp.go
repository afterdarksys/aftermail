package security

import (
	"bytes"
	"io"

	"github.com/ProtonMail/go-crypto/openpgp"
)

// EncryptPGP encrypts a message using a list of public keys
func EncryptPGP(message string, toKeys []*openpgp.Entity) ([]byte, error) {
	var buf bytes.Buffer
	w, err := openpgp.Encrypt(&buf, toKeys, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	_, err = w.Write([]byte(message))
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecryptPGP decrypts a PGP message using a private key keyring
func DecryptPGP(encryptedMessage []byte, keyring openpgp.EntityList) (string, error) {
	md, err := openpgp.ReadMessage(bytes.NewReader(encryptedMessage), keyring, nil, nil)
	if err != nil {
		return "", err
	}
	decrypted, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

// GeneratePGPKey generates a new PGP keypair
func GeneratePGPKey(name, comment, email string) (*openpgp.Entity, error) {
	return openpgp.NewEntity(name, comment, email, nil)
}
