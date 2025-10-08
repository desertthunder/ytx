// submodule cmd contains command definitions
package main

import "github.com/urfave/cli/v3"

// spotifyCommand handles Spotify operations
func spotifyCommand(r *Runner) *cli.Command {
	return &cli.Command{
		Name:    "spotify",
		Aliases: []string{"spot"},
		Usage:   "Spotify playlist operations",
		Commands: []*cli.Command{
			{
				Name:  "auth",
				Usage: "Authenticate with Spotify using OAuth2",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "Path to configuration file",
						Value:   "config.toml",
					},
				},
				Action: r.SpotifyAuth,
			},
			{
				Name:  "playlists",
				Usage: "List Spotify playlists",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "Path to configuration file",
						Value:   "config.toml",
					},
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
				Action: r.SpotifyPlaylists,
			},
			{
				Name:  "export",
				Usage: "Export playlist JSON for debugging",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "Path to configuration file",
						Value:   "config.toml",
					},
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
				Action: r.SpotifyExport,
			},
		},
	}
}

// apiCommand handles direct (proxy) API calls
func apiCommand(r *Runner) *cli.Command {
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
				Action: r.APIGet,
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
				Action: r.APIPost,
			},
			{
				Name:  "dump",
				Usage: "Full proxy state dump (cached playlists, songs, etc)",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "pretty",
						Usage: "Pretty-print output",
						Value: true,
					},
					&cli.BoolFlag{
						Name:  "save",
						Usage: "Save dump to api_dump.json",
						Value: false,
					},
				},
				Action: r.APIDump,
			},
		},
	}
}

// setupCommand handles setup operations for database and authentication.
func setupCommand(r *Runner) *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Setup and configuration commands",
		Commands: []*cli.Command{
			{
				Name:  "database",
				Usage: "Initialize database and run migrations",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "Path to configuration file",
						Value:   "config.toml",
					},
				},
				Action: r.SetupDatabase,
			},
			{
				Name:    "youtube",
				Aliases: []string{"yt", "ytmusic"},
				Usage:   "Configure YouTube Music authentication from browser headers",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "curl",
						Usage: "cURL command from browser DevTools (Copy as cURL)",
					},
					&cli.StringFlag{
						Name:  "curl-file",
						Usage: "Path to .sh file containing cURL command",
					},
					&cli.StringFlag{
						Name:  "output",
						Usage: "Output path for browser.json (default: ~/.ytx/browser.json)",
					},
				},
				Action: r.SetupYouTube,
			},
		},
	}
}

// transferCommand handles playlist transfer operations (v0.6 stubs)
func transferCommand(r *Runner) *cli.Command {
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
				},
				Action: r.TransferRun,
			},
			{
				Name:   "ui",
				Usage:  "Interactive TUI for playlist transfer",
				Action: r.TransferUI,
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
					&cli.StringFlag{
						Name:     "source-service",
						Usage:    "Source service (spotify or youtube)",
						Value:    "spotify",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "dest-service",
						Usage:    "Destination service (spotify or youtube)",
						Value:    "youtube",
						Required: false,
					},
				},
				Action: r.TransferDiff,
			},
		},
	}
}

// authCommand handles authentication operations
func authCommand(r *Runner) *cli.Command {
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
				Action: r.AuthLogin,
			},
			{
				Name:   "status",
				Usage:  "Check current authentication state (calls /health)",
				Action: r.AuthStatus,
			},
		},
	}
}

// ytmusicCommand handles YouTube Music operations (v0.5)
func ytmusicCommand(r *Runner) *cli.Command {
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
				Action: r.YTMusicSearch,
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
				Action: r.YTMusicCreate,
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
				Action: r.YTMusicAdd,
			},
		},
	}
}

// tuiCommand returns the top-level TUI command for interactive playlist management.
func tuiCommand(r *Runner) *cli.Command {
	return &cli.Command{
		Name:    "tui",
		Aliases: []string{"interactive", "ui"},
		Usage:   "Launch interactive TUI for playlist transfer",
		Action:  r.TUI,
	}
}
