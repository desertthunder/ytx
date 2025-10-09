package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestSpotifyService(t *testing.T) {
	t.Run("NewSpotifyService", func(t *testing.T) {
		t.Run("With Valid Credentials", func(t *testing.T) {
			credentials := map[string]string{
				"client_id":     "test_client_id",
				"client_secret": "test_client_secret",
				"redirect_uri":  "DefaultRedirectURI",
			}

			srv, err := NewSpotifyService(credentials)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if srv == nil {
				t.Fatal("expected service to be created")
			}

			if srv.Name() != "Spotify" {
				t.Errorf("expected service name 'Spotify', got %s", srv.Name())
			}
		})

		t.Run("Missing Client ID", func(t *testing.T) {
			credentials := map[string]string{
				"client_secret": "test_client_secret",
			}

			_, err := NewSpotifyService(credentials)
			if err == nil {
				t.Error("expected error for missing client_id")
			}
		})

		t.Run("Missing Client Secret", func(t *testing.T) {
			credentials := map[string]string{
				"client_id": "test_client_id",
			}

			_, err := NewSpotifyService(credentials)
			if err == nil {
				t.Error("expected error for missing client_secret")
			}
		})

		t.Run("Default Redirect URI", func(t *testing.T) {
			credentials := map[string]string{
				"client_id":     "test_client_id",
				"client_secret": "test_client_secret",
			}

			srv, err := NewSpotifyService(credentials)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if srv.config.RedirectURL != "DefaultRedirectURI" {
				t.Errorf("expected default redirect URI, got %s", srv.config.RedirectURL)
			}
		})
	})

	t.Run("Get AuthURL", func(t *testing.T) {
		credentials := map[string]string{
			"client_id":     "test_client_id",
			"client_secret": "test_client_secret",
		}

		srv, err := NewSpotifyService(credentials)
		if err != nil {
			t.Fatalf("failed to create service: %v", err)
		}

		authURL := srv.GetAuthURL("test_state")
		if authURL == "" {
			t.Error("expected auth URL to be generated")
		}

		if !strings.Contains(authURL, "accounts.spotify.com") {
			t.Error("auth URL should contain Spotify domain")
		}
		if !strings.Contains(authURL, "test_client_id") {
			t.Error("auth URL should contain client_id")
		}
		if !strings.Contains(authURL, "test_state") {
			t.Error("auth URL should contain state")
		}
	})

	t.Run("Authenticate", func(t *testing.T) {
		credentials := map[string]string{
			"client_id":     "test_client_id",
			"client_secret": "test_client_secret",
		}

		srv, err := NewSpotifyService(credentials)
		if err != nil {
			t.Fatalf("failed to create service: %v", err)
		}

		t.Run("WithAccessToken", func(t *testing.T) {
			authCreds := map[string]string{
				"access_token": "test_access_token",
			}

			err := srv.Authenticate(context.Background(), authCreds)
			if err != nil {
				t.Errorf("expected no error with access token, got %v", err)
			}

			if srv.token == nil {
				t.Error("expected token to be set")
			}

			if srv.token.AccessToken != "test_access_token" {
				t.Errorf("expected access token to be 'test_access_token', got %s", srv.token.AccessToken)
			}
		})

		t.Run("Missing Credentials", func(t *testing.T) {
			authCreds := map[string]string{}

			err := srv.Authenticate(context.Background(), authCreds)
			if err == nil {
				t.Error("expected error for missing credentials")
			}
		})
	})

	t.Run("Service Interface", func(t *testing.T) {
		credentials := map[string]string{
			"client_id":     "test_client_id",
			"client_secret": "test_client_secret",
		}

		srv, err := NewSpotifyService(credentials)
		if err != nil {
			t.Fatalf("failed to create service: %v", err)
		}

		var _ Service = srv
	})

	t.Run("SetTokenRefreshCallback", func(t *testing.T) {
		credentials := map[string]string{
			"client_id":     "test_client_id",
			"client_secret": "test_client_secret",
		}

		srv, err := NewSpotifyService(credentials)
		if err != nil {
			t.Fatalf("failed to create service: %v", err)
		}

		t.Run("sets callback successfully", func(t *testing.T) {
			srv.SetTokenRefreshCallback(func(token *oauth2.Token) {
				// Callback set for testing
			})

			if srv.onTokenRefresh == nil {
				t.Error("expected callback to be set")
			}
		})

		t.Run("can set nil callback", func(t *testing.T) {
			srv.SetTokenRefreshCallback(nil)
			if srv.onTokenRefresh != nil {
				t.Error("expected callback to be nil")
			}
		})

		t.Run("callback can be replaced", func(t *testing.T) {
			srv.SetTokenRefreshCallback(func(token *oauth2.Token) {
				// First callback
			})

			srv.SetTokenRefreshCallback(func(token *oauth2.Token) {
				// Second callback
			})

			if srv.onTokenRefresh == nil {
				t.Error("expected callback to be set")
			}
		})
	})

	t.Run("refreshableTokenSource", func(t *testing.T) {
		t.Run("calls callback on first token fetch", func(t *testing.T) {
			callbackCalled := false
			var capturedToken *oauth2.Token

			mockSource := &mockTokenSource{
				token: &oauth2.Token{AccessToken: "test_token"},
			}

			source := &refreshableTokenSource{
				source: mockSource,
				callback: func(token *oauth2.Token) {
					callbackCalled = true
					capturedToken = token
				},
			}

			token, err := source.Token()
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if !callbackCalled {
				t.Error("expected callback to be called on first fetch")
			}
			if capturedToken == nil {
				t.Error("expected token to be captured")
			}
			if capturedToken.AccessToken != "test_token" {
				t.Errorf("expected captured token to be 'test_token', got %s", capturedToken.AccessToken)
			}
			if token.AccessToken != "test_token" {
				t.Errorf("expected returned token to be 'test_token', got %s", token.AccessToken)
			}
		})

		t.Run("calls callback when token changes", func(t *testing.T) {
			callCount := 0
			var capturedTokens []*oauth2.Token

			mockSource := &mockTokenSource{
				token: &oauth2.Token{AccessToken: "token1"},
			}

			source := &refreshableTokenSource{
				source: mockSource,
				callback: func(token *oauth2.Token) {
					callCount++
					capturedTokens = append(capturedTokens, token)
				},
			}

			_, _ = source.Token()
			if callCount != 1 {
				t.Errorf("expected callback called once, got %d", callCount)
			}

			mockSource.token = &oauth2.Token{AccessToken: "token2"}
			token2, _ := source.Token()

			if callCount != 2 {
				t.Errorf("expected callback called twice, got %d", callCount)
			}
			if len(capturedTokens) != 2 {
				t.Errorf("expected 2 captured tokens, got %d", len(capturedTokens))
			}
			if token2.AccessToken != "token2" {
				t.Errorf("expected new token, got %s", token2.AccessToken)
			}
		})

		t.Run("doesn't call callback when token unchanged", func(t *testing.T) {
			callCount := 0

			mockSource := &mockTokenSource{
				token: &oauth2.Token{AccessToken: "same_token"},
			}

			source := &refreshableTokenSource{
				source: mockSource,
				callback: func(token *oauth2.Token) {
					callCount++
				},
			}

			source.Token()
			source.Token()
			source.Token()

			if callCount != 1 {
				t.Errorf("expected callback called once, got %d", callCount)
			}
		})

		t.Run("handles nil callback gracefully", func(t *testing.T) {
			mockSource := &mockTokenSource{
				token: &oauth2.Token{AccessToken: "test_token"},
			}

			source := &refreshableTokenSource{
				source:   mockSource,
				callback: nil,
			}

			token, err := source.Token()
			if err != nil {
				t.Fatalf("expected no error with nil callback, got %v", err)
			}
			if token.AccessToken != "test_token" {
				t.Error("expected token to be returned despite nil callback")
			}
		})

		t.Run("propagates source errors", func(t *testing.T) {
			mockSource := &mockTokenSource{
				err: errors.New("token source error"),
			}

			source := &refreshableTokenSource{
				source: mockSource,
				callback: func(token *oauth2.Token) {
					t.Error("callback should not be called on error")
				},
			}

			token, err := source.Token()
			if err == nil {
				t.Fatal("expected error from source")
			}
			if !strings.Contains(err.Error(), "token source error") {
				t.Errorf("expected source error, got %v", err)
			}
			if token != nil {
				t.Error("expected nil token on error")
			}
		})

		t.Run("handles callback panic gracefully", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Error("expected panic to be contained within callback")
				}
			}()

			mockSource := &mockTokenSource{
				token: &oauth2.Token{AccessToken: "test_token"},
			}

			source := &refreshableTokenSource{
				source: mockSource,
				callback: func(token *oauth2.Token) {
					panic("callback panic")
				},
			}

			func() {
				defer func() {
					_ = recover()
				}()
				source.Token()
			}()
		})
	})
}

// mockTokenSource implements [oauth2.TokenSource] for testing
type mockTokenSource struct {
	token *oauth2.Token
	err   error
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return m.token, m.err
}
