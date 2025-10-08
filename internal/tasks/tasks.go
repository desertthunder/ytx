// package tasks implements playlist transfer operations between music services.
//
// The core abstraction is SyncEngine, which orchestrates playlist transfers, comparisons, and data dumps.
// Operations emit progress updates via channels for non-blocking status reporting to CLI/UI layers.
package tasks

import (
	"context"
	"fmt"

	"github.com/desertthunder/ytx/internal/models"
	"github.com/desertthunder/ytx/internal/services"
	"github.com/desertthunder/ytx/internal/shared"
)

// TrackMatchResult represents the result of attempting to match a single track.
type TrackMatchResult struct {
	Original models.Track  // Original track from source
	Matched  *models.Track // Matched track (nil if not found)
	Error    error         // Error if match failed
}

// TransferRunResult contains all data from a full transfer operation.
type TransferRunResult struct {
	SourcePlaylist  *models.PlaylistExport // Source playlist with tracks
	DestPlaylist    *models.Playlist       // Created destination playlist
	TrackMatches    []TrackMatchResult     // Individual track match results
	SuccessCount    int                    // Number of successfully matched tracks
	FailedCount     int                    // Number of failed matches
	TotalTracks     int                    // Total tracks processed
	MatchPercentage float64                // Success rate as percentage
}

// ComparisonResult contains track comparison details between two playlists.
type ComparisonResult struct {
	SourcePlaylist *models.PlaylistExport // Source playlist
	DestPlaylist   *models.PlaylistExport // Destination playlist
	MatchedCount   int                    // Tracks found in both
	MissingInDest  []models.Track         // Tracks in source but not in dest
	ExtraInDest    []models.Track         // Tracks in dest but not in source
}

// TransferDiffResult contains the results of comparing two playlists.
type TransferDiffResult struct {
	Comparison ComparisonResult
}

// EndpointResult represents the result of fetching data from a single API endpoint.
type EndpointResult struct {
	Endpoint string
	Data     any
	Error    error
}

// DumpResult contains all data fetched from the API proxy.
type DumpResult struct {
	Health         any              // Health status
	Playlists      any              // Library playlists
	Songs          any              // Library songs
	Albums         any              // Library albums
	Artists        any              // Library artists
	LikedSongs     any              // Liked songs
	History        any              // Listening history
	UploadedSongs  any              // Uploaded songs
	UploadedAlbums any              // Uploaded albums
	Errors         []EndpointResult // Failed endpoint fetches
}

type DumpData struct {
	Health         any   `json:"health"`
	Playlists      any   `json:"playlists,omitempty"`
	Songs          any   `json:"songs,omitempty"`
	Albums         any   `json:"albums,omitempty"`
	Artists        any   `json:"artists,omitempty"`
	LikedSongs     any   `json:"liked_songs,omitempty"`
	History        any   `json:"history,omitempty"`
	UploadedSongs  any   `json:"uploaded_songs,omitempty"`
	UploadedAlbums any   `json:"uploaded_albums,omitempty"`
	Errors         []any `json:"errors,omitempty"`
}

type endpointOperation struct {
	name    string
	path    string
	target  *any
	phase   Phase
	message string
}

// SyncEngine defines operations for syncing playlists between services.
type SyncEngine interface {
	// Run performs a full Spotify → YouTube Music sync by fetching source playlist, searches for tracks, creates destination playlist.
	Run(ctx context.Context, progress chan<- ProgressUpdate, sourceIDOrName, destName string) (*TransferRunResult, error)

	// Diff compares two playlists across services by identifying matched tracks, missing tracks, and extra tracks.
	Diff(ctx context.Context, progress chan<- ProgressUpdate, sourceSvc, destSvc services.Service, sourceID, destID string) (*TransferDiffResult, error)

	// Dump fetches all data from the API proxy by retrieving health, playlists, songs, albums, artists, etc.
	Dump(ctx context.Context, progress chan<- ProgressUpdate) (*DumpResult, error)
}

// PlaylistEngine implements SyncEngine for playlist operations.
// Contains dependencies on music services and API client.
type PlaylistEngine struct {
	spotify services.Service
	youtube services.Service
	api     APIClient
}

// APIClient defines the interface for making API requests to the proxy.
// This abstraction allows for easier testing and decoupling from concrete implementation.
type APIClient interface {
	Get(ctx context.Context, path string) (*services.APIResponse, error)
}

// NewPlaylistEngine creates a new PlaylistEngine with the provided services.
func NewPlaylistEngine(spotify, youtube services.Service, api APIClient) *PlaylistEngine {
	return &PlaylistEngine{
		spotify: spotify,
		youtube: youtube,
		api:     api,
	}
}

