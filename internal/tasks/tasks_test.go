package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/desertthunder/song-migrations/internal/services"
	"github.com/desertthunder/song-migrations/internal/shared"
)

type mockService struct {
	name            string
	playlists       []services.Playlist
	playlistExports map[string]*services.PlaylistExport
	searchResults   map[string]*services.Track
	importResult    *services.Playlist
	authenticateErr error
	getPlaylistsErr error
	getPlaylistErr  error
	exportErr       error
	exportCallCount int
	exportErrOnce   bool // If true, only fail first export call
	importErr       error
	searchErr       error
}

func (m *mockService) Name() string {
	return m.name
}

func (m *mockService) Authenticate(ctx context.Context, credentials map[string]string) error {
	return m.authenticateErr
}

func (m *mockService) GetPlaylists(ctx context.Context) ([]services.Playlist, error) {
	if m.getPlaylistsErr != nil {
		return nil, m.getPlaylistsErr
	}
	return m.playlists, nil
}

func (m *mockService) GetPlaylist(ctx context.Context, playlistID string) (*services.Playlist, error) {
	if m.getPlaylistErr != nil {
		return nil, m.getPlaylistErr
	}
	if export, ok := m.playlistExports[playlistID]; ok {
		return &export.Playlist, nil
	}
	return nil, fmt.Errorf("playlist not found")
}

func (m *mockService) ExportPlaylist(ctx context.Context, playlistID string) (*services.PlaylistExport, error) {
	m.exportCallCount++
	if m.exportErr != nil {
		if m.exportErrOnce && m.exportCallCount > 1 {
			// Allow subsequent calls to succeed
		} else {
			return nil, m.exportErr
		}
	}
	if export, ok := m.playlistExports[playlistID]; ok {
		return export, nil
	}
	return nil, fmt.Errorf("playlist not found")
}

func (m *mockService) ImportPlaylist(ctx context.Context, playlist *services.PlaylistExport) (*services.Playlist, error) {
	if m.importErr != nil {
		return nil, m.importErr
	}
	return m.importResult, nil
}

func (m *mockService) SearchTrack(ctx context.Context, title, artist string) (*services.Track, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	key := title + "|" + artist
	if track, ok := m.searchResults[key]; ok {
		return track, nil
	}
	return nil, fmt.Errorf("track not found")
}

// Mock API client for testing
type mockAPIClient struct {
	responses map[string]*services.APIResponse
	getErr    error
}

func (m *mockAPIClient) Get(ctx context.Context, path string) (*services.APIResponse, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if resp, ok := m.responses[path]; ok {
		return resp, nil
	}
	return &services.APIResponse{
		StatusCode: 404,
		Body:       []byte("not found"),
	}, nil
}

