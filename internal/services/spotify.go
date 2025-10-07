// Spotify API implementation of [Service]
//
// Spotify API response types based on https://developer.spotify.com/documentation/web-api/reference/
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/desertthunder/song-migrations/internal/shared"
	"golang.org/x/oauth2"
)

const (
	spotifyAuthURL  = "https://accounts.spotify.com/authorize"
	spotifyTokenURL = "https://accounts.spotify.com/api/token"
	spotifyBaseURL  = "https://api.spotify.com/v1"
)

type followers struct {
	Total int `json:"total"`
}

// SpotifyUser represents a Spotify user profile.
type SpotifyUser struct {
	ID          string         `json:"id"`
	DisplayName string         `json:"display_name"`
	Email       string         `json:"email"`
	Country     string         `json:"country"`
	Product     string         `json:"product"` // premium, free, etc.
	Followers   followers      `json:"followers"`
	Images      []SpotifyImage `json:"images"`
}

// SpotifyImage represents an image resource.
type SpotifyImage struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

type externalIDs struct {
	ISRC string `json:"isrc"`
}

// SpotifyTrack represents a Spotify track.
type SpotifyTrack struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Artists     []SpotifyArtist `json:"artists"`
	Album       SpotifyAlbum    `json:"album"`
	DurationMS  int             `json:"duration_ms"`
	Explicit    bool            `json:"explicit"`
	ExternalIDs externalIDs     `json:"external_ids"`
	Popularity  int             `json:"popularity"`
	URI         string          `json:"uri"`
}

// SpotifyArtist represents a Spotify artist.
type SpotifyArtist struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Genres []string       `json:"genres"`
	Images []SpotifyImage `json:"images"`
	URI    string         `json:"uri"`
}

// SpotifyAlbum represents a Spotify album.
type SpotifyAlbum struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Artists     []SpotifyArtist `json:"artists"`
	ReleaseDate string          `json:"release_date"`
	TotalTracks int             `json:"total_tracks"`
	Images      []SpotifyImage  `json:"images"`
	URI         string          `json:"uri"`
}

type Owner struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type playlistTrack struct {
	Total int                    `json:"total"`
	Items []SpotifyPlaylistTrack `json:"items"`
}

// SpotifyPlaylist represents a Spotify playlist.
type SpotifyPlaylist struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Owner       Owner          `json:"owner"`
	Public      bool           `json:"public"`
	Tracks      playlistTrack  `json:"tracks"`
	Images      []SpotifyImage `json:"images"`
	URI         string         `json:"uri"`
}

// SpotifyPlaylistTrack represents a track within a playlist context.
type SpotifyPlaylistTrack struct {
	AddedAt string       `json:"added_at"`
	Track   SpotifyTrack `json:"track"`
}

// SpotifyPaginatedTracks represents a paginated response of saved tracks.
type SpotifyPaginatedTracks struct {
	Items    []SpotifySavedTrack `json:"items"`
	Total    int                 `json:"total"`
	Limit    int                 `json:"limit"`
	Offset   int                 `json:"offset"`
	Next     *string             `json:"next"`
	Previous *string             `json:"previous"`
}

// SpotifySavedTrack represents a track saved in the user's library.
type SpotifySavedTrack struct {
	AddedAt string       `json:"added_at"`
	Track   SpotifyTrack `json:"track"`
}

// SpotifyPaginatedPlaylists represents a paginated response of playlists.
type SpotifyPaginatedPlaylists struct {
	Items    []SpotifySimplePlaylist `json:"items"`
	Total    int                     `json:"total"`
	Limit    int                     `json:"limit"`
	Offset   int                     `json:"offset"`
	Next     *string                 `json:"next"`
	Previous *string                 `json:"previous"`
}

type simplePlaylistTrack struct {
	Total int `json:"total"`
}

// SpotifySimplePlaylist represents a simplified playlist object (used in lists).
type SpotifySimplePlaylist struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Owner       Owner               `json:"owner"`
	Public      bool                `json:"public"`
	Tracks      simplePlaylistTrack `json:"tracks"`
	Images      []SpotifyImage      `json:"images"`
	URI         string              `json:"uri"`
}

// SpotifyService implements the Service interface for Spotify API interactions.
// Uses [oauth2] for authentication and provides methods for playlist and track operations.
type SpotifyService struct {
	config      *oauth2.Config
	token       *oauth2.Token
	httpClient  *http.Client
	credentials map[string]string
}

