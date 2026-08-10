package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOAuthAuthenticatorCachesRefreshedToken(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "cached-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	defer server.Close()

	authenticator := NewOAuthAuthenticator("client", "secret", "refresh")
	authenticator.config.Endpoint.TokenURL = server.URL

	const callers = 16
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := authenticator.AccessToken(context.Background())
			if err == nil && token != "cached-token" {
				err = errors.New("unexpected access token")
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("token endpoint requests = %d, want 1", got)
	}

	if _, err := authenticator.AccessToken(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("cached call made another token request; requests = %d", got)
	}
}

func TestOAuthAuthenticatorCanceledWhileRefreshInProgress(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	defer server.Close()

	authenticator := NewOAuthAuthenticator("client", "secret", "refresh")
	authenticator.config.Endpoint.TokenURL = server.URL
	firstDone := make(chan error, 1)
	go func() {
		_, err := authenticator.AccessToken(context.Background())
		firstDone <- err
	}()
	<-started

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	secondDone := make(chan error, 1)
	go func() {
		_, err := authenticator.AccessToken(ctx)
		secondDone <- err
	}()
	select {
	case err := <-secondDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("AccessToken error = %v, want context canceled", err)
		}
	case <-time.After(500 * time.Millisecond):
		close(release)
		<-firstDone
		t.Fatal("canceled call waited for another refresh")
	}

	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
}