// sendProgress sends a progress update through the channel without blocking.
// Uses select with default to ensure progress reporting never blocks execution.
func (e *PlaylistEngine) sendProgress(progress chan<- ProgressUpdate, update ProgressUpdate) {
	if progress == nil {
		return
	}
	select {
	case progress <- update:
		// Sent successfully
	default:
		// Channel full or closed, skip this update
	}
}

// Run performs a full Spotify → YouTube Music playlist sync.
func (e *PlaylistEngine) Run(ctx context.Context, srcID string, progress chan<- ProgressUpdate) (*TransferRunResult, error) {
	if e.spotify == nil {
		return nil, fmt.Errorf("%w: Spotify service not initialized", shared.ErrServiceUnavailable)
	}
	if e.youtube == nil {
		return nil, fmt.Errorf("%w: YouTube Music service not initialized", shared.ErrServiceUnavailable)
	}

	result := &TransferRunResult{}

	e.sendProgress(progress, fetchingSourceUpdate(1, 1))

	srcPlaylist, err := e.spotify.ExportPlaylist(ctx, srcID)
	if err != nil {
		playlists, playlistsErr := e.spotify.GetPlaylists(ctx)
		if playlistsErr != nil {
			return nil, fmt.Errorf("%w: failed to get playlists: %v", shared.ErrAPIRequest, playlistsErr)
		}

		var matchedID string
		for _, pl := range playlists {
			if pl.Name == srcID {
				matchedID = pl.ID
				break
			}
		}

		if matchedID == "" {
			return nil, fmt.Errorf("%w: no playlist found with name '%s'", shared.ErrPlaylistNotFound, srcID)
		}

		srcPlaylist, err = e.spotify.ExportPlaylist(ctx, matchedID)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to export playlist: %v", shared.ErrAPIRequest, err)
		}
	}

	total := len(srcPlaylist.Tracks)
	result.SourcePlaylist = srcPlaylist
	result.TotalTracks = total

	e.sendProgress(progress, foundPlaylistUpdate(1, 1, srcPlaylist))
	e.sendProgress(progress, searchTracksUpdate(0, total, nil))

	matches := make([]TrackMatchResult, total)
	successCount := 0

	for i, track := range srcPlaylist.Tracks {
		e.sendProgress(progress, searchTracksUpdate(i+1, total, &track))

		ytTrack, err := e.youtube.SearchTrack(ctx, track.Title, track.Artist)
		matches[i] = TrackMatchResult{
			Original: track,
			Matched:  ytTrack,
			Error:    err,
		}

		if err == nil {
			successCount++
		}
	}

	result.TrackMatches = matches
	result.SuccessCount = successCount
	result.FailedCount = total - successCount
	if result.TotalTracks > 0 {
		result.MatchPercentage = float64(successCount) / float64(result.TotalTracks) * 100
	}

	if successCount == 0 {
		return result, fmt.Errorf("no tracks were matched - cannot create empty playlist")
	}

	e.sendProgress(progress, createDestinationUpdate(1, 1))

	matchedTracks := make([]models.Track, 0, successCount)
	for _, match := range matches {
		if match.Matched != nil {
			matchedTracks = append(matchedTracks, *match.Matched)
		}
	}
	destExport := &models.PlaylistExport{
		Playlist: models.Playlist{
			Name:        srcPlaylist.Playlist.Name,
			Description: fmt.Sprintf("Migrated from Spotify: %s", srcPlaylist.Playlist.Name),
			Public:      false,
		},
		Tracks: matchedTracks,
	}

	importedPl, err := e.youtube.ImportPlaylist(ctx, destExport)
	if err != nil {
		return result, fmt.Errorf("%w: failed to create playlist: %v", shared.ErrAPIRequest, err)
	}

	result.DestPlaylist = importedPl
	e.sendProgress(progress, createPlaylistUpdate(1, 1, importedPl))
	return result, nil
}

