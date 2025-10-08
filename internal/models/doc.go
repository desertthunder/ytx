// Package models defines domain entities and persistence interfaces for the YTX playlist migration service.
//
// The package contains two categories of types:
//
// 1. Data Transfer Objects (DTOs): Lightweight structs representing external service data
//   - [Playlist] : Basic playlist metadata from music services
//   - [PlaylistExport] : Playlist with complete track listing
//   - [Track] : Song metadata with ISRC for cross-service matching
//
// 2. Persistent Entities: Database-backed models with full lifecycle management
//   - [User] : User accounts with authentication and preferences
//   - [PersistedPlaylist] : Cached playlists with service metadata
//   - [PersistedTrack] : Cached tracks with ISRC for matching optimization
//   - [PlaylistTrack] : Junction table linking playlists to tracks with ordering
//   - [MigrationJob] : Migration operations tracking progress and results
//
// All persistent entities implement the Model interface providing ID generation, timestamps, validation, and soft delete support.
// The Repository[T] interface defines standard CRUD operations for database access.
package models
