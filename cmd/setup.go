package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/desertthunder/ytx/internal/shared"
	"github.com/urfave/cli/v3"
)

// SetupDatabase initializes the database and runs migrations.
func (r *Runner) SetupDatabase(ctx context.Context, cmd *cli.Command) error {
	configPath := cmd.String("config")

	var config *shared.Config
	if _, err := os.Stat(configPath); err == nil {
		if config, err = shared.LoadConfig(configPath); err != nil {
			r.logger.Warn("failed to load config, using defaults", "error", err)
			config = shared.DefaultConfig()
		}
	} else {
		r.logger.Info("config file not found, creating from template", "path", configPath)
		if err := shared.CreateConfigFile(configPath); err != nil {
			r.logger.Warn("failed to create config file, using defaults", "error", err)
			config = shared.DefaultConfig()
		} else {
			r.logger.Info("config file created", "path", configPath)
			if config, err = shared.LoadConfig(configPath); err != nil {
				r.logger.Warn("failed to load created config, using defaults", "error", err)
				config = shared.DefaultConfig()
			}
		}
	}

	r.logger.Info("initializing database", "path", config.Database.Path)

	db, err := shared.NewDatabase(config.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer db.Close()

	shared.ConfigureDatabase(db, config.Database.MaxOpenConns, config.Database.MaxIdleConns)

	r.logger.Info("running database migrations")
	if err := shared.RunMigrations(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	r.logger.Infof("setup complete for database: %v", config.Database.Path)
	return nil
}

// SetupYouTube configures YouTube Music authentication from browser headers.
//
// Accepts a cURL command generates browser.json.
func (r *Runner) SetupYouTube(ctx context.Context, cmd *cli.Command) error {
	curlCmd := cmd.String("curl")
	curlFile := cmd.String("curl-file")
	outputPath := cmd.String("output")

	if curlCmd == "" && curlFile == "" {
		return fmt.Errorf("%w: either --curl or --curl-file must be provided", shared.ErrMissingArgument)
	}

	if curlCmd != "" && curlFile != "" {
		return fmt.Errorf("%w: cannot specify both --curl and --curl-file", shared.ErrInvalidArgument)
	}

	r.logger.Info("parsing cURL command for YouTube Music headers")

	var curlHeaders *shared.CurlHeaders
	var err error

	if curlFile != "" {
		curlHeaders, err = shared.ParseCurlFile(curlFile)
		if err != nil {
			return fmt.Errorf("failed to parse cURL file: %w", err)
		}
		r.logger.Info("parsed cURL from file", "file", curlFile)
	} else {
		curlHeaders, err = shared.ParseCurlCommand(curlCmd)
		if err != nil {
			return fmt.Errorf("failed to parse cURL command: %w", err)
		}
		r.logger.Info("parsed cURL command")
	}

	headersRaw := curlHeaders.ToHeadersRaw()

	r.logger.Debug("generated headers_raw", "length", len(headersRaw))
	r.logger.Info("calling YouTube Music proxy setup endpoint")

	setupResp, err := r.api.SetupBrowser(ctx, headersRaw)
	if err != nil {
		return fmt.Errorf("setup request failed: %w", err)
	}

	if !setupResp.Success {
		return fmt.Errorf("setup failed: %s", setupResp.Message)
	}

	r.logger.Info("setup successful", "message", setupResp.Message)

	if outputPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		outputPath = filepath.Join(homeDir, ".ytx", "browser.json")
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	authJSON, err := json.MarshalIndent(setupResp.AuthContent, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth content: %w", err)
	}

	if err := os.WriteFile(outputPath, authJSON, 0600); err != nil {
		return fmt.Errorf("failed to write auth file: %w", err)
	}

	r.logger.Info("browser.json saved", "path", outputPath)

	r.writePlain("✓ YouTube Music authentication configured successfully\n")
	r.writePlain("Auth file saved to: %s\n", outputPath)
	r.writePlainln("Next steps:")
	r.writePlain("1. Update config.toml with: credentials.youtube.headers_path = \"%s\"\n", outputPath)
	r.writePlain("2. Run 'ytx ytmusic search \"your song\"' to test authentication\n")

	return nil
}