func TestPlaylistEngine_Run(t *testing.T) {
	tests := []struct {
		name           string
		sourceID       string
		destName       string
		spotifyService *mockService
		youtubeService *mockService
		wantErr        bool
		wantSuccess    int
		wantFailed     int
	}{
		{
			name:     "successful transfer by ID",
			sourceID: "playlist123",
			destName: "My YouTube Playlist",
			spotifyService: &mockService{
				name: "Spotify",
				playlistExports: map[string]*services.PlaylistExport{
					"playlist123": {
						Playlist: services.Playlist{
							ID:   "playlist123",
							Name: "My Spotify Playlist",
						},
						Tracks: []services.Track{
							{ID: "track1", Title: "Song 1", Artist: "Artist 1"},
							{ID: "track2", Title: "Song 2", Artist: "Artist 2"},
						},
					},
				},
			},
			youtubeService: &mockService{
				name: "YouTube Music",
				searchResults: map[string]*services.Track{
					"Song 1|Artist 1": {ID: "yt1", Title: "Song 1", Artist: "Artist 1"},
					"Song 2|Artist 2": {ID: "yt2", Title: "Song 2", Artist: "Artist 2"},
				},
				importResult: &services.Playlist{
					ID:         "yt_playlist",
					Name:       "My YouTube Playlist",
					TrackCount: 2,
				},
			},
			wantErr:     false,
			wantSuccess: 2,
			wantFailed:  0,
		},
		{
			name:     "successful transfer by name",
			sourceID: "My Spotify Playlist",
			destName: "My YouTube Playlist",
			spotifyService: &mockService{
				name: "Spotify",
				playlists: []services.Playlist{
					{ID: "playlist123", Name: "My Spotify Playlist"},
				},
				playlistExports: map[string]*services.PlaylistExport{
					"playlist123": {
						Playlist: services.Playlist{
							ID:   "playlist123",
							Name: "My Spotify Playlist",
						},
						Tracks: []services.Track{
							{ID: "track1", Title: "Song 1", Artist: "Artist 1"},
						},
					},
				},
				exportErr:     fmt.Errorf("not found"), // First export by ID fails
				exportErrOnce: true,                    // Only fail first call
			},
			youtubeService: &mockService{
				name: "YouTube Music",
				searchResults: map[string]*services.Track{
					"Song 1|Artist 1": {ID: "yt1", Title: "Song 1", Artist: "Artist 1"},
				},
				importResult: &services.Playlist{
					ID:         "yt_playlist",
					Name:       "My YouTube Playlist",
					TrackCount: 1,
				},
			},
			wantErr:     false,
			wantSuccess: 1,
			wantFailed:  0,
		},
		{
			name:     "partial success with some tracks not found",
			sourceID: "playlist123",
			destName: "My YouTube Playlist",
			spotifyService: &mockService{
				name: "Spotify",
				playlistExports: map[string]*services.PlaylistExport{
					"playlist123": {
						Playlist: services.Playlist{
							ID:   "playlist123",
							Name: "My Spotify Playlist",
						},
						Tracks: []services.Track{
							{ID: "track1", Title: "Song 1", Artist: "Artist 1"},
							{ID: "track2", Title: "Song 2", Artist: "Artist 2"},
							{ID: "track3", Title: "Song 3", Artist: "Artist 3"},
						},
					},
				},
			},
			youtubeService: &mockService{
				name: "YouTube Music",
				searchResults: map[string]*services.Track{
					"Song 1|Artist 1": {ID: "yt1", Title: "Song 1", Artist: "Artist 1"},
					// Song 2 not found
					"Song 3|Artist 3": {ID: "yt3", Title: "Song 3", Artist: "Artist 3"},
				},
				importResult: &services.Playlist{
					ID:         "yt_playlist",
					Name:       "My YouTube Playlist",
					TrackCount: 2,
				},
			},
			wantErr:     false,
			wantSuccess: 2,
			wantFailed:  1,
		},
		{
			name:     "no tracks matched - should error",
			sourceID: "playlist123",
			destName: "My YouTube Playlist",
			spotifyService: &mockService{
				name: "Spotify",
				playlistExports: map[string]*services.PlaylistExport{
					"playlist123": {
						Playlist: services.Playlist{
							ID:   "playlist123",
							Name: "My Spotify Playlist",
						},
						Tracks: []services.Track{
							{ID: "track1", Title: "Song 1", Artist: "Artist 1"},
						},
					},
				},
			},
			youtubeService: &mockService{
				name:          "YouTube Music",
				searchResults: map[string]*services.Track{},
			},
			wantErr:     true,
			wantSuccess: 0,
			wantFailed:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewPlaylistEngine(tt.spotifyService, tt.youtubeService, nil)

			progressCh := make(chan ProgressUpdate, 100)
			go func() {
				for range progressCh {
					// Drain progress channel
				}
			}()

			result, err := engine.Run(context.Background(), tt.sourceID, tt.destName, progressCh)
			close(progressCh)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.SuccessCount != tt.wantSuccess {
					t.Errorf("Run() successCount = %v, want %v", result.SuccessCount, tt.wantSuccess)
				}
				if result.FailedCount != tt.wantFailed {
					t.Errorf("Run() failedCount = %v, want %v", result.FailedCount, tt.wantFailed)
				}
			}
		})
	}
}

func TestPlaylistEngine_Run_ServiceErrors(t *testing.T) {
	t.Run("spotify service not initialized", func(t *testing.T) {
		engine := NewPlaylistEngine(nil, &mockService{}, nil)
		progressCh := make(chan ProgressUpdate, 10)

		_, err := engine.Run(context.Background(), "playlist123", "dest", progressCh)
		close(progressCh)

		if err == nil {
			t.Error("Run() expected error for nil spotify service")
		}
		if err != nil && !errors.Is(err, shared.ErrServiceUnavailable) {
			if !strings.Contains(err.Error(), "not initialized") {
				t.Errorf("Run() error should mention service not initialized, got: %v", err)
			}
		}
	})

	t.Run("youtube service not initialized", func(t *testing.T) {
		engine := NewPlaylistEngine(&mockService{}, nil, nil)
		progressCh := make(chan ProgressUpdate, 10)

		_, err := engine.Run(context.Background(), "playlist123", "dest", progressCh)
		close(progressCh)

		if err == nil {
			t.Error("Run() expected error for nil youtube service")
		}
	})
}

