package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// loginTimeout bounds how long we wait for the user to complete browser consent.
const loginTimeout = 5 * time.Minute

// AcquireRefreshToken runs the OAuth2 authorization-code flow with a local
// loopback redirect and returns a refresh token. It opens the system browser
// to Google's consent page and waits for the redirect.
//
// The OAuth client must be a "Desktop app" client, which permits loopback
// redirect URIs.
func AcquireRefreshToken(ctx context.Context, clientID, clientSecret string) (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to start local callback server: %w", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURI,
		Scopes:       []string{chromeWebStoreScope},
	}

	state, err := randomHex(16)
	if err != nil {
		return "", err
	}
	verifier := oauth2.GenerateVerifier()

	authURL := cfg.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.S256ChallengeOption(verifier),
	)

	type callbackResult struct {
		code string
		err  error
	}
	resultCh := make(chan callbackResult, 1)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}
			q := r.URL.Query()
			if q.Get("state") != state {
				http.Error(w, "state mismatch", http.StatusBadRequest)
				resultCh <- callbackResult{err: errors.New("OAuth state mismatch; try again")}
				return
			}
			if errMsg := q.Get("error"); errMsg != "" {
				http.Error(w, "authorization failed: "+errMsg, http.StatusBadRequest)
				resultCh <- callbackResult{err: fmt.Errorf("authorization failed: %s", errMsg)}
				return
			}
			fmt.Fprint(w, "<html><body><h2>cws: sign-in complete</h2><p>You can close this window and return to the terminal.</p></body></html>")
			resultCh <- callbackResult{code: q.Get("code")}
		}),
	}
	go func() { _ = server.Serve(ln) }()
	defer server.Close()

	fmt.Println("Opening your browser to complete sign-in...")
	fmt.Println("If it doesn't open automatically, visit:")
	fmt.Println("  " + authURL)
	openBrowser(authURL)

	ctx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()

	var code string
	select {
	case res := <-resultCh:
		if res.err != nil {
			return "", res.err
		}
		code = res.code
	case <-ctx.Done():
		return "", fmt.Errorf("timed out waiting for browser sign-in after %s", loginTimeout)
	}

	token, err := cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return "", fmt.Errorf("failed to exchange authorization code: %w", err)
	}
	if token.RefreshToken == "" {
		return "", errors.New("no refresh token returned. Remove the app's access at https://myaccount.google.com/permissions and try again")
	}

	return token.RefreshToken, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// openBrowser opens the URL in the default browser; failures are non-fatal
// since the URL is printed for manual use.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