// NewSpotifyService creates a new Spotify service with the given OAuth2 credentials.
func NewSpotifyService(credentials map[string]string) (*SpotifyService, error) {
	clientID, ok := credentials["client_id"]
	if !ok || clientID == "" {
		return nil, fmt.Errorf("missing client_id in credentials")
	}

	clientSecret, ok := credentials["client_secret"]
	if !ok || clientSecret == "" {
		return nil, fmt.Errorf("missing client_secret in credentials")
	}

	redirectURI, ok := credentials["redirect_uri"]
	if !ok || redirectURI == "" {
		redirectURI = "http://localhost:8080/callback"
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes: []string{
			"user-read-private",
			"user-read-email",
			"playlist-read-private",
			"playlist-read-collaborative",
			"user-library-read",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  spotifyAuthURL,
			TokenURL: spotifyTokenURL,
		},
	}

	return &SpotifyService{
		config:      config,
		httpClient:  http.DefaultClient,
		credentials: credentials,
	}, nil
}

// Authenticate performs OAuth2 authentication with Spotify. Expects either an "access_token" or "auth_code" in credentials.
func (s *SpotifyService) Authenticate(ctx context.Context, credentials map[string]string) error {
	if accessToken, ok := credentials["access_token"]; ok && accessToken != "" {
		s.token = &oauth2.Token{AccessToken: accessToken}
		s.httpClient = s.config.Client(ctx, s.token)
		return nil
	}

	if authCode, ok := credentials["auth_code"]; ok && authCode != "" {
		token, err := s.config.Exchange(ctx, authCode)
		if err != nil {
			return fmt.Errorf("failed to exchange auth code: %w", err)
		}
		s.token = token
		s.httpClient = s.config.Client(ctx, s.token)
		return nil
	}

	return fmt.Errorf("missing access_token or auth_code in credentials")
}

func (s *SpotifyService) Name() string {
	return "Spotify"
}

