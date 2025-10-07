// YouTube Music API [Service] implementation
//
// Communicates with the FastAPI proxy server (music/) running on port 8080.
// The proxy wraps ytmusicapi Python library for YouTube Music operations.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const defaultYTBaseURL string = "http://localhost:8080"

// YouTubeImage represents an image/thumbnail from YouTube Music.
type YouTubeImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// YouTubeArtist represents an artist in YouTube Music responses.
type YouTubeArtist struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type youtubeAlbum struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// YouTubeTrack represents a track/video in YouTube Music responses.
type YouTubeTrack struct {
	VideoID     string          `json:"videoId"`
	Title       string          `json:"title"`
	Artists     []YouTubeArtist `json:"artists"`
	Album       *youtubeAlbum   `json:"album"`
	Duration    string          `json:"duration"`
	DurationSec int             `json:"duration_seconds"` // Duration in seconds
	Thumbnails  []YouTubeImage  `json:"thumbnails"`
	ISRC        string          `json:"isrc,omitempty"`
	SetVideoID  string          `json:"setVideoId,omitempty"` // For playlist operations
}

// YouTubePlaylist represents a playlist from YouTube Music.
type YouTubePlaylist struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Privacy     string         `json:"privacy"`
	Thumbnails  []YouTubeImage `json:"thumbnails"`
	TrackCount  int            `json:"trackCount"`
	Tracks      []YouTubeTrack `json:"tracks,omitempty"`
}

// YouTubeService implements the Service interface for YouTube Music via proxy.
type YouTubeService struct {
	baseURL    string
	authFile   string
	httpClient *http.Client
}

// NewYouTubeService creates a new YouTube Music service instance.
func NewYouTubeService(baseURL string) *YouTubeService {
	if baseURL == "" {
		baseURL = defaultYTBaseURL
	}

	return &YouTubeService{
		baseURL:    baseURL,
		httpClient: http.DefaultClient,
	}
}

// Name returns the service name.
func (y *YouTubeService) Name() string {
	return "YouTube Music"
}

// Authenticate stores the authentication file path for subsequent requests.
//
// Expects credentials["auth_file"] to contain the path to browser.json or oauth.json.
func (y *YouTubeService) Authenticate(ctx context.Context, credentials map[string]string) error {
	authFile, ok := credentials["auth_file"]
	if !ok || authFile == "" {
		return fmt.Errorf("missing auth_file in credentials")
	}

	y.authFile = authFile
	return nil
}

