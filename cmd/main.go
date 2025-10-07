package main

import (
	"context"
	"errors"
	"os"

	"github.com/desertthunder/song-migrations/internal/services"
	"github.com/desertthunder/song-migrations/internal/shared"
	"github.com/urfave/cli/v3"
)

func main() {
	logger := shared.NewLogger(nil)

	var spotifyService services.Service
	var youtubeService services.Service

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
			spotifyService = svc
		}
	}

	youtubeService = services.NewYouTubeService(config.Credentials.YouTube.ProxyURL)
	apiService := services.NewAPIService(config.Credentials.YouTube.ProxyURL, nil)

	runner := NewRunner(RunnerConfig{
		Config:  config,
		Spotify: spotifyService,
		YouTube: youtubeService,
		API:     apiService,
		Logger:  logger,
	})

	app := &cli.Command{
		Name:     "ytx",
		Usage:    "Transfer playlists between Spotify & YouTube Music",
		Version:  "0.5.0",
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

func setupCommand(r *Runner) *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Initialize database and run migrations",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to configuration file",
				Value:   "config.toml",
			},
		},
		Action: r.Setup,
	}
}
