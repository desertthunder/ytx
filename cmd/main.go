package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/desertthunder/song-migrations/internal/services"
	"github.com/desertthunder/song-migrations/internal/shared"
	"github.com/urfave/cli/v3"
)

var logger *log.Logger

func main() {
	logger = shared.NewLogger(nil)

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
		Name:    "ytx",
		Usage:   "Transfer playlists between Spotify & YouTube Music",
		Version: "0.5.0",
		Commands: []*cli.Command{
			setupCommand(),
			authCommand(runner),
			spotifyCommand(runner),
			apiCommand(runner),
			ytmusicCommand(runner),
			transferCommand(runner),
		},
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

func setupCommand() *cli.Command {
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
		Action: func(ctx context.Context, cmd *cli.Command) error {
			configPath := cmd.String("config")

			var config *shared.Config
			if _, err := os.Stat(configPath); err == nil {
				if config, err = shared.LoadConfig(configPath); err != nil {
					logger.Warn("failed to load config, using defaults", "error", err)
					config = shared.DefaultConfig()
				}
			} else {
				logger.Info("config file not found, creating from template", "path", configPath)
				if err := shared.CreateConfigFile(configPath); err != nil {
					logger.Warn("failed to create config file, using defaults", "error", err)
					config = shared.DefaultConfig()
				} else {
					logger.Info("config file created", "path", configPath)
					if config, err = shared.LoadConfig(configPath); err != nil {
						logger.Warn("failed to load created config, using defaults", "error", err)
						config = shared.DefaultConfig()
					}
				}
			}

			logger.Info("initializing database", "path", config.Database.Path)

			db, err := shared.NewDatabase(config.Database.Path)
			if err != nil {
				return fmt.Errorf("failed to create database: %w", err)
			}
			defer db.Close()

			shared.ConfigureDatabase(db, config.Database.MaxOpenConns, config.Database.MaxIdleConns)

			logger.Info("running database migrations")
			if err := shared.RunMigrations(db); err != nil {
				return fmt.Errorf("failed to run migrations: %w", err)
			}
			logger.Infof("setup complete for database: %v", config.Database.Path)
			return nil
		},
	}
}

// authCommand handles authentication operations (v0.1)
func authCommand(runner *Runner) *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Manage authentication",
		Commands: []*cli.Command{
			{
				Name:  "login",
				Usage: "Upload headers_auth.json to FastAPI /auth/upload endpoint",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "path"},
				},
				Action: runner.AuthLogin,
			},
			{
				Name:   "status",
				Usage:  "Check current authentication state (calls /health)",
				Action: runner.AuthStatus,
			},
		},
	}
}

// spotifyCommand handles Spotify operations (v0.2)
func spotifyCommand(runner *Runner) *cli.Command {
	return &cli.Command{
		Name:    "spotify",
		Aliases: []string{"spot"},
		Usage:   "Spotify playlist operations",
		Commands: []*cli.Command{
			{
				Name:  "playlists",
				Usage: "List Spotify playlists",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "limit",
						Usage: "Maximum number of playlists to return",
						Value: 50,
					},
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output raw JSON",
					},
					&cli.BoolFlag{
						Name:  "pretty",
						Usage: "Pretty-print output",
					},
					&cli.BoolFlag{
						Name:  "save",
						Usage: "Save API response locally",
					},
				},
				Action: runner.SpotifyPlaylists,
			},
			{
				Name:  "export",
				Usage: "Export playlist JSON for debugging",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Usage:    "Playlist ID to export",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output file path",
					},
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output raw JSON",
					},
					&cli.BoolFlag{
						Name:  "pretty",
						Usage: "Pretty-print output",
						Value: true,
					},
					&cli.BoolFlag{
						Name:  "save",
						Usage: "Save API response locally",
					},
				},
				Action: runner.SpotifyExport,
			},
		},
	}
}

// apiCommand handles direct API calls (v0.4)
func apiCommand(runner *Runner) *cli.Command {
	return &cli.Command{
		Name:  "api",
		Usage: "Direct API calls to FastAPI proxy",
		Commands: []*cli.Command{
			{
				Name:  "get",
				Usage: "Direct GET to FastAPI proxy, prints raw JSON",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "path",
					},
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output raw JSON",
						Value: true,
					},
				},
				Action: runner.APIGet,
			},
			{
				Name:  "post",
				Usage: "Direct POST with JSON body",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "path",
					},
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "data",
						Aliases:  []string{"d"},
						Usage:    "JSON body to send",
						Required: true,
					},
				},
				Action: runner.APIPost,
			},
		},
	}
}

// ytmusicCommand handles YouTube Music operations (v0.5)
func ytmusicCommand(runner *Runner) *cli.Command {
	return &cli.Command{
		Name:    "ytmusic",
		Aliases: []string{"ytm", "yt"},
		Usage:   "YouTube Music operations",
		Commands: []*cli.Command{
			{
				Name:  "search",
				Usage: "Search YouTube Music proxy for a track",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "query",
					},
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output raw JSON",
					},
					&cli.BoolFlag{
						Name:  "pretty",
						Usage: "Pretty-print output",
						Value: true,
					},
				},
				Action: runner.YTMusicSearch,
			},
			{
				Name:  "create",
				Usage: "Create playlist on YouTube Music",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "name",
					},
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "description",
						Usage: "Playlist description",
					},
					&cli.BoolFlag{
						Name:  "private",
						Usage: "Make playlist private",
						Value: true,
					},
				},
				Action: runner.YTMusicCreate,
			},
			{
				Name:  "add",
				Usage: "Add tracks to an existing playlist",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "playlist-id",
						Usage:    "Playlist ID to add tracks to",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "track",
						Usage:    "Track search query",
						Required: true,
					},
				},
				Action: runner.YTMusicAdd,
			},
		},
	}
}

// transferCommand handles playlist transfer operations (v0.6 stubs)
func transferCommand(runner *Runner) *cli.Command {
	return &cli.Command{
		Name:  "transfer",
		Usage: "Transfer playlists between services",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Run full Spotify → YouTube Music sync",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "source",
						Usage:    "Source playlist name or ID",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "dest",
						Usage:    "Destination playlist name",
						Required: true,
					},
				},
				Action: runner.TransferRun,
			},
			{
				Name:  "diff",
				Usage: "Compare and show missing tracks between two playlists",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "source-id",
						Usage:    "Source playlist ID",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "dest-id",
						Usage:    "Destination playlist ID",
						Required: true,
					},
				},
				Action: runner.TransferDiff,
			},
		},
	}
}
