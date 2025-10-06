package shared

import "testing"

func TestConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultConfig()

		if config.Database.Path != "./tmp/ytx.db" {
			t.Errorf("expected database path ./tmp/ytx.db, got %s", config.Database.Path)
		}

		if config.Server.Port != 3000 {
			t.Errorf("expected server port 3000, got %d", config.Server.Port)
		}

		if config.Credentials.YouTube.ProxyURL != "http://localhost:8080" {
			t.Errorf("expected youtube proxy URL http://localhost:8080, got %s", config.Credentials.YouTube.ProxyURL)
		}

		if config.Credentials.Spotify.ClientID != "your_spotify_client_id" {
			t.Errorf("expected spotify client_id your_spotify_client_id, got %s", config.Credentials.Spotify.ClientID)
		}
	})
}
