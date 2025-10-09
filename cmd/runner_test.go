package main

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/desertthunder/ytx/internal/services"
	"github.com/desertthunder/ytx/internal/shared"
	tu "github.com/desertthunder/ytx/internal/testing"
	"golang.org/x/oauth2"
)

func TestRunner(t *testing.T) {
	t.Run("NewRunner", func(t *testing.T) {
		t.Run("with all dependencies provided", func(t *testing.T) {
			config := shared.DefaultConfig()
			logger := shared.NewLogger(nil)
			output := &bytes.Buffer{}
			httpClient := &http.Client{}
			spotify := &tu.MockService{}
			youtube := &tu.MockService{}
			api := &services.APIService{}

			runner := NewRunner(RunnerOpts{
				Config:     config,
				Logger:     logger,
				Output:     output,
				HTTPClient: httpClient,
				Spotify:    spotify,
				YouTube:    youtube,
				API:        api,
			})

			if runner.config != config {
				t.Error("expected config to be set")
			}
			if runner.logger != logger {
				t.Error("expected logger to be set")
			}
			if runner.output != output {
				t.Error("expected output to be set")
			}
			if runner.httpClient != httpClient {
				t.Error("expected httpClient to be set")
			}
			if runner.spotify != spotify {
				t.Error("expected spotify to be set")
			}
			if runner.youtube != youtube {
				t.Error("expected youtube to be set")
			}
			if runner.api != api {
				t.Error("expected api to be set")
			}
		})

		t.Run("with nil config uses defaults", func(t *testing.T) {
			runner := NewRunner(RunnerOpts{
				Config: nil,
			})

			if runner.config == nil {
				t.Error("expected default config to be set")
			}
		})

		t.Run("with nil logger uses default", func(t *testing.T) {
			runner := NewRunner(RunnerOpts{
				Logger: nil,
			})

			if runner.logger == nil {
				t.Error("expected default logger to be set")
			}
		})

		t.Run("with nil output uses stdout", func(t *testing.T) {
			runner := NewRunner(RunnerOpts{
				Output: nil,
			})

			if runner.output != os.Stdout {
				t.Error("expected output to default to os.Stdout")
			}
		})

		t.Run("with nil httpClient uses default", func(t *testing.T) {
			runner := NewRunner(RunnerOpts{
				HTTPClient: nil,
			})

			if runner.httpClient != http.DefaultClient {
				t.Error("expected httpClient to default to http.DefaultClient")
			}
		})

		t.Run("with configPath sets field", func(t *testing.T) {
			runner := NewRunner(RunnerOpts{
				ConfigPath: "/test/path/config.toml",
			})

			if runner.configPath != "/test/path/config.toml" {
				t.Errorf("expected configPath to be set, got %s", runner.configPath)
			}
		})

		t.Run("with empty configPath", func(t *testing.T) {
			runner := NewRunner(RunnerOpts{
				ConfigPath: "",
			})

			if runner.configPath != "" {
				t.Errorf("expected empty configPath, got %s", runner.configPath)
			}
		})
	})

	t.Run("writeJSON", func(t *testing.T) {
		t.Run("writes formatted JSON successfully", func(t *testing.T) {
			output := &bytes.Buffer{}
			runner := NewRunner(RunnerOpts{Output: output})

			data := map[string]string{"key": "value"}
			err := runner.writeJSON(data, true)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			result := output.String()
			if !strings.Contains(result, `"key": "value"`) {
				t.Errorf("expected formatted JSON, got %s", result)
			}
			if !strings.HasSuffix(result, "\n") {
				t.Error("expected output to end with newline")
			}
		})

		t.Run("writes compact JSON successfully", func(t *testing.T) {
			output := &bytes.Buffer{}
			runner := NewRunner(RunnerOpts{Output: output})

			data := map[string]string{"key": "value"}
			err := runner.writeJSON(data, false)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			result := output.String()
			expected := `{"key":"value"}` + "\n"
			if result != expected {
				t.Errorf("expected %q, got %q", expected, result)
			}
		})

		t.Run("handles marshal error with non-serializable data", func(t *testing.T) {
			output := &bytes.Buffer{}
			runner := NewRunner(RunnerOpts{Output: output})

			// channels cannot be marshaled to JSON
			data := make(chan int)
			err := runner.writeJSON(data, false)

			if err == nil {
				t.Fatal("expected error for non-serializable data")
			}
			if !strings.Contains(err.Error(), "failed to marshal JSON") {
				t.Errorf("expected marshal error, got %v", err)
			}
		})

		t.Run("handles write failure", func(t *testing.T) {
			failing := &tu.FWriter{}
			runner := NewRunner(RunnerOpts{Output: failing})

			data := map[string]string{"key": "value"}
			err := runner.writeJSON(data, false)

			if err == nil {
				t.Fatal("expected error from failing writer")
			}
			if !strings.Contains(err.Error(), "failed to write output") {
				t.Errorf("expected write error, got %v", err)
			}
		})

		t.Run("handles newline write failure", func(t *testing.T) {
			data := map[string]string{"key": "value"}
			limitedWriter := tu.NewLimitedWriter(1, 0, &bytes.Buffer{})
			runner := NewRunner(RunnerOpts{Output: &limitedWriter})

			err := runner.writeJSON(data, false)

			if err == nil {
				t.Fatal("expected error writing newline")
			}
			if !strings.Contains(err.Error(), "failed to write newline") {
				t.Errorf("expected newline write error, got %v", err)
			}
		})
	})

	t.Run("writePlain", func(t *testing.T) {
		t.Run("writes plain text successfully", func(t *testing.T) {
			output := &bytes.Buffer{}
			runner := NewRunner(RunnerOpts{Output: output})

			err := runner.writePlain("hello %s", "world")

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			result := output.String()
			if result != "hello world" {
				t.Errorf("expected 'hello world', got %q", result)
			}
		})

		t.Run("writes plain text without formatting", func(t *testing.T) {
			output := &bytes.Buffer{}
			runner := NewRunner(RunnerOpts{Output: output})

			err := runner.writePlain("simple text")

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			result := output.String()
			if result != "simple text" {
				t.Errorf("expected 'simple text', got %q", result)
			}
		})

		t.Run("handles write failure", func(t *testing.T) {
			failing := &tu.FWriter{}
			runner := NewRunner(RunnerOpts{Output: failing})

			err := runner.writePlain("test")

			if err == nil {
				t.Fatal("expected error from failing writer")
			}
			if !strings.Contains(err.Error(), "failed to write output") {
				t.Errorf("expected write error, got %v", err)
			}
		})
	})

	t.Run("register", func(t *testing.T) {
		runner := NewRunner(RunnerOpts{})
		commands := runner.register()

		if len(commands) == 0 {
			t.Error("expected at least one command to be registered")
		}

		for i, cmd := range commands {
			if cmd == nil {
				t.Errorf("command at index %d is nil", i)
			}
		}
	})

	t.Run("saveTokens", func(t *testing.T) {
		t.Run("saves tokens successfully", func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.toml")

			config := shared.DefaultConfig()
			config.Credentials.Spotify.ClientID = "test_id"
			config.Credentials.Spotify.ClientSecret = "test_secret"

			if err := shared.SaveConfig(configPath, config); err != nil {
				t.Fatalf("failed to create test config: %v", err)
			}

			runner := NewRunner(RunnerOpts{
				Config:     config,
				ConfigPath: configPath,
			})

			token := &oauth2.Token{
				AccessToken:  "new_access_token",
				RefreshToken: "new_refresh_token",
			}

			err := runner.saveTokens(token)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			loadedConfig, err := shared.LoadConfig(configPath)
			if err != nil {
				t.Fatalf("failed to reload config: %v", err)
			}

			if loadedConfig.Credentials.Spotify.AccessToken != "new_access_token" {
				t.Errorf("expected access token to be updated, got %s", loadedConfig.Credentials.Spotify.AccessToken)
			}
			if loadedConfig.Credentials.Spotify.RefreshToken != "new_refresh_token" {
				t.Errorf("expected refresh token to be updated, got %s", loadedConfig.Credentials.Spotify.RefreshToken)
			}
		})

		t.Run("handles nil config error", func(t *testing.T) {
			runner := NewRunner(RunnerOpts{
				Config:     nil,
				ConfigPath: "/tmp/test.toml",
			})

			runner.config = nil

			token := &oauth2.Token{AccessToken: "test"}
			err := runner.saveTokens(token)

			if err == nil {
				t.Fatal("expected error with nil config")
			}
			if !strings.Contains(err.Error(), "config is nil") {
				t.Errorf("expected nil config error, got %v", err)
			}
		})

		t.Run("handles empty configPath", func(t *testing.T) {
			config := shared.DefaultConfig()
			runner := NewRunner(RunnerOpts{
				Config:     config,
				ConfigPath: "",
			})

			token := &oauth2.Token{
				AccessToken:  "new_token",
				RefreshToken: "new_refresh",
			}

			err := runner.saveTokens(token)
			if err != nil {
				t.Fatalf("expected no error with empty path, got %v", err)
			}

			if config.Credentials.Spotify.AccessToken != "new_token" {
				t.Error("expected config to be updated in memory")
			}
		})

		t.Run("handles SaveConfig failure", func(t *testing.T) {
			config := shared.DefaultConfig()
			invalidPath := "/root/readonly/impossible/config.toml"

			runner := NewRunner(RunnerOpts{
				Config:     config,
				ConfigPath: invalidPath,
			})

			token := &oauth2.Token{AccessToken: "test"}
			err := runner.saveTokens(token)

			if err == nil {
				t.Fatal("expected error with invalid path")
			}
			if !strings.Contains(err.Error(), "failed to save config") {
				t.Errorf("expected save config error, got %v", err)
			}
		})

		t.Run("handles Update error", func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.toml")

			config := shared.DefaultConfig()
			runner := NewRunner(RunnerOpts{
				Config:     config,
				ConfigPath: configPath,
			})

			err := runner.saveTokens(nil)
			if err == nil {
				t.Fatal("expected error when Update fails with nil token")
			}
			if !strings.Contains(err.Error(), "failed to update spotify configuration") {
				t.Errorf("expected update error, got %v", err)
			}
			if !strings.Contains(err.Error(), "token cannot be nil") {
				t.Errorf("expected nil token error in chain, got %v", err)
			}
		})

		t.Run("updates config reference", func(t *testing.T) {
			config := shared.DefaultConfig()
			runner := NewRunner(RunnerOpts{
				Config:     config,
				ConfigPath: "",
			})

			originalAccess := config.Credentials.Spotify.AccessToken
			token := &oauth2.Token{
				AccessToken:  "updated_access",
				RefreshToken: "updated_refresh",
			}

			err := runner.saveTokens(token)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if runner.config.Credentials.Spotify.AccessToken == originalAccess {
				t.Error("expected config reference to be updated")
			}
			if runner.config.Credentials.Spotify.AccessToken != "updated_access" {
				t.Errorf("expected updated access token in runner config")
			}
		})
	})
}
