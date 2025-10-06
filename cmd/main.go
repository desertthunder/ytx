package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/desertthunder/song-migrations/internal/shared"
	"github.com/urfave/cli/v3"
)

var logger *log.Logger

func main() {
	logger = shared.NewLogger(nil)

	app := &cli.Command{
		Name:    "ytx",
		Usage:   "Transfer playlists between Spotify & YouTube Music",
		Version: "0.1.0",
		Commands: []*cli.Command{
			setupCommand(),
			authCommand(),
			spotifyCommand(),
			apiCommand(),
			ytmusicCommand(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		logger.Fatal("application error", "error", err)
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
				// Config file exists, load it
				config, err = shared.LoadConfig(configPath)
				if err != nil {
					logger.Warn("failed to load config, using defaults", "error", err)
					config = shared.DefaultConfig()
				}
			} else {
				// Config file doesn't exist, create it
				logger.Info("config file not found, creating from template", "path", configPath)
				if err := shared.CreateConfigFile(configPath); err != nil {
					logger.Warn("failed to create config file, using defaults", "error", err)
					config = shared.DefaultConfig()
				} else {
					logger.Info("config file created", "path", configPath)
					// Load the newly created config
					config, err = shared.LoadConfig(configPath)
					if err != nil {
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

			logger.Info("setup complete", "database", config.Database.Path)
			return nil
		},
	}
}

// authCommand handles authentication operations (v0.1)
func authCommand() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Manage authentication",
		Commands: []*cli.Command{
			{
				Name:      "login",
				Usage:     "Upload headers_auth.json to FastAPI /auth/upload endpoint",
				ArgsUsage: "<path-to-headers_auth.json>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.NArg() != 1 {
						return fmt.Errorf("expected 1 argument: path to headers_auth.json")
					}
					filePath := cmd.Args().Get(0)
					logger.Info("uploading auth headers", "file", filePath)
					// TODO: implement auth upload
					logger.Warn("not implemented")
					return nil
				},
			},
			{
				Name:  "status",
				Usage: "Check current authentication state (calls /health)",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					logger.Info("checking auth status")
					// TODO: implement health check
					logger.Warn("not implemented")
					return nil
				},
			},
		},
	}
}

// spotifyCommand handles Spotify operations (v0.2)
func spotifyCommand() *cli.Command {
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
						Value: 50},
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
				Action: func(ctx context.Context, cmd *cli.Command) error {
					limit := cmd.Int("limit")
					logger.Info("listing spotify playlists", "limit", limit)
					// TODO: implement playlist listing
					logger.Warn("not implemented")
					return nil
				},
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
				Action: func(ctx context.Context, cmd *cli.Command) error {
					playlistID := cmd.String("id")
					output := cmd.String("output")
					logger.Info("exporting spotify playlist", "id", playlistID, "output", output)
					// TODO: implement playlist export
					logger.Warn("not implemented")
					return nil
				},
			},
		},
	}
}

// apiCommand handles direct API calls (v0.4)
func apiCommand() *cli.Command {
	return &cli.Command{
		Name:  "api",
		Usage: "Direct API calls to FastAPI proxy",
		Commands: []*cli.Command{
			{
				Name:      "get",
				Usage:     "Direct GET to FastAPI proxy, prints raw JSON",
				ArgsUsage: "<path>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output raw JSON",
						Value: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.NArg() != 1 {
						return fmt.Errorf("expected 1 argument: API path")
					}
					path := cmd.Args().Get(0)
					logger.Info("GET request", "path", path)
					// TODO: implement GET request
					logger.Warn("not implemented")
					return nil
				},
			},
			{
				Name:      "post",
				Usage:     "Direct POST with JSON body",
				ArgsUsage: "<path>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "data",
						Aliases:  []string{"d"},
						Usage:    "JSON body to send",
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.NArg() != 1 {
						return fmt.Errorf("expected 1 argument: API path")
					}
					path := cmd.Args().Get(0)
					data := cmd.String("data")
					logger.Info("POST request", "path", path, "data", data)
					// TODO: implement POST request
					logger.Warn("not implemented")
					return nil
				},
			},
		},
	}
}

// ytmusicCommand handles YouTube Music operations (v0.5)
func ytmusicCommand() *cli.Command {
	return &cli.Command{
		Name:    "ytmusic",
		Aliases: []string{"ytm", "yt"},
		Usage:   "YouTube Music operations",
		Commands: []*cli.Command{
			{
				Name:      "search",
				Usage:     "Search YouTube Music proxy for a track",
				ArgsUsage: "<query>",
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
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.NArg() != 1 {
						return fmt.Errorf("expected 1 argument: search query")
					}
					query := cmd.Args().Get(0)
					logger.Info("searching youtube music", "query", query)
					// TODO: implement search
					return nil
				},
			},
			{
				Name:      "create",
				Usage:     "Create playlist on YouTube Music",
				ArgsUsage: "<playlist-name>",
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
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.NArg() != 1 {
						return fmt.Errorf("expected 1 argument: playlist name")
					}
					name := cmd.Args().Get(0)
					logger.Info("creating youtube music playlist", "name", name)
					// TODO: implement playlist creation
					return nil
				},
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
				Action: func(ctx context.Context, cmd *cli.Command) error {
					playlistID := cmd.String("playlist-id")
					track := cmd.String("track")
					logger.Info("adding track to playlist", "playlist_id", playlistID, "track", track)
					// TODO: implement add track
					return nil
				},
			},
		},
	}
}
