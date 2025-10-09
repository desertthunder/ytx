package main

import (
	"context"
	"errors"
	"os"

	"github.com/desertthunder/ytx/internal/services"
	"github.com/desertthunder/ytx/internal/shared"
	"github.com/desertthunder/ytx/internal/tasks"
	"github.com/urfave/cli/v3"
	"golang.org/x/oauth2"
)

func main() {
	var spot services.Service
	var yt services.Service

	logger := shared.NewLogger(nil)
	config := shared.DefaultConfig()
	configPath := "config.toml"

	if _, err := os.Stat(configPath); err == nil {
		if loadedConfig, err := shared.LoadConfig(configPath); err == nil {
			config = loadedConfig
		}
	}

	if config.Credentials.Spotify.ClientID != "" && config.Credentials.Spotify.ClientSecret != "" {
		creds := config.Credentials.Spotify.Map()
		if svc, err := services.NewSpotifyService(creds); err == nil {
			spot = svc

			if config.Credentials.Spotify.AccessToken != "" {
				ctx := context.Background()
				creds["access_token"] = config.Credentials.Spotify.AccessToken
				creds["refresh_token"] = config.Credentials.Spotify.RefreshToken
				if err := svc.Authenticate(ctx, creds); err != nil {
					logger.Warnf("failed to authenticate with stored token %v", err)
				} else {
					logger.Debug("authenticated with stored access token")
				}
			}
		}
	}

	rconf := RunnerOpts{
		Config:     config,
		ConfigPath: configPath,
		Spotify:    spot,
		YouTube:    nil,
		API:        nil,
		Logger:     logger,
	}
	runner := NewRunner(rconf)

	if spotifyService, ok := spot.(*services.SpotifyService); ok && spot != nil {
		spotifyService.SetTokenRefreshCallback(func(token *oauth2.Token) {
			logger.Info("token refreshed, saving to config")
			if err := runner.saveTokens(token); err != nil {
				logger.Warnf("failed to save refreshed tokens: %v", err)
			}
		})
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

	runner.youtube = yt
	runner.api = api
	runner.engine = tasks.NewPlaylistEngine(spot, yt, api)

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