// GetAuthURL returns the OAuth2 authorization URL for user login.
func (s *SpotifyService) GetAuthURL(state string) string {
	return s.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// doRequest performs an authenticated HTTP request to the Spotify API.
func (s *SpotifyService) doRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	if s.token == nil {
		return fmt.Errorf("not authenticated: call Authenticate first")
	}

	apiURL := spotifyBaseURL + endpoint

	var req *http.Request
	var err error

	if body != nil {
		// TODO: handle request body if needed for POST/PUT
		return fmt.Errorf("%w request body", shared.ErrNotImplemented)
	}

	req, err = http.NewRequestWithContext(ctx, method, apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("spotify API error: status %d", resp.StatusCode)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// UserProfile retrieves the current authenticated user's profile.
func (s *SpotifyService) UserProfile(ctx context.Context) (*SpotifyUser, error) {
	var user SpotifyUser
	if err := s.doRequest(ctx, "GET", "/me", nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// Track retrieves a single track by ID.
func (s *SpotifyService) Track(ctx context.Context, trackID string) (*SpotifyTrack, error) {
	var track SpotifyTrack
	endpoint := fmt.Sprintf("/tracks/%s", trackID)
	if err := s.doRequest(ctx, "GET", endpoint, nil, &track); err != nil {
		return nil, err
	}
	return &track, nil
}

// SeveralTracks retrieves multiple tracks by their IDs (up to 50).
func (s *SpotifyService) SeveralTracks(ctx context.Context, trackIDs []string) ([]SpotifyTrack, error) {
	if len(trackIDs) == 0 {
		return nil, fmt.Errorf("no track IDs provided")
	}
	if len(trackIDs) > 50 {
		return nil, fmt.Errorf("maximum 50 track IDs allowed")
	}

	ids := strings.Join(trackIDs, ",")
	endpoint := fmt.Sprintf("/tracks?ids=%s", url.QueryEscape(ids))

	var response struct {
		Tracks []SpotifyTrack `json:"tracks"`
	}

	if err := s.doRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return nil, err
	}

	return response.Tracks, nil
}

// SavedTracks retrieves the user's saved tracks with pagination.
func (s *SpotifyService) SavedTracks(ctx context.Context, limit, offset int) (*SpotifyPaginatedTracks, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	endpoint := fmt.Sprintf("/me/tracks?limit=%d&offset=%d", limit, offset)

	var response SpotifyPaginatedTracks
	if err := s.doRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// UserPlaylists retrieves the current user's playlists with pagination.
func (s *SpotifyService) UserPlaylists(ctx context.Context, limit, offset int) (*SpotifyPaginatedPlaylists, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	endpoint := fmt.Sprintf("/me/playlists?limit=%d&offset=%d", limit, offset)

	var response SpotifyPaginatedPlaylists
	if err := s.doRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Playlist retrieves a playlist by ID.
func (s *SpotifyService) Playlist(ctx context.Context, playlistID string) (*SpotifyPlaylist, error) {
	endpoint := fmt.Sprintf("/playlists/%s", playlistID)

	var playlist SpotifyPlaylist
	if err := s.doRequest(ctx, "GET", endpoint, nil, &playlist); err != nil {
		return nil, err
	}

	return &playlist, nil
}

// Album retrieves an album by ID (stub for future implementation).
func (s *SpotifyService) Album(ctx context.Context, albumID string) (*SpotifyAlbum, error) {
	// TODO: implement album retrieval
	return nil, shared.ErrNotImplemented
}

// SeveralAlbums retrieves multiple albums by IDs (stub for future implementation).
func (s *SpotifyService) SeveralAlbums(ctx context.Context, albumIDs []string) ([]SpotifyAlbum, error) {
	// TODO: implement multiple album retrieval
	return nil, shared.ErrNotImplemented
}

// Artist retrieves an artist by ID (stub for future implementation).
func (s *SpotifyService) Artist(ctx context.Context, artistID string) (*SpotifyArtist, error) {
	// TODO: implement artist retrieval
	return nil, shared.ErrNotImplemented
}

// SeveralArtists retrieves multiple artists by IDs (stub for future implementation).
func (s *SpotifyService) SeveralArtists(ctx context.Context, artistIDs []string) ([]SpotifyArtist, error) {
	// TODO: implement multiple artist retrieval
	return nil, shared.ErrNotImplemented
}

// Service interface implementation

// GetPlaylists retrieves all playlists for the authenticated user.
func (s *SpotifyService) GetPlaylists(ctx context.Context) ([]Playlist, error) {
	var allPlaylists []Playlist
	limit := 50
	offset := 0

	for {
		response, err := s.UserPlaylists(ctx, limit, offset)
		if err != nil {
			return nil, err
		}

		for _, sp := range response.Items {
			allPlaylists = append(allPlaylists, Playlist{
				ID:          sp.ID,
				Name:        sp.Name,
				Description: sp.Description,
				TrackCount:  sp.Tracks.Total,
				Public:      sp.Public,
			})
		}

		if response.Next == nil {
			break
		}
		offset += limit
	}

	return allPlaylists, nil
}

// GetPlaylist retrieves a specific playlist by ID.
func (s *SpotifyService) GetPlaylist(ctx context.Context, playlistID string) (*Playlist, error) {
	sp, err := s.Playlist(ctx, playlistID)
	if err != nil {
		return nil, err
	}

	return &Playlist{
		ID:          sp.ID,
		Name:        sp.Name,
		Description: sp.Description,
		TrackCount:  sp.Tracks.Total,
		Public:      sp.Public,
	}, nil
}

// ExportPlaylist exports a playlist with all its tracks.
func (s *SpotifyService) ExportPlaylist(ctx context.Context, playlistID string) (*PlaylistExport, error) {
	sp, err := s.Playlist(ctx, playlistID)
	if err != nil {
		return nil, err
	}

	playlist := Playlist{
		ID:          sp.ID,
		Name:        sp.Name,
		Description: sp.Description,
		TrackCount:  sp.Tracks.Total,
		Public:      sp.Public,
	}

	var tracks []Track
	for _, item := range sp.Tracks.Items {
		track := Track{
			ID:       item.Track.ID,
			Title:    item.Track.Name,
			Duration: item.Track.DurationMS / 1000,
			ISRC:     item.Track.ExternalIDs.ISRC,
		}

		if len(item.Track.Artists) > 0 {
			track.Artist = item.Track.Artists[0].Name
		}

		if item.Track.Album.Name != "" {
			track.Album = item.Track.Album.Name
		}

		tracks = append(tracks, track)
	}

	return &PlaylistExport{
		Playlist: playlist,
		Tracks:   tracks,
	}, nil
}

// ImportPlaylist imports a playlist into Spotify (stub - requires write permissions).
func (s *SpotifyService) ImportPlaylist(ctx context.Context, playlist *PlaylistExport) (*Playlist, error) {
	// TODO: implement playlist creation and track addition
	// Requires additional OAuth scopes: playlist-modify-public, playlist-modify-private
	return nil, fmt.Errorf("requires playlist creation and track addition: %w", shared.ErrNotImplemented)
}

// SearchTrack searches for a track by title and artist.
func (s *SpotifyService) SearchTrack(ctx context.Context, title, artist string) (*Track, error) {
	// TODO: implement search endpoint
	return nil, fmt.Errorf("requires search endpoint: %w", shared.ErrNotImplemented)
}
