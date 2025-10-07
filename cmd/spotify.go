package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/desertthunder/ytx/internal/server"
	"github.com/desertthunder/ytx/internal/services"
	"github.com/desertthunder/ytx/internal/shared"
	"github.com/urfave/cli/v3"
)

// SpotifyAuth performs OAuth2 authentication flow for Spotify.
// Starts a local HTTP server, opens browser for user authorization,
// exchanges auth code for tokens, and saves tokens to config.
func (r *Runner) SpotifyAuth(ctx context.Context, cmd *cli.Command) error {
	configPath := cmd.String("config")

	config := r.config
	if config == nil {
		var err error
		if _, statErr := os.Stat(configPath); statErr == nil {
			config, err = shared.LoadConfig(configPath)
			if err != nil {
				r.logger.Warn("failed to load config, using defaults", "error", err)
				config = shared.DefaultConfig()
			}
		} else {
			config = shared.DefaultConfig()
		}
	}

	if config.Credentials.Spotify.ClientID == "" || config.Credentials.Spotify.ClientSecret == "" {
		return fmt.Errorf("%w: Spotify client_id and client_secret must be set in config.toml", shared.ErrInvalidArgument)
	}

	spotifyService, err := services.NewSpotifyService(map[string]string{
		"client_id":     config.Credentials.Spotify.ClientID,
		"client_secret": config.Credentials.Spotify.ClientSecret,
		"redirect_uri":  config.Credentials.Spotify.RedirectURI,
	})
	if err != nil {
		return fmt.Errorf("failed to create Spotify service: %w", err)
	}

	state, err := shared.GenerateState()
	if err != nil {
		return fmt.Errorf("failed to generate state token: %w", err)
	}

	authURL := spotifyService.GetAuthURL(state)

	oauthHandler := server.NewOAuthHandler(spotifyService.GetOAuthConfig(), state)
	router := server.NewBasicRouter()
	router.Handler(oauthHandler)

	serverAddr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	httpServer := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	serverErrors := make(chan error, 1)
	go func() {
		r.logger.Info("starting OAuth server", "address", serverAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	time.Sleep(100 * time.Millisecond)

	r.writePlain("→ Opening browser for Spotify authorization...\n")
	if err := shared.OpenBrowser(authURL); err != nil {
		r.logger.Warn("failed to open browser automatically", "error", err)
		r.writePlain("\n⚠ Could not open browser automatically.\n")
		r.writePlain("Please open this URL in your browser:\n%s\n\n", authURL)
	}

	r.writePlain("→ Waiting for authorization (2 minute timeout)...\n")

	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()

	var result server.OAuthResult

	select {
	case result = <-oauthHandler.Result():
		// Got result from callback
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case <-timeout.C:
		return fmt.Errorf("%w: authorization timed out after 2 minutes", shared.ErrTimeout)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		r.logger.Warn("error shutting down server", "error", err)
	}

	if result.Error() != nil {
		return fmt.Errorf("authorization failed: %w", result.Error())
	}

	if result.Token == nil {
		return fmt.Errorf("no token received")
	}

	config.Credentials.Spotify.AccessToken = result.Token.AccessToken
	config.Credentials.Spotify.RefreshToken = result.Token.RefreshToken

	if err := shared.SaveConfig(configPath, config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	r.writePlain("\n✓ Authorization successful\n")
	r.writePlain("✓ Tokens saved to %s\n\n", configPath)
	r.writePlain("You can now use: ytx spotify playlists\n")

	return nil
}

// SpotifyPlaylists lists Spotify playlists with optional limit.
func (r *Runner) SpotifyPlaylists(ctx context.Context, cmd *cli.Command) error {
	if r.spotify == nil {
		return fmt.Errorf("%w: Spotify service not initialized", shared.ErrServiceUnavailable)
	}

	limit := cmd.Int("limit")
	useJSON := cmd.Bool("json")
	pretty := cmd.Bool("pretty")
	save := cmd.Bool("save")

	r.logger.Info("listing spotify playlists", "limit", limit)

	playlists, err := r.spotify.GetPlaylists(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", shared.ErrAPIRequest, err)
	}

	if limit > 0 && limit < len(playlists) {
		playlists = playlists[:limit]
	}

	if save {
		saveFile := "spotify_playlists.json"
		data, err := shared.MarshalJSON(playlists, true)
		if err != nil {
			return fmt.Errorf("failed to marshal playlists: %w", err)
		}
		if err := os.WriteFile(saveFile, data, 0644); err != nil {
			r.logger.Warn("failed to save playlists", "error", err)
		} else {
			r.logger.Info("playlists saved", "file", saveFile)
		}
	}

	if useJSON {
		return r.writeJSON(playlists, pretty)
	}

	r.writePlain("Found %d playlists:\n\n", len(playlists))
	for i, p := range playlists {
		r.writePlain("%d. %s\n", i+1, p.Name)
		if p.Description != "" {
			r.writePlain("   Description: %s\n", p.Description)
		}
		r.writePlain("   ID: %s\n", p.ID)
		r.writePlain("   Tracks: %d\n", p.TrackCount)
		if p.Public {
			r.writePlain("   Visibility: Public\n")
		} else {
			r.writePlain("   Visibility: Private\n")
		}
		r.writePlain("\n")
	}

	return nil
}

// SpotifyExport exports a playlist with all tracks to JSON.
func (r *Runner) SpotifyExport(ctx context.Context, cmd *cli.Command) error {
	if r.spotify == nil {
		return fmt.Errorf("%w: Spotify service not initialized", shared.ErrServiceUnavailable)
	}

	playlistID := cmd.String("id")
	if playlistID == "" {
		return fmt.Errorf("%w: --id flag is required", shared.ErrMissingArgument)
	}

	outputFile := cmd.String("output")
	useJSON := cmd.Bool("json")
	pretty := cmd.Bool("pretty")
	save := cmd.Bool("save")

	r.logger.Info("exporting spotify playlist", "id", playlistID)

	export, err := r.spotify.ExportPlaylist(ctx, playlistID)
	if err != nil {
		return fmt.Errorf("%w: %v", shared.ErrAPIRequest, err)
	}

	if outputFile == "" && (save || !useJSON) {
		outputFile = fmt.Sprintf("spotify_%s.json", export.Playlist.Name)
	}

	if outputFile != "" {
		data, err := shared.MarshalJSON(export, true)
		if err != nil {
			return fmt.Errorf("failed to marshal export: %w", err)
		}
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		r.logger.Info("playlist exported", "file", outputFile, "tracks", len(export.Tracks))
		r.writePlain("✓ Playlist exported to %s\n", outputFile)
		r.writePlain("  Playlist: %s\n", export.Playlist.Name)
		r.writePlain("  Tracks: %d\n", len(export.Tracks))
		return nil
	}

	if useJSON {
		return r.writeJSON(export, pretty)
	}

	r.writePlain("Playlist: %s\n", export.Playlist.Name)
	if export.Playlist.Description != "" {
		r.writePlain("Description: %s\n", export.Playlist.Description)
	}
	r.writePlain("Tracks: %d\n\n", len(export.Tracks))

	for i, track := range export.Tracks {
		r.writePlain("%d. %s - %s\n", i+1, track.Artist, track.Title)
		if track.Album != "" {
			r.writePlain("   Album: %s\n", track.Album)
		}
		if track.ISRC != "" {
			r.writePlain("   ISRC: %s\n", track.ISRC)
		}
	}

	return nil
}

// spotifyCommand handles Spotify operations (v0.2)
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
