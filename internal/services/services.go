// package services defines interface Service for interacting with HTTP APIs
//
// Spotify, YouTube (via proxy)
package services

import (
	"context"

	"github.com/desertthunder/ytx/internal/models"
)

// Service defines the interface for music service providers (Spotify, YouTube Music) that can export and import playlists and songs.
type Service interface {
	// Authenticate performs the OAuth flow or API key authentication with the service.
	Authenticate(ctx context.Context, credentials map[string]string) error
	// GetPlaylists retrieves all playlists for the authenticated user.
	GetPlaylists(ctx context.Context) ([]models.Playlist, error)
	// GetPlaylist retrieves a specific playlist by ID.
	GetPlaylist(ctx context.Context, playlistID string) (*models.Playlist, error)
	// ExportPlaylist exports a playlist with all its tracks.
	ExportPlaylist(ctx context.Context, playlistID string) (*models.PlaylistExport, error)
	// ImportPlaylist imports a playlist into the service, by creating a new playlist and populates it with the provided tracks.
	ImportPlaylist(ctx context.Context, playlist *models.PlaylistExport) (*models.Playlist, error)
	// SearchTrack searches for a track by title and artist and returns the best match or an error if no match is found.
	SearchTrack(ctx context.Context, title, artist string) (*models.Track, error)
	// Name returns the name of the service (e.g., "Spotify", "YouTube Music")
	Name() string
}
