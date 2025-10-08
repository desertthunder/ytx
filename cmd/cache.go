package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// CachePlaylistSpotify caches a Spotify playlist and its tracks to the database.
//
// Tracks are automatically cached via the transfer engine.
func (r *Runner) CachePlaylistSpotify(ctx context.Context, cmd *cli.Command) error {
	playlistID := cmd.String("id")
	if playlistID == "" {
		return fmt.Errorf("playlist ID is required")
	}

	r.logger.Infof("caching Spotify playlist: %s", playlistID)

	if r.spotify == nil {
		return fmt.Errorf("uninitialized Spotify service")
	}

	playlist, err := r.spotify.ExportPlaylist(ctx, playlistID)
	if err != nil {
		return fmt.Errorf("failed to export playlist: %w", err)
	}

	r.logger.Infof("fetched playlist: %s (%d tracks)", playlist.Playlist.Name, len(playlist.Tracks))

	r.writePlainln("✓ Playlist fetched: %s", playlist.Playlist.Name)
	r.writePlainln("  Tracks: %d", len(playlist.Tracks))
	r.writePlainln("Note: Tracks are automatically cached during 'ytx transfer run' operations.")
	r.writePlainln("Playlist metadata caching requires user context (not yet implemented).")

	return nil
}

// CachePlaylistYouTube caches a YouTube Music playlist and its tracks to the database.
func (r *Runner) CachePlaylistYouTube(ctx context.Context, cmd *cli.Command) error {
	playlistID := cmd.String("id")
	if playlistID == "" {
		return fmt.Errorf("playlist ID is required")
	}

	r.logger.Infof("caching YouTube Music playlist: %s", playlistID)

	if r.youtube == nil {
		return fmt.Errorf("YouTube Music service not initialized")
	}

	playlist, err := r.youtube.ExportPlaylist(ctx, playlistID)
	if err != nil {
		return fmt.Errorf("failed to export playlist: %w", err)
	}

	r.logger.Infof("fetched playlist: %s (%d tracks)", playlist.Playlist.Name, len(playlist.Tracks))

	r.writePlainln("✓ Playlist fetched: %s", playlist.Playlist.Name)
	r.writePlainln("  Tracks: %d", len(playlist.Tracks))
	r.writePlainln("Note: Tracks are automatically cached during 'ytx transfer run' operations.")
	r.writePlainln("Playlist metadata caching requires user context (not yet implemented).")

	return nil
}

// cacheCommand handles opt-in playlist and track caching
func cacheCommand(r *Runner) *cli.Command {
	return &cli.Command{
		Name:  "cache",
		Usage: "Cache playlists and tracks locally",
		Commands: []*cli.Command{
			{
				Name:  "playlist",
				Usage: "Cache a playlist (Spotify or YouTube Music)",
				Commands: []*cli.Command{
					{
						Name:  "spotify",
						Usage: "Cache a Spotify playlist",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "id",
								Usage:    "Playlist ID to cache",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "config",
								Aliases: []string{"c"},
								Usage:   "Path to configuration file",
								Value:   "config.toml",
							},
						},
						Action: r.CachePlaylistSpotify,
					},
					{
						Name:    "youtube",
						Aliases: []string{"yt", "ytmusic"},
						Usage:   "Cache a YouTube Music playlist",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "id",
								Usage:    "Playlist ID to cache",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "config",
								Aliases: []string{"c"},
								Usage:   "Path to configuration file",
								Value:   "config.toml",
							},
						},
						Action: r.CachePlaylistYouTube,
					},
				},
			},
		},
	}
}
