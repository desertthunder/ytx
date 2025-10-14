package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/desertthunder/ytx/internal/formatter"
	"github.com/desertthunder/ytx/internal/models"
	"github.com/desertthunder/ytx/internal/server"
	"github.com/desertthunder/ytx/internal/services"
	"github.com/desertthunder/ytx/internal/shared"
	"github.com/urfave/cli/v3"
	"golang.org/x/oauth2"
)

// SpotifyReauth performs the full OAuth2 flow to get new tokens
func (r *Runner) SpotifyReauth(ctx context.Context, configPath string, config *shared.Config, srv services.OAuthService) (*shared.Config, error) {
	token, err := r.doOAuth(config, srv, "reauthorization")
	if err != nil {
		return nil, err
	}

	if err := config.Credentials.Spotify.Update(token); err != nil {
		return nil, fmt.Errorf("failed to update spotify configuration: %w", err)
	}

	if err := shared.SaveConfig(configPath, config); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	r.writePlainln("✓ Reauthorization successful")
	r.writePlain("✓ New tokens saved to %s\n", configPath)

	return config, nil
}

// SpotifyAuth performs OAuth2 authentication flow for Spotify.
//
// Starts a local HTTP server, opens browser for user authorization, and exchanges auth code for tokens.
func (r *Runner) SpotifyAuth(ctx context.Context, cmd *cli.Command) error {
	configPath := cmd.String("config")

	config := r.config
	if config == nil {
		var err error
		if _, statErr := os.Stat(configPath); statErr == nil {
			config, err = shared.LoadConfig(configPath)
			if err != nil {
				r.logger.Warnf("failed to load config, using defaults %v", err)
				config = shared.DefaultConfig()
			}
		} else {
			config = shared.DefaultConfig()
		}
	}

	if config.Credentials.Spotify.ClientID == "" || config.Credentials.Spotify.ClientSecret == "" {
		return fmt.Errorf("%w: Spotify client_id and client_secret must be set in config.toml", shared.ErrInvalidArgument)
	}

	spotifyService, err := services.NewSpotifyService(config.Credentials.Spotify.Map())
	if err != nil {
		return fmt.Errorf("failed to create Spotify service: %w", err)
	}

	token, err := r.doOAuth(config, spotifyService, "authorization")
	if err != nil {
		return err
	}

	if err := config.Credentials.Spotify.Update(token); err != nil {
		return fmt.Errorf("failed to update spotify configuration: %w", err)
	}

	if err := shared.SaveConfig(configPath, config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	r.writePlainln("✓ Authorization successful")
	r.writePlain("✓ Tokens saved to %s\n\n", configPath)
	r.writePlain("You can now use: ytx spotify playlists\n")

	return nil
}

// SpotifyPlaylists lists Spotify playlists with optional limit.
func (r *Runner) SpotifyPlaylists(ctx context.Context, cmd *cli.Command) error {
	limit := cmd.Int("limit")
	useJSON := cmd.Bool("json")
	pretty := cmd.Bool("pretty")
	save := cmd.Bool("save")
	userFilter := cmd.String("user")

	if r.spotify == nil {
		return fmt.Errorf("%w: Spotify service not initialized", shared.ErrServiceUnavailable)
	}

	r.logger.Infof("listing spotify playlists with limit %v", limit)

	playlists, err := r.spotify.GetPlaylists(ctx)
	if err != nil {
		if reauthed, authErr := r.handleSpotifyAuthError(ctx, err, cmd); reauthed {
			if authErr != nil {
				return authErr
			}
			if playlists, err = r.spotify.GetPlaylists(ctx); err != nil {
				return fmt.Errorf("%w: %v", shared.ErrAPIRequest, err)
			}
		} else {
			return fmt.Errorf("%w: %v", shared.ErrAPIRequest, err)
		}
	}

	// Filter by user if specified
	if userFilter != "" {
		spotifySvc, ok := r.spotify.(*services.SpotifyService)
		if !ok {
			return fmt.Errorf("spotify service type assertion failed")
		}

		var targetUserID string
		if userFilter == "me" {
			user, err := spotifySvc.UserProfile(ctx)
			if err != nil {
				return fmt.Errorf("failed to get user profile: %w", err)
			}
			targetUserID = user.ID
			r.logger.Debugf("filtering playlists for current user: %v", targetUserID)
		} else {
			targetUserID = userFilter
			r.logger.Debugf("filtering playlists for user: %v", targetUserID)
		}

		// Filter playlists by owner
		var filtered []models.Playlist
		for _, pl := range playlists {
			spotifyPl, err := spotifySvc.Playlist(ctx, pl.ID)
			if err != nil {
				r.logger.Warnf("failed to get playlist details for %v: %v", pl.ID, err)
				continue
			}
			if spotifyPl.Owner.ID == targetUserID {
				filtered = append(filtered, pl)
			}
		}
		playlists = filtered
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

	if userFilter != "" {
		r.writePlain("Found %d playlists (filtered by user: %s):\n\n", len(playlists), userFilter)
	} else {
		r.writePlain("Found %d playlists:\n\n", len(playlists))
	}
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
	outputFile := cmd.String("output")
	useJSON := cmd.Bool("json")
	pretty := cmd.Bool("pretty")
	save := cmd.Bool("save")
	playlistID := cmd.String("id")
	format := cmd.String("format")

	if playlistID == "" {
		return fmt.Errorf("%w: --id flag is required", shared.ErrMissingArgument)
	}

	if r.spotify == nil {
		return fmt.Errorf("%w: Spotify service not initialized", shared.ErrServiceUnavailable)
	}

	r.logger.Infof("exporting spotify playlist %v in format %v", playlistID, format)

	export, err := r.spotify.ExportPlaylist(ctx, playlistID)
	if err != nil {
		if reauthed, authErr := r.handleSpotifyAuthError(ctx, err, cmd); reauthed {
			if authErr != nil {
				return authErr
			}
			export, err = r.spotify.ExportPlaylist(ctx, playlistID)
			if err != nil {
				return fmt.Errorf("%w: %v", shared.ErrAPIRequest, err)
			}
		} else {
			return fmt.Errorf("%w: %v", shared.ErrAPIRequest, err)
		}
	}

	// Handle format-specific export
	switch format {
	case "csv":
		return r.exportCSV(export, outputFile, save)
	case "markdown":
		return r.exportMarkdown(ctx, export, outputFile, save)
	case "txt":
		return r.exportText(export, outputFile, save)
	case "json":
		return r.exportJSON(export, outputFile, save, useJSON, pretty)
	default:
		return fmt.Errorf("unsupported format: %s (supported: json, csv, markdown, txt)", format)
	}
}

// exportCSV exports a playlist to CSV format with accompanying metadata JSON
func (r *Runner) exportCSV(export *models.PlaylistExport, filepath string, save bool) error {
	if filepath == "" && !save {
		return fmt.Errorf("CSV format requires --save flag or --output flag")
	}

	result, err := formatter.WriteCSVExport(export, filepath)
	if err != nil {
		return err
	}

	r.logger.Infof("playlist exported to CSV: %v and %v", result.TracksFile, result.MetadataFile)
	r.writePlain("✓ Playlist exported to:\n")
	r.writePlain("  Tracks: %s (%d tracks)\n", result.TracksFile, len(export.Tracks))
	r.writePlain("  Metadata: %s\n", result.MetadataFile)

	return nil
}

// exportMarkdown exports a playlist to Markdown format with cover image in a directory
func (r *Runner) exportMarkdown(ctx context.Context, export *models.PlaylistExport, outputDir string, save bool) error {
	if outputDir == "" && !save {
		return fmt.Errorf("markdown format requires --save flag or --output flag")
	}

	var imageURL string
	spotifySvc, ok := r.spotify.(*services.SpotifyService)
	if ok {
		spotifyPl, err := spotifySvc.Playlist(ctx, export.Playlist.ID)
		if err == nil && len(spotifyPl.Images) > 0 {
			imageURL = spotifyPl.Images[0].URL
		}
	}

	result, err := formatter.WriteMarkdownExport(export, outputDir, imageURL)
	if err != nil {
		return err
	}

	r.logger.Infof("playlist exported to Markdown directory: %v", result.Directory)
	r.writePlain("✓ Playlist exported to directory: %s\n", result.Directory)
	for _, file := range result.Files {
		r.writePlain("  - %s\n", file)
	}

	return nil
}

// exportText exports a playlist to plain text format
func (r *Runner) exportText(export *models.PlaylistExport, outputFile string, save bool) error {
	if outputFile == "" && !save {
		return fmt.Errorf("text format requires --save flag or --output flag")
	}

	filepath, err := formatter.WriteTextExport(export, outputFile)
	if err != nil {
		return err
	}

	r.logger.Infof("playlist exported to text: %v", filepath)
	r.writePlain("✓ Playlist exported to %s (%d tracks)\n", filepath, len(export.Tracks))

	return nil
}

// exportJSON exports a playlist to JSON format (legacy behavior)
func (r *Runner) exportJSON(export *models.PlaylistExport, outputFile string, save bool, useJSON bool, pretty bool) error {
	if outputFile == "" && (save || !useJSON) {
		outputFile = fmt.Sprintf("%s.json", export.Playlist.ID)
	}

	if outputFile != "" {
		data, err := shared.MarshalJSON(export, true)
		if err != nil {
			return fmt.Errorf("failed to marshal export: %w", err)
		}
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		r.logger.Infof("playlist exported to %v with %v tracks", outputFile, len(export.Tracks))

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

// doOAuth executes the OAuth2 authorization flow with a local HTTP server
func (r *Runner) doOAuth(config *shared.Config, oauthSrv services.OAuthService, prefix string) (*oauth2.Token, error) {
	state, err := shared.GenerateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state token: %w", err)
	}

	authURL := oauthSrv.GetAuthURL(state)
	oauthHandler := server.NewOAuthHandler(oauthSrv.GetOAuthConfig(), state)
	router := server.NewBasicRouter()
	router.Handler(oauthHandler)

	serverAddr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	httpServer := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	serverErrors := make(chan error, 1)
	go func() {
		r.logger.Infof("starting OAuth server for %s at %v", prefix, serverAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	time.Sleep(100 * time.Millisecond)

	r.writePlain("→ Opening browser for Spotify %s...\n", prefix)
	if err := shared.OpenBrowser(authURL); err != nil {
		r.logger.Warnf("failed to open browser automatically %v", err)
		r.writePlainln("⚠ Could not open browser automatically.")
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
		return nil, fmt.Errorf("server error: %w", err)
	case <-timeout.C:
		return nil, fmt.Errorf("%w: authorization timed out after 2 minutes", shared.ErrTimeout)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		r.logger.Warn("error shutting down server", "error", err)
	}

	if result.Error() != nil {
		return nil, fmt.Errorf("authorization failed: %w", result.Error())
	}

	if result.Token == nil {
		return nil, fmt.Errorf("no token received")
	}

	return result.Token, nil
}

// handleSpotifyAuthError checks if an error is a token expiration error and triggers reauthorization if needed.
func (r *Runner) handleSpotifyAuthError(ctx context.Context, err error, cmd *cli.Command) (bool, error) {
	if err == nil {
		return false, nil
	}

	if !errors.Is(err, shared.ErrTokenExpired) {
		return false, err
	}

	hasRefreshToken := r.config != nil && r.config.Credentials.Spotify.RefreshToken != ""
	if hasRefreshToken {
		r.writePlainln("⚠ Token expired. Automatic refresh failed, opening browser for re-authentication...\n")
	} else {
		r.writePlainln("⚠ No refresh token found. Opening browser for authentication...\n")
	}

	configPath := cmd.String("config")
	if configPath == "" {
		configPath = "config.toml"
	}

	config := r.config
	if config == nil {
		if _, statErr := os.Stat(configPath); statErr == nil {
			var loadErr error
			if config, loadErr = shared.LoadConfig(configPath); loadErr != nil {
				return true, fmt.Errorf("failed to load config: %w", loadErr)
			}
		} else {
			return true, fmt.Errorf("config file not found: %w", statErr)
		}
	}

	spotifyService, ok := r.spotify.(services.OAuthService)
	if !ok {
		return true, fmt.Errorf("spotify service does not support reauthorization")
	}

	updatedConfig, reauthErr := r.SpotifyReauth(ctx, configPath, config, spotifyService)
	if reauthErr != nil {
		return true, fmt.Errorf("reauthorization failed: %w", reauthErr)
	}

	if authErr := spotifyService.OAuthenticate(ctx, updatedConfig.Credentials.Spotify.Token()); authErr != nil {
		return true, fmt.Errorf("failed to authenticate with new tokens: %w", authErr)
	}

	r.config = updatedConfig
	r.writePlainln("✓ Successfully reauthenticated. Retrying operation...\n")

	return true, nil
}
