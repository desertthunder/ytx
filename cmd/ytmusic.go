package main

import (
	"context"
	"fmt"

	"github.com/desertthunder/song-migrations/internal/services"
	"github.com/desertthunder/song-migrations/internal/shared"
	"github.com/urfave/cli/v3"
)

// YTMusicSearch searches YouTube Music for tracks.
func (r *Runner) YTMusicSearch(ctx context.Context, cmd *cli.Command) error {
	if r.youtube == nil {
		return fmt.Errorf("%w: YouTube Music service not initialized", shared.ErrServiceUnavailable)
	}

	query := cmd.StringArg("query")
	useJSON := cmd.Bool("json")
	pretty := cmd.Bool("pretty")

	r.logger.Info("searching youtube music", "query", query)

	track, err := r.youtube.SearchTrack(ctx, query, "")
	if err != nil {
		return fmt.Errorf("%w: %v", shared.ErrAPIRequest, err)
	}

	if useJSON {
		return r.writeJSON(track, pretty)
	}

	r.writePlain("Found track:\n\n")
	r.writePlain("Title: %s\n", track.Title)
	if track.Artist != "" {
		r.writePlain("Artist: %s\n", track.Artist)
	}
	if track.Album != "" {
		r.writePlain("Album: %s\n", track.Album)
	}
	r.writePlain("ID: %s\n", track.ID)
	if track.Duration > 0 {
		minutes := track.Duration / 60
		seconds := track.Duration % 60
		r.writePlain("Duration: %d:%02d\n", minutes, seconds)
	}
	if track.ISRC != "" {
		r.writePlain("ISRC: %s\n", track.ISRC)
	}

	return nil
}

// YTMusicCreate creates a new playlist on YouTube Music.
func (r *Runner) YTMusicCreate(ctx context.Context, cmd *cli.Command) error {
	if r.youtube == nil {
		return fmt.Errorf("%w: YouTube Music service not initialized", shared.ErrServiceUnavailable)
	}

	name := cmd.StringArg("name")
	description := cmd.String("description")
	private := cmd.Bool("private")

	r.logger.Info("creating youtube music playlist", "name", name, "private", private)

	export := &services.PlaylistExport{
		Playlist: services.Playlist{
			Name:        name,
			Description: description,
			Public:      !private,
		},
		Tracks: []services.Track{},
	}

	playlist, err := r.youtube.ImportPlaylist(ctx, export)
	if err != nil {
		return fmt.Errorf("%w: %v", shared.ErrAPIRequest, err)
	}

	r.logger.Info("playlist created", "id", playlist.ID, "name", playlist.Name)
	r.writePlain("✓ Playlist created successfully\n")
	r.writePlain("Name: %s\n", playlist.Name)
	r.writePlain("ID: %s\n", playlist.ID)
	if private {
		r.writePlain("Visibility: Private\n")
	} else {
		r.writePlain("Visibility: Public\n")
	}

	return nil
}

// YTMusicAdd adds tracks to an existing YouTube Music playlist.
func (r *Runner) YTMusicAdd(ctx context.Context, cmd *cli.Command) error {
	if r.youtube == nil {
		return fmt.Errorf("%w: YouTube Music service not initialized", shared.ErrServiceUnavailable)
	}

	playlistID := cmd.String("playlist-id")
	trackQuery := cmd.String("track")

	if playlistID == "" {
		return fmt.Errorf("%w: --playlist-id flag is required", shared.ErrMissingArgument)
	}
	if trackQuery == "" {
		return fmt.Errorf("%w: --track flag is required", shared.ErrMissingArgument)
	}

	r.logger.Info("adding track to playlist", "playlist_id", playlistID, "track", trackQuery)

	track, err := r.youtube.SearchTrack(ctx, trackQuery, "")
	if err != nil {
		return fmt.Errorf("%w: failed to find track: %v", shared.ErrTrackNotFound, err)
	}

	r.logger.Info("found track", "id", track.ID, "title", track.Title, "artist", track.Artist)

	playlist, err := r.youtube.GetPlaylist(ctx, playlistID)
	if err != nil {
		return fmt.Errorf("%w: %v", shared.ErrPlaylistNotFound, err)
	}

	// FIXME: implement AddTrack API endpoint
	export := &services.PlaylistExport{
		Playlist: *playlist,
		Tracks:   []services.Track{*track},
	}

	_, err = r.youtube.ImportPlaylist(ctx, export)
	if err != nil {
		return fmt.Errorf("%w: failed to add track: %v", shared.ErrAPIRequest, err)
	}

	r.logger.Info("track added successfully")
	r.writePlain("✓ Track added to playlist\n")
	r.writePlain("Playlist: %s (ID: %s)\n", playlist.Name, playlistID)
	r.writePlain("Added: %s - %s\n", track.Artist, track.Title)

	return nil
}
