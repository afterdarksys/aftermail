package accounts

import (
	"golang.org/x/oauth2"
	"testing"
)

// mockTokenSource simulates a token source that changes its token
type mockTokenSource struct {
	token *oauth2.Token
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return m.token, nil
}

func TestNotifyTokenSource(t *testing.T) {
	initialToken := &oauth2.Token{
		AccessToken:  "old-access-token",
		RefreshToken: "old-refresh-token",
	}

	mockSource := &mockTokenSource{
		token: initialToken,
	}

	var callbackFired bool
	var refreshedToken *oauth2.Token

	callback := func(newToken *oauth2.Token) {
		callbackFired = true
		refreshedToken = newToken
	}

	notifySource := NewNotifyTokenSource(mockSource, initialToken, callback)

	// Call it once with the initial token. Callback should NOT fire.
	tok, err := notifySource.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "old-access-token" {
		t.Fatalf("expected old token")
	}
	if callbackFired {
		t.Fatalf("callback fired unexpectedly on identical token")
	}

	// Change the underlying token
	mockSource.token = &oauth2.Token{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
	}

	// Call it again. Callback SHOULD fire.
	tok, err = notifySource.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "new-access-token" {
		t.Fatalf("expected new token")
	}
	if !callbackFired {
		t.Fatalf("callback did not fire on new token")
	}
	if refreshedToken == nil || refreshedToken.AccessToken != "new-access-token" {
		t.Fatalf("callback ran but passed incorrect token")
	}
}
