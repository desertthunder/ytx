// Package tasks orchestrates playlist operations between music services with real-time progress reporting.
//
// # Core Operations
//
// The [SyncEngine] interface defines three operations:
//
//  1. [SyncEngine.Run] : Full Spotify → YouTube Music transfer
//     - Fetches source playlist from Spotify
//     - Searches each track on YouTube Music (ISRC or fuzzy match)
//     - Creates destination playlist with matched tracks
//     - Returns detailed results including failed matches
//
//  2. [SyncEngine.Diff] : Compare playlists across services
//     - Exports both source and destination playlists
//     - Matches tracks via ISRC (preferred) or normalized title/artist
//     - Reports matched count, missing tracks, and extra tracks
//
//  3. [SyncEngine.Dump] : Fetch all YouTube Music library data
//     - Retrieves playlists, songs, albums, artists, history, uploads
//     - Returns structured data for backup or analysis
//
// # Progress Reporting
//
// # All operations use non-blocking channels for progress updates
//
// The [ProgressUpdate] struct contains phase, step counters, messages, and optional data for advanced UI rendering.
// Updates use select with default to prevent blocking.
//
// # Track Caching
//
// The optional [TrackCacher] interface enables automatic track persistence during transfers
//
// Tracks are cached silently (errors ignored) to avoid disrupting transfers.

// This supports ISRC-based matching across future operations and analytics on migration patterns.
//
// # Implementation
//
// [PlaylistEngine] implements [SyncEngine] with dependencies on:
//   - [services.Service] : Spotify and YouTube Music API clients
//   - [APIClient] : HTTP client for YouTube Music proxy
//   - [TrackCacher] : Optional persistence layer (repositories.TrackRepository)
package tasks
