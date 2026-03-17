package accounts

import (
	"golang.org/x/oauth2"
	"sync"
)

// notifyTokenSource wraps an oauth2.TokenSource to detect when a token is refreshed
// and trigger a callback to persist the new token.
type notifyTokenSource struct {
	base oauth2.TokenSource
	tok  *oauth2.Token
	mu   sync.Mutex
	f    func(*oauth2.Token)
}

// Token returns a token or an error. It caches the old token and checks if the new token
// differs, firing the callback if so.
func (s *notifyTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newToken, err := s.base.Token()
	if err != nil {
		return nil, err
	}

	if s.tok == nil || s.tok.AccessToken != newToken.AccessToken || s.tok.RefreshToken != newToken.RefreshToken {
		s.tok = newToken
		if s.f != nil {
			s.f(newToken)
		}
	}

	return newToken, nil
}

// NewNotifyTokenSource creates a notifyTokenSource wrapped around the provided base TokenSource.
func NewNotifyTokenSource(base oauth2.TokenSource, initial *oauth2.Token, f func(*oauth2.Token)) oauth2.TokenSource {
	return &notifyTokenSource{
		base: base,
		tok:  initial,
		f:    f,
	}
}
