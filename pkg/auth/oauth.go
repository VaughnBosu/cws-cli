package auth

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const chromeWebStoreScope = "https://www.googleapis.com/auth/chromewebstore"

// Authenticator handles OAuth2 token management.
type Authenticator interface {
	AccessToken(ctx context.Context) (string, error)
}

// OAuthAuthenticator refreshes access tokens using OAuth2 credentials.
type OAuthAuthenticator struct {
	config      *oauth2.Config
	tokenMu     sync.RWMutex
	token       *oauth2.Token
	refreshLock chan struct{}
}

// NewOAuthAuthenticator creates an authenticator from client credentials and a refresh token.
func NewOAuthAuthenticator(clientID, clientSecret, refreshToken string) *OAuthAuthenticator {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{chromeWebStoreScope},
	}

	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	return &OAuthAuthenticator{
		config:      cfg,
		token:       token,
		refreshLock: make(chan struct{}, 1),
	}
}

// AccessToken returns a valid access token, refreshing if necessary.
func (a *OAuthAuthenticator) AccessToken(ctx context.Context) (string, error) {
	if token := a.currentToken(); token.Valid() {
		return token.AccessToken, nil
	}

	select {
	case a.refreshLock <- struct{}{}:
		defer func() { <-a.refreshLock }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	current := a.currentToken()
	if current.Valid() {
		return current.AccessToken, nil
	}

	token, err := a.config.TokenSource(ctx, current).Token()
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w. Run 'cws init --global' to reconfigure credentials", err)
	}

	a.tokenMu.Lock()
	a.token = token
	a.tokenMu.Unlock()
	return token.AccessToken, nil
}

func (a *OAuthAuthenticator) currentToken() *oauth2.Token {
	a.tokenMu.RLock()
	defer a.tokenMu.RUnlock()
	return a.token
}
