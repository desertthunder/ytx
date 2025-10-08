// Package services defines the [Service] interface for music streaming providers and implements it for Spotify and YouTube Music.
//
// # Service Interface
//
// All music providers implement a common abstraction, enabling playlist operations to work uniformly across providers.
//
// # Spotify Implementation
//
// [SpotifyService] uses OAuth2 for authentication with automatic token refresh.
//
// The [oauth2.Client] automatically refreshes expired tokens using the refresh token.
//
// # YouTube Music Implementation
//
// YouTubeService communicates with the FastAPI proxy server (music/) wrapping ytmusicapi
//
// The proxy handles YouTube Music authentication complexities.
// The auth_file path is sent via X-Auth-File header on each request.
// All YouTube operations are synchronous HTTP calls to the proxy endpoints.
//
// # OAuth Service Extension
//
// The [OAuthService] interface extends Service for OAuth providers
//
// [SpotifyService] implements this for server-side OAuth flows used by the CLI and web app.
//
// # Error Handling
//
// Services use typed errors from shared package:
//   - [shared.ErrNotAuthenticated] : Authenticate() not called
//   - [shared.ErrTokenExpired] : OAuth token expired, reauthorization needed
//   - [shared.ErrAPIRequest] : HTTP request failed
//   - [shared.ErrPlaylistNotFound] : Playlist ID not found
//
// # API Mappings
//
// Both services convert provider-specific JSON responses to models.Playlist and models.Track:
//   - Spotify: Maps [SpotifyPlaylist] → [models.Playlist] with ISRC from external_ids
//   - YouTube: Maps [YouTubePlaylist] → [models.Playlist] with ISRC from search results
//
// Track matching uses ISRC when available, falling back to normalized title/artist comparison.
package services
