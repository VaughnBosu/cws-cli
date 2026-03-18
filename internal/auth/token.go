package auth

import (
	"context"
	"fmt"
)

// ValidateCredentials attempts a token refresh to verify credentials are valid.
func ValidateCredentials(clientID, clientSecret, refreshToken string) error {
	auth := NewOAuthAuthenticator(clientID, clientSecret, refreshToken)
	_, err := auth.AccessToken(context.Background())
	if err != nil {
		return fmt.Errorf("credential validation failed: %w", err)
	}
	return nil
}
