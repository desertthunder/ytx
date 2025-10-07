package services

import (
	"context"
	"strings"
	"testing"
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
}
