package shared

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

//go:embed config.example.toml
var exampleConf []byte

// Config represents the application configuration loaded from a TOML file.
type Config struct {
	Credentials CredentialsConfig `toml:"credentials"`
	Database    DatabaseConfig    `toml:"database"`
	Server      ServerConfig      `toml:"server"`
}

// CredentialsConfig contains service-specific credentials.
type CredentialsConfig struct {
	Spotify SpotifyConfig `toml:"spotify"`
	YouTube YouTubeConfig `toml:"youtube"`
}

// SpotifyConfig contains Spotify API credentials.
type SpotifyConfig struct {
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
	RedirectURI  string `toml:"redirect_uri"`
	AccessToken  string `toml:"access_token,omitempty"`
	RefreshToken string `toml:"refresh_token,omitempty"`
}

// YouTubeConfig contains YouTube Music API credentials.
type YouTubeConfig struct {
	APIKey      string `toml:"api_key"`
	ProxyURL    string `toml:"proxy_url"`
	HeadersPath string `toml:"headers_path"`
}

// DatabaseConfig contains database connection settings.
type DatabaseConfig struct {
	Path         string `toml:"path"`
	MaxOpenConns int    `toml:"max_open_conns"`
	MaxIdleConns int    `toml:"max_idle_conns"`
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

func (s SpotifyConfig) Map() map[string]string {
	return map[string]string{
		"client_id":     s.ClientID,
		"client_secret": s.ClientSecret,
		"redirect_uri":  s.RedirectURI,
	}
}

// LoadConfig reads and parses a TOML configuration file from the specified path.
//
// Expands ~ in file paths to the user's home directory.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config.Credentials.YouTube.HeadersPath = ExpandPath(config.Credentials.YouTube.HeadersPath)
	config.Database.Path = ExpandPath(config.Database.Path)

	return &config, nil
}

// DefaultConfig returns a Config with sensible defaults loaded from the embedded example config.
func DefaultConfig() *Config {
	var config Config
	if err := toml.Unmarshal(exampleConf, &config); err != nil {
		panic(fmt.Sprintf("failed to parse embedded default config: %v", err))
	}
	return &config
}

// CreateConfigFile creates a config.toml file at the specified path using the embedded example config.
func CreateConfigFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists at %s: %w", path, err)
	}

	if err := os.WriteFile(path, exampleConf, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SaveConfig writes a Config struct to a TOML file at the specified path.
func SaveConfig(path string, config *Config) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open config file for writing: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}
