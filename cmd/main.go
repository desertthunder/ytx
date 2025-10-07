package main

import (
	"context"
	"errors"
	"os"

	"github.com/desertthunder/ytx/internal/services"
	"github.com/desertthunder/ytx/internal/shared"
	"github.com/urfave/cli/v3"
)

func main() {
	var spot services.Service
	var yt services.Service

	logger := shared.NewLogger(nil)
	config := shared.DefaultConfig()

	if _, err := os.Stat("config.toml"); err == nil {
		if loadedConfig, err := shared.LoadConfig("config.toml"); err == nil {
			config = loadedConfig
		}
	}

	if config.Credentials.Spotify.ClientID != "" && config.Credentials.Spotify.ClientSecret != "" {
		if svc, err := services.NewSpotifyService(map[string]string{
			"client_id":     config.Credentials.Spotify.ClientID,
			"client_secret": config.Credentials.Spotify.ClientSecret,
			"redirect_uri":  config.Credentials.Spotify.RedirectURI,
		}); err == nil {
			spot = svc

			if config.Credentials.Spotify.AccessToken != "" {
				ctx := context.Background()
				if err := svc.Authenticate(ctx, map[string]string{
					"access_token": config.Credentials.Spotify.AccessToken,
				}); err != nil {
					logger.Warn("failed to authenticate with stored token", "error", err)
				} else {
					logger.Debug("authenticated with stored access token")
				}
			}
		}
	}

	yt = services.NewYouTubeService(config.Credentials.YouTube.ProxyURL)

	if config.Credentials.YouTube.HeadersPath != "" {
		ctx := context.Background()
		headersPath := config.Credentials.YouTube.HeadersPath

		if absPath, err := shared.AbsolutePath(headersPath); err == nil {
			headersPath = absPath
		}

		logger.Info("authenticating YouTube service", "headers_path", headersPath)
		if err := yt.Authenticate(ctx, map[string]string{"auth_file": headersPath}); err != nil {
			logger.Error("failed to authenticate YouTube service", "error", err)
		} else {
			logger.Info("authenticated YouTube service successfully")
		}
	}

	api := services.NewAPIService(config.Credentials.YouTube.ProxyURL, nil)
	if config.Credentials.YouTube.HeadersPath != "" {
		if absPath, err := shared.AbsolutePath(config.Credentials.YouTube.HeadersPath); err == nil {
			api.SetAuthFile(absPath)
			logger.Info("configured API service with auth file", "headers_path", absPath)
		}
	}

	rconf := RunnerConfig{
		Config:  config,
		Spotify: spot,
		YouTube: yt,
		API:     api,
		Logger:  logger,
	}
	runner := NewRunner(rconf)

	app := &cli.Command{
		Name:     "ytx",
		Usage:    "Transfer playlists between Spotify & YouTube Music",
		Version:  "0.2.0",
		Commands: runner.register(),
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		err_ := errors.Unwrap(err)
		if errors.Is(err_, shared.ErrNotImplemented) {
			logger.Warn("not implemented")
			os.Exit(0)
		} else {
			logger.Fatalf("application error: %v", err)
		}
	}
}