func (y *YouTubeService) doRequest(ctx context.Context, method, endpoint string, _, result any) error {
	apiURL := y.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, method, apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if y.authFile != "" {
		req.Header.Set("X-Auth-File", y.authFile)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := y.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Detail string `json:"detail"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Detail != "" {
			return fmt.Errorf("youtube music API error (status %d): %s", resp.StatusCode, errResp.Detail)
		}
		return fmt.Errorf("youtube music API error: status %d", resp.StatusCode)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// GetPlaylists retrieves all playlists for the authenticated user.
//
// Calls GET /api/library/playlists on the proxy.
func (y *YouTubeService) GetPlaylists(ctx context.Context) ([]Playlist, error) {
	var ytPlaylists []struct {
		PlaylistID  string         `json:"playlistId"`
		Title       string         `json:"title"`
		Description string         `json:"description"`
		Privacy     string         `json:"privacy"`
		Count       int            `json:"count"`
		Thumbnails  []YouTubeImage `json:"thumbnails"`
	}

	if err := y.doRequest(ctx, http.MethodGet, "/api/library/playlists", nil, &ytPlaylists); err != nil {
		return nil, err
	}

	playlists := make([]Playlist, len(ytPlaylists))
	for i, ytp := range ytPlaylists {
		playlists[i] = Playlist{
			ID:          ytp.PlaylistID,
			Name:        ytp.Title,
			Description: ytp.Description,
			TrackCount:  ytp.Count,
			Public:      ytp.Privacy == "PUBLIC",
		}
	}

	return playlists, nil
}

// GetPlaylist retrieves a specific playlist by ID without tracks.
//
// Calls GET /api/playlists/{id} on the proxy.
func (y *YouTubeService) GetPlaylist(ctx context.Context, playlistID string) (*Playlist, error) {
	var ytPlaylist struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Privacy     string `json:"privacy"`
		TrackCount  int    `json:"trackCount"`
	}

	endpoint := fmt.Sprintf("/api/playlists/%s", playlistID)
	if err := y.doRequest(ctx, http.MethodGet, endpoint, nil, &ytPlaylist); err != nil {
		return nil, err
	}

	return &Playlist{
		ID:          ytPlaylist.ID,
		Name:        ytPlaylist.Title,
		Description: ytPlaylist.Description,
		TrackCount:  ytPlaylist.TrackCount,
		Public:      ytPlaylist.Privacy == "PUBLIC",
	}, nil
}

// ExportPlaylist exports a playlist with all its tracks.
//
// Calls GET /api/playlists/{id} on the proxy.
func (y *YouTubeService) ExportPlaylist(ctx context.Context, playlistID string) (*PlaylistExport, error) {
	var ytPlaylist struct {
		ID          string         `json:"id"`
		Title       string         `json:"title"`
		Description string         `json:"description"`
		Privacy     string         `json:"privacy"`
		TrackCount  int            `json:"trackCount"`
		Tracks      []YouTubeTrack `json:"tracks"`
	}

	endpoint := fmt.Sprintf("/api/playlists/%s", playlistID)
	if err := y.doRequest(ctx, http.MethodGet, endpoint, nil, &ytPlaylist); err != nil {
		return nil, err
	}

	playlist := Playlist{
		ID:          ytPlaylist.ID,
		Name:        ytPlaylist.Title,
		Description: ytPlaylist.Description,
		TrackCount:  ytPlaylist.TrackCount,
		Public:      ytPlaylist.Privacy == "PUBLIC",
	}

	tracks := make([]Track, len(ytPlaylist.Tracks))
	for i, ytt := range ytPlaylist.Tracks {
		track := Track{
			ID:       ytt.VideoID,
			Title:    ytt.Title,
			Duration: ytt.DurationSec,
			ISRC:     ytt.ISRC,
		}

		if len(ytt.Artists) > 0 {
			track.Artist = ytt.Artists[0].Name
		}

		if ytt.Album != nil {
			track.Album = ytt.Album.Name
		}

		tracks[i] = track
	}

	return &PlaylistExport{
		Playlist: playlist,
		Tracks:   tracks,
	}, nil
}

// ImportPlaylist imports a playlist into YouTube Music.
//
// Creates the playlist via POST /api/playlists and adds tracks via POST /api/playlists/{id}/items.
func (y *YouTubeService) ImportPlaylist(ctx context.Context, playlist *PlaylistExport) (*Playlist, error) {
	createReq := struct {
		Title         string `json:"title"`
		Description   string `json:"description"`
		PrivacyStatus string `json:"privacy_status"`
	}{
		Title:         playlist.Playlist.Name,
		Description:   playlist.Playlist.Description,
		PrivacyStatus: "PRIVATE",
	}

	if playlist.Playlist.Public {
		createReq.PrivacyStatus = "PUBLIC"
	}

	reqBody := fmt.Sprintf(`{"title":"%s","description":"%s","privacy_status":"%s"}`,
		createReq.Title, createReq.Description, createReq.PrivacyStatus)

	apiURL := y.baseURL + "/api/playlists"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if y.authFile != "" {
		req.Header.Set("X-Auth-File", y.authFile)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := y.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to create playlist: status %d", resp.StatusCode)
	}

	var createResp struct {
		PlaylistID string `json:"playlist_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode create response: %w", err)
	}

	if len(playlist.Tracks) > 0 {
		videoIDs := make([]string, len(playlist.Tracks))
		for i, track := range playlist.Tracks {
			videoIDs[i] = track.ID
		}

		addReq := struct {
			VideoIDs []string `json:"video_ids"`
		}{
			VideoIDs: videoIDs,
		}

		addBody, err := json.Marshal(addReq)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal add tracks request: %w", err)
		}

		addURL := fmt.Sprintf("%s/api/playlists/%s/items", y.baseURL, createResp.PlaylistID)
		addReqHTTP, err := http.NewRequestWithContext(ctx, "POST", addURL, strings.NewReader(string(addBody)))
		if err != nil {
			return nil, fmt.Errorf("failed to create add tracks request: %w", err)
		}

		if y.authFile != "" {
			addReqHTTP.Header.Set("X-Auth-File", y.authFile)
		}
		addReqHTTP.Header.Set("Content-Type", "application/json")

		addResp, err := y.httpClient.Do(addReqHTTP)
		if err != nil {
			return nil, fmt.Errorf("failed to add tracks: %w", err)
		}
		defer addResp.Body.Close()

		if addResp.StatusCode < 200 || addResp.StatusCode >= 300 {
			return nil, fmt.Errorf("failed to add tracks to playlist: status %d", addResp.StatusCode)
		}
	}

	return &Playlist{
		ID:          createResp.PlaylistID,
		Name:        playlist.Playlist.Name,
		Description: playlist.Playlist.Description,
		TrackCount:  len(playlist.Tracks),
		Public:      playlist.Playlist.Public,
	}, nil
}

// SearchTrack searches for a track by title and artist, returning the best match.
//
// Calls GET /api/search?q={title} {artist}&filter=songs on the proxy.
func (y *YouTubeService) SearchTrack(ctx context.Context, title, artist string) (*Track, error) {
	query := fmt.Sprintf("%s %s", title, artist)
	endpoint := fmt.Sprintf("/api/search?q=%s&filter=songs", url.QueryEscape(query))

	var results []struct {
		VideoID string          `json:"videoId"`
		Title   string          `json:"title"`
		Artists []YouTubeArtist `json:"artists"`
		Album   *struct {
			Name string `json:"name"`
		} `json:"album"`
		Duration   string `json:"duration"`
		DurationMS int    `json:"duration_seconds"`
		ISRC       string `json:"isrc,omitempty"`
	}

	if err := y.doRequest(ctx, http.MethodGet, endpoint, nil, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for '%s' by '%s'", title, artist)
	}

	result := results[0]
	track := &Track{
		ID:       result.VideoID,
		Title:    result.Title,
		Duration: result.DurationMS,
		ISRC:     result.ISRC,
	}

	if len(result.Artists) > 0 {
		track.Artist = result.Artists[0].Name
	}

	if result.Album != nil {
		track.Album = result.Album.Name
	}

	return track, nil
}
