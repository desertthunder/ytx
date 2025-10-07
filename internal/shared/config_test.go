package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultConfig()

		if config.Database.Path != "./ytx.db" {
			t.Errorf("expected database path ./ytx.db, got %s", config.Database.Path)
		}

		if config.Server.Port != 3000 {
			t.Errorf("expected server port 3000, got %d", config.Server.Port)
		}

		if config.Credentials.YouTube.ProxyURL != "http://127.0.0.1:8080" {
			t.Errorf("expected youtube proxy URL http://127.0.0.1:8080, got %s", config.Credentials.YouTube.ProxyURL)
		}

		if config.Credentials.Spotify.ClientID != "your_spotify_client_id" {
			t.Errorf("expected spotify client_id your_spotify_client_id, got %s", config.Credentials.Spotify.ClientID)
		}
	})

	t.Run("CreateConfigFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		if err := CreateConfigFile(configPath); err != nil {
			t.Fatalf("failed to create config file: %v", err)
		}

		if _, err := os.Stat(configPath); err != nil {
			t.Fatalf("config file should exist: %v", err)
		}

		config, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load created config: %v", err)
		}

		defaultConfig := DefaultConfig()
		if config.Database.Path != defaultConfig.Database.Path {
			t.Errorf("created config database path doesn't match default")
		}

		if err := CreateConfigFile(configPath); err == nil {
			t.Error("creating config file again should fail")
		}
	})

	t.Run("LoadConfig", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		testConfig := `[database]
path = "/custom/path.db"
max_open_conns = 20
max_idle_conns = 10

[server]
host = "0.0.0.0"
port = 8080

[credentials.spotify]
client_id = "test_client_id"
client_secret = "test_secret"
redirect_uri = "http://localhost:3000/callback"

[credentials.youtube]
api_key = "test_api_key"
proxy_url = "http://localhost:9090"
headers_path = "/path/to/headers.json"
`
		if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		config, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		if config.Database.Path != "/custom/path.db" {
			t.Errorf("expected database path /custom/path.db, got %s", config.Database.Path)
		}

		if config.Server.Port != 8080 {
			t.Errorf("expected server port 8080, got %d", config.Server.Port)
		}

		if config.Credentials.Spotify.ClientID != "test_client_id" {
			t.Errorf("expected spotify client_id test_client_id, got %s", config.Credentials.Spotify.ClientID)
		}
	})
}