// Diff compares two playlists and identifies differences.
func (e *PlaylistEngine) Diff(ctx context.Context, sourceSvc, destSvc services.Service, sourceID, destID string, progress chan<- ProgressUpdate) (*TransferDiffResult, error) {
	if sourceSvc == nil || destSvc == nil {
		return nil, fmt.Errorf("%w: service not initialized", shared.ErrServiceUnavailable)
	}

	result := &TransferDiffResult{}

	e.sendProgress(progress, fetchSourceUpdate(1, 2, sourceSvc.Name()))
	sourceExport, err := sourceSvc.ExportPlaylist(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to export source playlist: %v", shared.ErrPlaylistNotFound, err)
	}

	e.sendProgress(progress, fetchDestUpdate(2, 2, destSvc.Name()))
	destExport, err := destSvc.ExportPlaylist(ctx, destID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to export destination playlist: %v", shared.ErrPlaylistNotFound, err)
	}

	result.Comparison.SourcePlaylist = sourceExport
	result.Comparison.DestPlaylist = destExport

	e.sendProgress(progress, buildDestMapUpdate(1, 2))
	destTrackMap := make(map[string]models.Track)
	destISRCMap := make(map[string]models.Track)

	for _, track := range destExport.Tracks {
		normalizedKey := shared.NormalizeTrackKey(track.Title, track.Artist)
		destTrackMap[normalizedKey] = track
		if track.ISRC != "" {
			destISRCMap[track.ISRC] = track
		}
	}

	e.sendProgress(progress, missingTrackUpdate(2, 2))
	var missingInDest []models.Track
	matchedCount := 0

	for _, srcTrack := range sourceExport.Tracks {
		matched := false

		if srcTrack.ISRC != "" {
			if _, found := destISRCMap[srcTrack.ISRC]; found {
				matched = true
			}
		}

		if !matched {
			normalizedKey := shared.NormalizeTrackKey(srcTrack.Title, srcTrack.Artist)
			if _, found := destTrackMap[normalizedKey]; found {
				matched = true
			}
		}

		if matched {
			matchedCount++
		} else {
			missingInDest = append(missingInDest, srcTrack)
		}
	}

	sourceTrackMap := make(map[string]models.Track)
	sourceISRCMap := make(map[string]models.Track)

	for _, track := range sourceExport.Tracks {
		normalizedKey := shared.NormalizeTrackKey(track.Title, track.Artist)
		sourceTrackMap[normalizedKey] = track
		if track.ISRC != "" {
			sourceISRCMap[track.ISRC] = track
		}
	}

	var extraInDest []models.Track
	for _, destTrack := range destExport.Tracks {
		matched := false

		if destTrack.ISRC != "" {
			if _, found := sourceISRCMap[destTrack.ISRC]; found {
				matched = true
			}
		}

		if !matched {
			normalizedKey := shared.NormalizeTrackKey(destTrack.Title, destTrack.Artist)
			if _, found := sourceTrackMap[normalizedKey]; found {
				matched = true
			}
		}

		if !matched {
			extraInDest = append(extraInDest, destTrack)
		}
	}

	result.Comparison.MatchedCount = matchedCount
	result.Comparison.MissingInDest = missingInDest
	result.Comparison.ExtraInDest = extraInDest

	return result, nil
}

// Dump fetches all data from the API proxy.
func (e *PlaylistEngine) Dump(ctx context.Context, progress chan<- ProgressUpdate) (*DumpResult, error) {
	if e.api == nil {
		return nil, fmt.Errorf("%w: API client not initialized", shared.ErrServiceUnavailable)
	}

	result := &DumpResult{
		Errors: []EndpointResult{},
	}

	endpoints := []endpointOperation{
		{name: "health", path: "/health", target: &result.Health, phase: FetchHealth, message: "Fetching health status..."},
		{name: "playlists", path: "/api/library/playlists", target: &result.Playlists, phase: FetchPlaylists, message: "Fetching playlists..."},
		{name: "songs", path: "/api/library/songs", target: &result.Songs, phase: FetchSongs, message: "Fetching songs..."},
		{name: "albums", path: "/api/library/albums", target: &result.Albums, phase: FetchAlbums, message: "Fetching albums..."},
		{name: "artists", path: "/api/library/artists", target: &result.Artists, phase: FetchArtists, message: "Fetching artists..."},
		{name: "liked_songs", path: "/api/library/liked-songs", target: &result.LikedSongs, phase: FetchLiked, message: "Fetching liked songs..."},
		{name: "history", path: "/api/library/history", target: &result.History, phase: FetchHistory, message: "Fetching history..."},
		{name: "uploaded_songs", path: "/api/uploads/songs", target: &result.UploadedSongs, phase: FetchUploads, message: "Fetching uploaded songs..."},
		{name: "uploaded_albums", path: "/api/uploads/albums", target: &result.UploadedAlbums, phase: FetchUploads, message: "Fetching uploaded albums..."},
	}

	totalSteps := len(endpoints)

	for i, endpoint := range endpoints {
		e.sendProgress(progress, operationUpdate(endpoint, i+1, totalSteps))

		resp, err := e.api.Get(ctx, endpoint.path)
		if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			} else {
				errMsg = fmt.Sprintf("status %d", resp.StatusCode)
			}
			result.Errors = append(result.Errors, EndpointResult{
				Endpoint: endpoint.path,
				Error:    fmt.Errorf("%s", errMsg),
			})
		} else {
			*endpoint.target = resp.JSONData
		}
	}

	return result, nil
}
