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
		creds := config.Credentials.Spotify.Map()
		if svc, err := services.NewSpotifyService(creds); err == nil {
			spot = svc

			if config.Credentials.Spotify.AccessToken != "" {
				ctx := context.Background()
				if err := svc.Authenticate(ctx, creds); err != nil {
					logger.Warnf("failed to authenticate with stored token %v", err)
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

		logger.Debugf("authenticating YouTube service with header path %v", headersPath)
		if err := yt.Authenticate(ctx, map[string]string{"auth_file": headersPath}); err != nil {
			logger.Errorf("failed to authenticate YouTube service %v", err)
		} else {
			logger.Debug("authenticated YouTube service successfully")
		}
	}

	api := services.NewAPIService(config.Credentials.YouTube.ProxyURL, nil)
	if config.Credentials.YouTube.HeadersPath != "" {
		if absPath, err := shared.AbsolutePath(config.Credentials.YouTube.HeadersPath); err == nil {
			api.SetAuthFile(absPath)
			logger.Debugf("configured API service with auth file header path %v", absPath)
		}
	}

	rconf := RunnerOpts{
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