func TestPlaylistEngine_Diff(t *testing.T) {
	sourceExport := &services.PlaylistExport{
		Playlist: services.Playlist{ID: "src", Name: "Source"},
		Tracks: []services.Track{
			{ID: "1", Title: "Track 1", Artist: "Artist A", ISRC: "ISRC1"},
			{ID: "2", Title: "Track 2", Artist: "Artist B", ISRC: "ISRC2"},
			{ID: "3", Title: "Track 3", Artist: "Artist C", ISRC: "ISRC3"},
		},
	}

	destExport := &services.PlaylistExport{
		Playlist: services.Playlist{ID: "dest", Name: "Destination"},
		Tracks: []services.Track{
			{ID: "10", Title: "Track 1", Artist: "Artist A", ISRC: "ISRC1"}, // Match by ISRC
			{ID: "20", Title: "Track 2", Artist: "Artist B"},                // Match by title+artist
			{ID: "40", Title: "Track 4", Artist: "Artist D", ISRC: "ISRC4"}, // Extra track
		},
	}

	sourceSvc := &mockService{
		name: "Spotify",
		playlistExports: map[string]*services.PlaylistExport{
			"src": sourceExport,
		},
	}

	destSvc := &mockService{
		name: "YouTube Music",
		playlistExports: map[string]*services.PlaylistExport{
			"dest": destExport,
		},
	}

	engine := NewPlaylistEngine(nil, nil, nil)

	progressCh := make(chan ProgressUpdate, 100)
	go func() {
		for range progressCh {
			// Drain progress channel
		}
	}()

	result, err := engine.Diff(context.Background(), sourceSvc, destSvc, "src", "dest", progressCh)
	close(progressCh)

	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	if result.Comparison.MatchedCount != 2 {
		t.Errorf("Diff() matchedCount = %v, want 2", result.Comparison.MatchedCount)
	}

	if len(result.Comparison.MissingInDest) != 1 {
		t.Errorf("Diff() missingInDest count = %v, want 1", len(result.Comparison.MissingInDest))
	} else if result.Comparison.MissingInDest[0].ID != "3" {
		t.Errorf("Diff() missing track ID = %v, want '3'", result.Comparison.MissingInDest[0].ID)
	}

	if len(result.Comparison.ExtraInDest) != 1 {
		t.Errorf("Diff() extraInDest count = %v, want 1", len(result.Comparison.ExtraInDest))
	} else if result.Comparison.ExtraInDest[0].ID != "40" {
		t.Errorf("Diff() extra track ID = %v, want '40'", result.Comparison.ExtraInDest[0].ID)
	}
}

func TestPlaylistEngine_Dump(t *testing.T) {
	apiClient := &mockAPIClient{
		responses: map[string]*services.APIResponse{
			"/health": {
				StatusCode: 200,
				IsJSON:     true,
				JSONData:   map[string]string{"status": "ok"},
			},
			"/api/library/playlists": {
				StatusCode: 200,
				IsJSON:     true,
				JSONData:   []string{"playlist1", "playlist2"},
			},
			"/api/library/songs": {
				StatusCode: 500,
				Body:       []byte("internal error"),
			},
		},
	}

	engine := NewPlaylistEngine(nil, nil, apiClient)

	progressCh := make(chan ProgressUpdate, 100)
	progressUpdates := []ProgressUpdate{}
	done := make(chan bool)

	go func() {
		for update := range progressCh {
			progressUpdates = append(progressUpdates, update)
		}
		done <- true
	}()

	result, err := engine.Dump(context.Background(), progressCh)
	close(progressCh)
	<-done

	if err != nil {
		t.Fatalf("Dump() error = %v", err)
	}

	if result.Health == nil {
		t.Error("Dump() health data should not be nil")
	}

	if result.Playlists == nil {
		t.Error("Dump() playlists data should not be nil")
	}

	if len(result.Errors) == 0 {
		t.Error("Dump() should have errors for failed endpoints")
	}

	if len(progressUpdates) == 0 {
		t.Error("Dump() should send progress updates")
	}
}

func TestPlaylistEngine_Dump_APIClientError(t *testing.T) {
	engine := NewPlaylistEngine(nil, nil, nil)
	progressCh := make(chan ProgressUpdate, 10)

	_, err := engine.Dump(context.Background(), progressCh)
	close(progressCh)

	if err == nil {
		t.Error("Dump() expected error for nil API client")
	}
}

func TestProgressUpdate_NonBlocking(t *testing.T) {
	engine := NewPlaylistEngine(
		&mockService{
			name: "Spotify",
			playlistExports: map[string]*services.PlaylistExport{
				"p1": {
					Playlist: services.Playlist{ID: "p1", Name: "Test"},
					Tracks:   []services.Track{{ID: "t1", Title: "Song", Artist: "Artist"}},
				},
			},
		},
		&mockService{
			name: "YouTube Music",
			searchResults: map[string]*services.Track{
				"Song|Artist": {ID: "yt1", Title: "Song", Artist: "Artist"},
			},
			importResult: &services.Playlist{ID: "ytp1", Name: "Test", TrackCount: 1},
		},
		nil,
	)

	// Create a channel with buffer 0 to test non-blocking behavior
	progressCh := make(chan ProgressUpdate)

	// Don't consume from channel to simulate blocked consumer

	// Run should complete even though progress channel is not being read
	done := make(chan bool)
	go func() {
		_, err := engine.Run(context.Background(), "p1", "dest", progressCh)
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
		done <- true
	}()

	select {
	case <-done:
		// Success - operation completed even with blocked progress channel
	case <-context.Background().Done():
		t.Error("Run() should not block on progress sends")
	}
}
