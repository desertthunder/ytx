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

	"github.com/desertthunder/ytx/internal/models"
	"github.com/desertthunder/ytx/internal/shared"
	"golang.org/x/oauth2"
)

const (
	spotifyAuthURL     = "https://accounts.spotify.com/authorize"
	spotifyTokenURL    = "https://accounts.spotify.com/api/token"
	spotifyBaseURL     = "https://api.spotify.com/v1"
	DefaultRedirectURI = "http://localhost:3000/callback"
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

type createPlaylistReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
}

// SpotifySearchResults represents the response from Spotify's search API.
type SpotifySearchResults struct {
	Tracks struct {
		Items []SpotifyTrack `json:"items"`
		Total int            `json:"total"`
	} `json:"tracks"`
}

// tokenRefreshCallback is called when a token is refreshed by the TokenSource
type tokenRefreshCallback func(*oauth2.Token)

// refreshableTokenSource wraps an oauth2.TokenSource to intercept token refreshes
type refreshableTokenSource struct {
	source    oauth2.TokenSource
	callback  tokenRefreshCallback
	lastToken *oauth2.Token
}

func (r *refreshableTokenSource) Token() (*oauth2.Token, error) {
	token, err := r.source.Token()
	if err != nil {
		return nil, err
	}

	// Check if token was refreshed (access token changed)
	if r.lastToken == nil || r.lastToken.AccessToken != token.AccessToken {
		if r.callback != nil {
			r.callback(token)
		}
		r.lastToken = token
	}

	return token, nil
}

// SpotifyService implements the Service interface for Spotify API interactions.
//
// Uses [oauth2] for authentication and provides methods for playlist and track operations.
type SpotifyService struct {
	config         *oauth2.Config
	token          *oauth2.Token
	httpClient     *http.Client
	credentials    map[string]string
	onTokenRefresh tokenRefreshCallback
}

