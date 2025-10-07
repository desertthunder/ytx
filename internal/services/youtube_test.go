package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestYouTubeService(t *testing.T) {
	t.Run("NewYouTubeService", func(t *testing.T) {
		t.Run("creates service with default URL", func(t *testing.T) {
			if svc := NewYouTubeService(""); svc == nil {
				t.Fatal("expected service to be created")
			} else if svc.baseURL != defaultYTBaseURL {
				t.Errorf("expected baseURL to be %s, got %s", defaultYTBaseURL, svc.baseURL)
			}
		})

		t.Run("creates service with custom URL", func(t *testing.T) {
			customURL := "http://localhost:9000"
			if svc := NewYouTubeService(customURL); svc.baseURL != customURL {
				t.Errorf("expected baseURL to be %s, got %s", customURL, svc.baseURL)
			}
		})
	})

	t.Run("Name", func(t *testing.T) {
		if svc := NewYouTubeService(""); svc.Name() != "YouTube Music" {
			t.Errorf("expected name to be 'YouTube Music', got %s", svc.Name())
		}
	})

	t.Run("Authenticate", func(t *testing.T) {
		svc := NewYouTubeService("")
		ctx := context.Background()

		t.Run("authenticates with auth_file", func(t *testing.T) {
			credentials := map[string]string{"auth_file": "/path/to/browser.json"}
			if err := svc.Authenticate(ctx, credentials); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if svc.authFile != credentials["auth_file"] {
				t.Errorf("expected authFile to be %s, got %s", credentials["auth_file"], svc.authFile)
			}
		})

		t.Run("fails without auth_file", func(t *testing.T) {
			credentials := map[string]string{}
			err := svc.Authenticate(ctx, credentials)
			if err == nil {
				t.Fatal("expected error for missing auth_file")
			}
		})
	})

	t.Run("GetPlaylists", func(t *testing.T) {
		mockPlaylists := []map[string]any{
			{
				"playlistId":  "PL123",
				"title":       "My Playlist",
				"description": "Test playlist",
				"privacy":     "PUBLIC",
				"count":       10,
			},
			{
				"playlistId":  "PL456",
				"title":       "Private Mix",
				"description": "Secret songs",
				"privacy":     "PRIVATE",
				"count":       5,
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/library/playlists" {
				t.Errorf("expected path /api/library/playlists, got %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("expected GET method, got %s", r.Method)
			}
			if r.Header.Get("X-Auth-File") != "/path/to/auth.json" {
				t.Errorf("expected X-Auth-File header")
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockPlaylists)
		}))
		defer server.Close()

		svc := NewYouTubeService(server.URL)
		svc.authFile = "/path/to/auth.json"

		playlists, err := svc.GetPlaylists(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(playlists) != 2 {
			t.Fatalf("expected 2 playlists, got %d", len(playlists))
		}

		if playlists[0].ID != "PL123" {
			t.Errorf("expected first playlist ID to be PL123, got %s", playlists[0].ID)
		}
		if playlists[0].Name != "My Playlist" {
			t.Errorf("expected first playlist name to be 'My Playlist', got %s", playlists[0].Name)
		}
		if !playlists[0].Public {
			t.Error("expected first playlist to be public")
		}

		if playlists[1].Public {
			t.Error("expected second playlist to be private")
		}
	})

	t.Run("GetPlaylist", func(t *testing.T) {
		mockPlaylist := map[string]any{
			"id":          "PL123",
			"title":       "Test Playlist",
			"description": "A test playlist",
			"privacy":     "PUBLIC",
			"trackCount":  15,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/playlists/PL123" {
				t.Errorf("expected path /api/playlists/PL123, got %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockPlaylist)
		}))
		defer server.Close()

		svc := NewYouTubeService(server.URL)
		playlist, err := svc.GetPlaylist(context.Background(), "PL123")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if playlist.ID != "PL123" {
			t.Errorf("expected ID PL123, got %s", playlist.ID)
		}
		if playlist.TrackCount != 15 {
			t.Errorf("expected track count 15, got %d", playlist.TrackCount)
		}
	})

	t.Run("ExportPlaylist", func(t *testing.T) {
		mockPlaylist := map[string]any{
			"id":          "PL123",
			"title":       "Export Test",
			"description": "Test export",
			"privacy":     "PRIVATE",
			"trackCount":  2,
			"tracks": []map[string]any{
				{
					"videoId": "vid1",
					"title":   "Song 1",
					"artists": []map[string]any{
						{"name": "Artist 1", "id": "art1"},
					},
					"album": map[string]any{
						"name": "Album 1",
						"id":   "alb1",
					},
					"duration_seconds": 180,
					"isrc":             "USABC1234567",
				},
				{
					"videoId": "vid2",
					"title":   "Song 2",
					"artists": []map[string]any{
						{"name": "Artist 2", "id": "art2"},
					},
					"duration_seconds": 240,
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockPlaylist)
		}))
		defer server.Close()

		svc := NewYouTubeService(server.URL)
		export, err := svc.ExportPlaylist(context.Background(), "PL123")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if export.Playlist.ID != "PL123" {
			t.Errorf("expected playlist ID PL123, got %s", export.Playlist.ID)
		}
		if len(export.Tracks) != 2 {
			t.Fatalf("expected 2 tracks, got %d", len(export.Tracks))
		}

		track1 := export.Tracks[0]
		if track1.ID != "vid1" {
			t.Errorf("expected track ID vid1, got %s", track1.ID)
		}
		if track1.Title != "Song 1" {
			t.Errorf("expected track title 'Song 1', got %s", track1.Title)
		}
		if track1.Artist != "Artist 1" {
			t.Errorf("expected artist 'Artist 1', got %s", track1.Artist)
		}
		if track1.Album != "Album 1" {
			t.Errorf("expected album 'Album 1', got %s", track1.Album)
		}
		if track1.Duration != 180 {
			t.Errorf("expected duration 180, got %d", track1.Duration)
		}
		if track1.ISRC != "USABC1234567" {
			t.Errorf("expected ISRC USABC1234567, got %s", track1.ISRC)
		}
	})

	t.Run("ImportPlaylist", func(t *testing.T) {
		var createdPlaylistID string
		var receivedTracks []string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/playlists" && r.Method == "POST" {
				var req struct {
					Title         string `json:"title"`
					Description   string `json:"description"`
					PrivacyStatus string `json:"privacy_status"`
				}
				json.NewDecoder(r.Body).Decode(&req)

				if req.Title != "Import Test" {
					t.Errorf("expected title 'Import Test', got %s", req.Title)
				}
				if req.PrivacyStatus != "PUBLIC" {
					t.Errorf("expected privacy_status PUBLIC, got %s", req.PrivacyStatus)
				}

				createdPlaylistID = "PL_NEW_123"
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"playlist_id": createdPlaylistID,
				})
			} else if r.URL.Path == "/api/playlists/PL_NEW_123/items" && r.Method == "POST" {
				var req struct {
					VideoIDs []string `json:"video_ids"`
				}
				json.NewDecoder(r.Body).Decode(&req)
				receivedTracks = req.VideoIDs

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"status": "success",
				})
			} else {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		svc := NewYouTubeService(server.URL)
		svc.authFile = "/path/to/auth.json"

		export := &PlaylistExport{
			Playlist: Playlist{Name: "Import Test", Description: "Test import", Public: true},
			Tracks:   []Track{{ID: "vid1", Title: "Track 1"}, {ID: "vid2", Title: "Track 2"}},
		}

		result, err := svc.ImportPlaylist(context.Background(), export)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if result.ID != "PL_NEW_123" {
			t.Errorf("expected playlist ID PL_NEW_123, got %s", result.ID)
		}
		if result.Name != "Import Test" {
			t.Errorf("expected name 'Import Test', got %s", result.Name)
		}
		if !result.Public {
			t.Error("expected playlist to be public")
		}

		if len(receivedTracks) != 2 {
			t.Fatalf("expected 2 tracks to be added, got %d", len(receivedTracks))
		}
		if receivedTracks[0] != "vid1" || receivedTracks[1] != "vid2" {
			t.Errorf("expected tracks [vid1, vid2], got %v", receivedTracks)
		}
	})

	t.Run("SearchTrack", func(t *testing.T) {
		mockResults := []map[string]any{
			{
				"videoId":          "vid123",
				"title":            "Harder Better Faster Stronger",
				"artists":          []map[string]any{{"name": "Daft Punk", "id": "art1"}},
				"album":            map[string]any{"name": "Discovery"},
				"duration_seconds": 224,
				"isrc":             "USVIRGIN01234",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/search" {
				t.Errorf("expected path /api/search, got %s", r.URL.Path)
			}

			query := r.URL.Query().Get("q")
			if query != "Harder Better Faster Stronger Daft Punk" {
				t.Errorf("expected query to contain title and artist, got %s", query)
			}

			filter := r.URL.Query().Get("filter")
			if filter != "songs" {
				t.Errorf("expected filter 'songs', got %s", filter)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResults)
		}))
		defer server.Close()

		svc := NewYouTubeService(server.URL)
		track, err := svc.SearchTrack(context.Background(), "Harder Better Faster Stronger", "Daft Punk")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if track.ID != "vid123" {
			t.Errorf("expected track ID vid123, got %s", track.ID)
		}
		if track.Title != "Harder Better Faster Stronger" {
			t.Errorf("expected title 'Harder Better Faster Stronger', got %s", track.Title)
		}
		if track.Artist != "Daft Punk" {
			t.Errorf("expected artist 'Daft Punk', got %s", track.Artist)
		}
		if track.Album != "Discovery" {
			t.Errorf("expected album 'Discovery', got %s", track.Album)
		}
	})

	t.Run("No Results from SearchTrack", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]any{})
		}))
		defer server.Close()

		svc := NewYouTubeService(server.URL)
		_, err := svc.SearchTrack(context.Background(), "Unknown Song", "Unknown Artist")
		if err == nil {
			t.Fatal("expected error for no results")
		}
	})

	t.Run("Error Handling", func(t *testing.T) {
		t.Run("handles 401 unauthorized", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"detail": "Authentication required",
				})
			}))
			defer server.Close()

			svc := NewYouTubeService(server.URL)
			if _, err := svc.GetPlaylists(context.Background()); err == nil {
				t.Fatal("expected error for 401")
			}
		})

		t.Run("handles 404 not found", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"detail": "Playlist not found"})
			}))
			defer server.Close()

			svc := NewYouTubeService(server.URL)
			if _, err := svc.GetPlaylist(context.Background(), "INVALID"); err == nil {
				t.Fatal("expected error for 404")
			}
		})

		t.Run("handles 500 internal error", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"detail": "Internal server error"})
			}))
			defer server.Close()

			svc := NewYouTubeService(server.URL)
			if _, err := svc.GetPlaylists(context.Background()); err == nil {
				t.Fatal("expected error for 500")
			}
		})
	})
}
