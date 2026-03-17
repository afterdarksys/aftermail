package accounts

import (
	"fmt"
	"log"

	"github.com/zalando/go-keyring"
)

const keychainService = "aftermail"

// SecureStorage defines an interface for storing and retrieving secrets
type SecureStorage interface {
	StoreAccountSecret(accountID string, key string, data string) error
	RetrieveAccountSecret(accountID string, key string) (string, error)
	DeleteAccountSecret(accountID string, key string) error
}

// DefaultSecureStorage is the production implementation using OS keychain
type DefaultSecureStorage struct{}

// NewDefaultSecureStorage creates a new keychain wrapper
func NewDefaultSecureStorage() *DefaultSecureStorage {
	return &DefaultSecureStorage{}
}

// formatKeyringUser creates a unique composite key for the OS keychain
func formatKeyringUser(accountID string, key string) string {
	return fmt.Sprintf("%s:%s", accountID, key)
}

// StoreAccountSecret saves sensitive data to the OS keychain
func (s *DefaultSecureStorage) StoreAccountSecret(accountID string, key string, data string) error {
	if data == "" {
		return nil // Don't store empty secrets
	}
	user := formatKeyringUser(accountID, key)
	err := keyring.Set(keychainService, user, data)
	if err != nil {
		log.Printf("Warning: Failed to store %s to OS keychain for account %s: %v", key, accountID, err)
		return err
	}
	return nil
}

// RetrieveAccountSecret fetches sensitive data from the OS keychain
func (s *DefaultSecureStorage) RetrieveAccountSecret(accountID string, key string) (string, error) {
	user := formatKeyringUser(accountID, key)
	data, err := keyring.Get(keychainService, user)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", nil // Return empty without error if not found, preserving backwards compatibility tests
		}
		log.Printf("Warning: Failed to retrieve %s from OS keychain for account %s: %v", key, accountID, err)
		return "", err
	}
	return data, nil
}

// DeleteAccountSecret deletes a secret from the OS keychain
func (s *DefaultSecureStorage) DeleteAccountSecret(accountID string, key string) error {
	user := formatKeyringUser(accountID, key)
	err := keyring.Delete(keychainService, user)
	if err != nil && err != keyring.ErrNotFound {
		log.Printf("Warning: Failed to delete %s from OS keychain for account %s: %v", key, accountID, err)
		return err
	}
	return nil
}