// SetTokenRefreshCallback sets a callback to be invoked when tokens are refreshed
func (s *SpotifyService) SetTokenRefreshCallback(callback tokenRefreshCallback) {
	s.onTokenRefresh = callback
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
		redirectURI = "DefaultRedirectURI"
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
			"playlist-modify-public",
			"playlist-modify-private",
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

// Authenticate performs OAuth2 authentication with Spotify.
//
// Expects either an "access_token" or "auth_code" in credentials. Optionally accepts a "refresh_token" to enable automatic token refresh.
func (s *SpotifyService) Authenticate(ctx context.Context, credentials map[string]string) error {
	if accessToken, ok := credentials["access_token"]; ok && accessToken != "" {
		s.token = &oauth2.Token{
			AccessToken:  accessToken,
			RefreshToken: credentials["refresh_token"], // May be empty
		}
		s.httpClient = s.createClientWithRefreshCallback(ctx, s.token)
		return nil
	}

	if authCode, ok := credentials["auth_code"]; ok && authCode != "" {
		token, err := s.config.Exchange(ctx, authCode)
		if err != nil {
			return fmt.Errorf("failed to exchange auth code: %w", err)
		}
		s.token = token
		s.httpClient = s.createClientWithRefreshCallback(ctx, s.token)
		return nil
	}

	return fmt.Errorf("missing access_token or auth_code in credentials")
}

// createClientWithRefreshCallback creates an HTTP client with a TokenSource that captures token refreshes
func (s *SpotifyService) createClientWithRefreshCallback(ctx context.Context, token *oauth2.Token) *http.Client {
	tokenSource := s.config.TokenSource(ctx, token)

	if s.onTokenRefresh != nil {
		tokenSource = &refreshableTokenSource{
			source:    tokenSource,
			callback:  s.onTokenRefresh,
			lastToken: token,
		}
	}

	return oauth2.NewClient(ctx, tokenSource)
}

func (s *SpotifyService) Name() string {
	return "Spotify"
}

// GetOAuthConfig returns the OAuth2 config for external use (e.g., OAuth handler).
func (s *SpotifyService) GetOAuthConfig() *oauth2.Config {
	return s.config
}

// GetAuthURL returns the OAuth2 authorization URL for user login.
func (s *SpotifyService) GetAuthURL(state string) string {
	return s.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// GetToken returns the current OAuth2 token (may have been refreshed automatically).
func (s *SpotifyService) GetToken() *oauth2.Token {
	return s.token
}

// OAuthenticate authenticates the service using an OAuth2 token directly.
// Implements the OAuthService interface for reauthorization flows.
func (s *SpotifyService) OAuthenticate(ctx context.Context, token *oauth2.Token) error {
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}
	s.token = token
	s.httpClient = s.createClientWithRefreshCallback(ctx, s.token)
	return nil
}

// doRequest performs an authenticated HTTP request to the Spotify API.
// The oauth2 client automatically handles token refresh on 401 responses.
func (s *SpotifyService) doRequest(ctx context.Context, method, endpoint string, body any, result any) error {
	if s.token == nil {
		return fmt.Errorf("%w: call Authenticate first", shared.ErrNotAuthenticated)
	}

	apiURL := spotifyBaseURL + endpoint

	var req *http.Request
	var err error

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		req, err = http.NewRequestWithContext(ctx, method, apiURL, strings.NewReader(string(jsonBody)))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
	} else {
		req, err = http.NewRequestWithContext(ctx, method, apiURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("%w: %s", shared.ErrTokenExpired, "Spotify returned 401 - reauthorization required")
	}

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
	if err := s.doRequest(ctx, http.MethodGet, "/me", nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// Track retrieves a single track by ID.
func (s *SpotifyService) Track(ctx context.Context, trackID string) (*SpotifyTrack, error) {
	var track SpotifyTrack
	endpoint := fmt.Sprintf("/tracks/%s", trackID)
	if err := s.doRequest(ctx, http.MethodGet, endpoint, nil, &track); err != nil {
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

	if err := s.doRequest(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
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
	if err := s.doRequest(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
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
	if err := s.doRequest(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Playlist retrieves a playlist by ID.
func (s *SpotifyService) Playlist(ctx context.Context, playlistID string) (*SpotifyPlaylist, error) {
	endpoint := fmt.Sprintf("/playlists/%s", playlistID)

	var playlist SpotifyPlaylist
	if err := s.doRequest(ctx, http.MethodGet, endpoint, nil, &playlist); err != nil {
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

// GetPlaylists retrieves all playlists for the authenticated user.
func (s *SpotifyService) GetPlaylists(ctx context.Context) ([]models.Playlist, error) {
	var allPlaylists []models.Playlist
	limit := 50
	offset := 0

	for {
		response, err := s.UserPlaylists(ctx, limit, offset)
		if err != nil {
			return nil, err
		}

		for _, sp := range response.Items {
			allPlaylists = append(allPlaylists, models.Playlist{
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
func (s *SpotifyService) GetPlaylist(ctx context.Context, playlistID string) (*models.Playlist, error) {
	sp, err := s.Playlist(ctx, playlistID)
	if err != nil {
		return nil, err
	}

	return &models.Playlist{
		ID:          sp.ID,
		Name:        sp.Name,
		Description: sp.Description,
		TrackCount:  sp.Tracks.Total,
		Public:      sp.Public,
	}, nil
}

// ExportPlaylist exports a playlist with all its tracks.
func (s *SpotifyService) ExportPlaylist(ctx context.Context, playlistID string) (*models.PlaylistExport, error) {
	sp, err := s.Playlist(ctx, playlistID)
	if err != nil {
		return nil, err
	}

	playlist := models.Playlist{
		ID:          sp.ID,
		Name:        sp.Name,
		Description: sp.Description,
		TrackCount:  sp.Tracks.Total,
		Public:      sp.Public,
	}

	var tracks []models.Track
	for _, item := range sp.Tracks.Items {
		track := models.Track{
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

	return &models.PlaylistExport{
		Playlist: playlist,
		Tracks:   tracks,
	}, nil
}

// ImportPlaylist imports a playlist into Spotify by creating a new playlist and adding tracks.
//
// Requires OAuth scopes: playlist-modify-public, playlist-modify-private
func (s *SpotifyService) ImportPlaylist(ctx context.Context, playlist *models.PlaylistExport) (*models.Playlist, error) {
	user, err := s.UserProfile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	createReq := createPlaylistReq{
		Name:        playlist.Playlist.Name,
		Description: playlist.Playlist.Description,
		Public:      playlist.Playlist.Public,
	}

	var createdPlaylist SpotifyPlaylist
	endpoint := fmt.Sprintf("/users/%s/playlists", user.ID)
	if err := s.doRequest(ctx, http.MethodPost, endpoint, createReq, &createdPlaylist); err != nil {
		return nil, fmt.Errorf("failed to create playlist: %w", err)
	}

	if len(playlist.Tracks) > 0 {
		const batchSize = 100
		for i := 0; i < len(playlist.Tracks); i += batchSize {
			end := min(i+batchSize, len(playlist.Tracks))

			batch := playlist.Tracks[i:end]
			trackURIs := make([]string, len(batch))
			for j, track := range batch {
				trackURIs[j] = fmt.Sprintf("spotify:track:%s", track.ID)
			}

			addReq := struct {
				URIs []string `json:"uris"`
			}{
				URIs: trackURIs,
			}

			addEndpoint := fmt.Sprintf("/playlists/%s/tracks", createdPlaylist.ID)
			if err := s.doRequest(ctx, http.MethodPost, addEndpoint, addReq, nil); err != nil {
				return nil, fmt.Errorf("failed to add tracks (batch %d-%d): %w", i, end, err)
			}
		}
	}

	return &models.Playlist{
		ID:          createdPlaylist.ID,
		Name:        createdPlaylist.Name,
		Description: createdPlaylist.Description,
		TrackCount:  len(playlist.Tracks),
		Public:      createdPlaylist.Public,
	}, nil
}

// SearchTrack searches for a track by title and artist and returns the best match.
func (s *SpotifyService) SearchTrack(ctx context.Context, title, artist string) (*models.Track, error) {
	query := fmt.Sprintf("track:%s artist:%s", title, artist)
	endpoint := fmt.Sprintf("/search?q=%s&type=track&limit=1", url.QueryEscape(query))

	var results SpotifySearchResults
	if err := s.doRequest(ctx, http.MethodGet, endpoint, nil, &results); err != nil {
		return nil, err
	}

	if len(results.Tracks.Items) == 0 {
		return nil, fmt.Errorf("no results found for track '%s' by artist '%s'", title, artist)
	}

	spotifyTrack := results.Tracks.Items[0]
	track := &models.Track{
		ID:       spotifyTrack.ID,
		Title:    spotifyTrack.Name,
		Duration: spotifyTrack.DurationMS / 1000,
		ISRC:     spotifyTrack.ExternalIDs.ISRC,
	}

	if len(spotifyTrack.Artists) > 0 {
		track.Artist = spotifyTrack.Artists[0].Name
	}

	if spotifyTrack.Album.Name != "" {
		track.Album = spotifyTrack.Album.Name
	}

	return track, nil
}
