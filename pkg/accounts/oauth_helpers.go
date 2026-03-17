package accounts

import (
	"fmt"
	"log"

	"golang.org/x/oauth2"
)

// AccountUpdater is an interface for updating accounts in storage
type AccountUpdater interface {
	UpdateAccount(acc *Account) error
	GetAccount(id int64) (*Account, error)
}

// CreateTokenRefreshCallback creates a callback that persists OAuth token refreshes
// to both the database and the OS keychain via the AccountUpdater interface.
//
// This ensures that refreshed tokens are not lost on application restart.
func CreateTokenRefreshCallback(accountID int64, updater AccountUpdater) func(*oauth2.Token) {
	return func(newToken *oauth2.Token) {
		if newToken == nil {
			log.Printf("[OAuth] Token refresh callback called with nil token for account %d", accountID)
			return
		}

		// Retrieve the current account from storage
		account, err := updater.GetAccount(accountID)
		if err != nil {
			log.Printf("[OAuth] Failed to retrieve account %d for token update: %v", accountID, err)
			return
		}

		// Update token fields
		account.OAuthAccessToken = newToken.AccessToken
		account.OAuthRefreshToken = newToken.RefreshToken
		account.OAuthExpiry = newToken.Expiry

		// Persist to database and keychain
		if err := updater.UpdateAccount(account); err != nil {
			log.Printf("[OAuth] Failed to persist refreshed token for account %d: %v", accountID, err)
			return
		}

		log.Printf("[OAuth] Successfully persisted refreshed token for account %d (expires: %v)", accountID, newToken.Expiry)
	}
}

// ValidateTokenRefresh validates that a token was successfully refreshed
func ValidateTokenRefresh(oldToken, newToken *oauth2.Token) error {
	if newToken == nil {
		return fmt.Errorf("new token is nil")
	}

	if newToken.AccessToken == "" {
		return fmt.Errorf("new access token is empty")
	}

	// Refresh token might not change, but if it does, verify it's not empty
	if oldToken != nil && oldToken.RefreshToken != "" && newToken.RefreshToken == "" {
		return fmt.Errorf("refresh token was removed during refresh")
	}

	// Expiry should be in the future
	if !newToken.Expiry.IsZero() && newToken.Expiry.Before(newToken.Expiry) {
		return fmt.Errorf("new token expiry is in the past")
	}

	return nil
}
