package tasks

import (
	"fmt"

	"github.com/desertthunder/ytx/internal/models"
)

// ProgressUpdate represents a progress event during a long-running operation.
//
// Used to send real-time updates to the CLI or UI layer for display.
type ProgressUpdate struct {
	Phase   Phase  // Operation phase
	Step    int    // Current step number within phase
	Total   int    // Total steps in this phase
	Message string // Human-readable message for display
	Data    any    // Optional phase-specific data for advanced UIs
}

// Operation phase enumeration
type Phase int

const (
	FetchSource Phase = iota
	FetchDest
	Compare
	FetchHealth
	FetchPlaylists
	FetchSongs
	FetchAlbums
	FetchArtists
	FetchLiked
	FetchHistory
	FetchUploads
	CreatePlaylist
	SearchTracks
	ExportPlaylist
)

func (p Phase) String() string {
	switch p {
	case FetchSource:
		return "fetch_source"
	case FetchDest:
		return "fetch_dest"
	case Compare:
		return "compare"
	case FetchHealth:
		return "fetch_health"
	case FetchPlaylists:
		return "fetch_playlists"
	case FetchSongs:
		return "fetch_songs"
	case FetchAlbums:
		return "fetch_albums"
	case FetchArtists:
		return "fetch_artists"
	case FetchLiked:
		return "fetch_liked"
	case FetchHistory:
		return "fetch_history"
	case FetchUploads:
		return "fetch_uploads"
	case CreatePlaylist:
		return "create_playlist"
	case SearchTracks:
		return "search_tracks"
	case ExportPlaylist:
		return "export_playlist"
	default:
		return ""
	}
}

func fetchingSourceUpdate(step, total int) ProgressUpdate {
	return ProgressUpdate{
		Phase:   FetchSource,
		Step:    step,
		Total:   total,
		Message: "Fetching source playlist from Spotify...",
	}
}

func fetchSourceUpdate(step, total int, name string) ProgressUpdate {
	return ProgressUpdate{
		Phase:   FetchSource,
		Step:    step,
		Total:   total,
		Message: fmt.Sprintf("Fetching source playlist (%s)...", name),
	}
}

func fetchDestUpdate(step, total int, name string) ProgressUpdate {
	return ProgressUpdate{
		Phase:   FetchDest,
		Step:    step,
		Total:   total,
		Message: fmt.Sprintf("Fetching destination playlist (%s)...", name),
	}
}

func buildDestMapUpdate(step, total int) ProgressUpdate {
	return ProgressUpdate{
		Phase:   Compare,
		Step:    step,
		Total:   total,
		Message: "Building track comparison maps...",
	}
}

func missingTrackUpdate(step, total int) ProgressUpdate {
	return ProgressUpdate{
		Phase:   Compare,
		Step:    step,
		Total:   total,
		Message: "Comparing tracks...",
	}
}

func operationUpdate(endpoint endpointOperation, step int, total int) ProgressUpdate {
	return ProgressUpdate{
		Phase:   endpoint.phase,
		Step:    step,
		Total:   total,
		Message: endpoint.message,
	}
}

func createPlaylistUpdate(step, total int, pl *models.Playlist) ProgressUpdate {
	return ProgressUpdate{
		Phase:   CreatePlaylist,
		Step:    step,
		Total:   total,
		Message: fmt.Sprintf("Playlist created: %s (ID: %s)", pl.Name, pl.ID),
		Data:    pl,
	}
}

func createDestinationUpdate(step, total int) ProgressUpdate {
	return ProgressUpdate{
		Phase:   CreatePlaylist,
		Step:    step,
		Total:   total,
		Message: "Creating playlist on YouTube Music...",
	}
}

func searchTracksUpdate(step, total int, tr *models.Track) ProgressUpdate {
	if tr == nil {
		return ProgressUpdate{
			Phase:   SearchTracks,
			Step:    step,
			Total:   total,
			Message: "Searching for tracks on YouTube Music...",
		}
	}
	return ProgressUpdate{
		Phase:   SearchTracks,
		Step:    step,
		Total:   total,
		Message: fmt.Sprintf("[%d/%d] %s - %s", step, total, tr.Artist, tr.Title),
	}
}

func foundPlaylistUpdate(step, total int, export *models.PlaylistExport) ProgressUpdate {
	return ProgressUpdate{
		Phase:   FetchSource,
		Step:    step,
		Total:   total,
		Message: fmt.Sprintf("Found playlist: %s (%d tracks)", export.Playlist.Name, total),
		Data:    export,
	}
}

func exportingPlaylistUpdate(step, total int, name string) ProgressUpdate {
	return ProgressUpdate{
		Phase:   ExportPlaylist,
		Step:    step,
		Total:   total,
		Message: fmt.Sprintf("[%d/%d] Exporting: %s...", step, total, name),
	}
}

func exportCompletedUpdate(step, total int, name string, filesCount int) ProgressUpdate {
	return ProgressUpdate{
		Phase:   ExportPlaylist,
		Step:    step,
		Total:   total,
		Message: fmt.Sprintf("[%d/%d] ✓ %s (%d files)", step, total, name, filesCount),
	}
}

func exportFailedUpdate(step, total int, name string, err error) ProgressUpdate {
	return ProgressUpdate{
		Phase:   ExportPlaylist,
		Step:    step,
		Total:   total,
		Message: fmt.Sprintf("[%d/%d] ✗ %s: %v", step, total, name, err),
	}
}
