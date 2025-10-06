// package services defines interface Service for interacting with HTTP APIs
//
// Spotify, YouTube (via proxy)
package services

import (
	"context"
)

// Service defines the interface for music service providers (Spotify, YouTube Music) that can export and import playlists and songs.
type Service interface {
	// Authenticate performs OAuth or API key authentication with the service.
	// Returns an error if authentication fails.
	Authenticate(ctx context.Context, credentials map[string]string) error

	// GetPlaylists retrieves all playlists for the authenticated user.
	GetPlaylists(ctx context.Context) ([]Playlist, error)

	// GetPlaylist retrieves a specific playlist by ID.
	GetPlaylist(ctx context.Context, playlistID string) (*Playlist, error)

	// ExportPlaylist exports a playlist with all its tracks.
	ExportPlaylist(ctx context.Context, playlistID string) (*PlaylistExport, error)

	// ImportPlaylist imports a playlist into the service.
	// Creates a new playlist and populates it with the provided tracks.
	ImportPlaylist(ctx context.Context, playlist *PlaylistExport) (*Playlist, error)

	// SearchTrack searches for a track by title and artist.
	// Returns the best match or an error if no match is found.
	SearchTrack(ctx context.Context, title, artist string) (*Track, error)

	// Name returns the name of the service (e.g., "Spotify", "YouTube Music")
	Name() string
}

// Playlist represents a music playlist from any service
type Playlist struct {
	ID          string
	Name        string
	Description string
	TrackCount  int
	Public      bool
}

// PlaylistExport represents a playlist with all its tracks for migration
type PlaylistExport struct {
	Playlist Playlist
	Tracks   []Track
}

// Track represents a music track from any service
type Track struct {
	ID       string
	Title    string
	Artist   string
	Album    string
	Duration int    // Duration in seconds
	ISRC     string // International Standard Recording Code for matching
}
