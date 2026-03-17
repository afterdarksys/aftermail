package web3mail

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// EncryptedShare defines the structure for sharing a file
type EncryptedShare struct {
	Filename    string
	ContentType string
	Ciphertext  []byte
	Nonce       []byte
	X25519Key   []byte // The symmetric key encrypted to the receiver's X25519 DID public key
}

// ShareFile encrypts an arbitrary file using AES-GCM and wraps the key for the recipient
func ShareFile(filename, contentType string, data []byte, receiverPublicKey []byte) (*EncryptedShare, error) {
	// 1. Generate ephemeral symmetric key
	symKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, symKey); err != nil {
		return nil, err
	}

	// 2. Encrypt the data
	block, err := aes.NewCipher(symKey)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := aesgcm.Seal(nil, nonce, data, nil)

	// 3. Encrypt the symKey using Target's X25519 (STUB)
	// In production, we'd use curve25519 scalar multiplication to derive the shared secret.
	wrappedKey := append([]byte("ENC:"), symKey...)

	return &EncryptedShare{
		Filename:    filename,
		ContentType: contentType,
		Ciphertext:  ciphertext,
		Nonce:       nonce,
		X25519Key:   wrappedKey,
	}, nil
}

// DecryptShare unpacks the payload using our local private X25519 key
func DecryptShare(share *EncryptedShare, localPrivateKey []byte) ([]byte, error) {
	// 1. Decrypt SymKey
	// STUB: Use curve25519 scalar mult to recover shared secret and unwrap
	if len(share.X25519Key) < 4 {
		return nil, fmt.Errorf("invalid wrapped key")
	}
	symKey := share.X25519Key[4:]

	// 2. Decrypt data
	block, err := aes.NewCipher(symKey)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := aesgcm.Open(nil, share.Nonce, share.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("file decryption failed: %w", err)
	}

	return plaintext, nil
}
