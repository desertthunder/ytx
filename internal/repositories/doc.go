// Package repositories implements SQLite persistence for all domain entities.
//
// Each repository handles CRUD operations with atomic sequence generation for human-readable ordering.
// All repositories support soft deletes via deleted_at timestamps and exclude deleted records from queries by default.
//
// Key Implementations:
//   - [UserRepository] : User account persistence with email-based lookups
//   - [PlaylistRepository] : Playlist caching with service-specific queries
//   - [TrackRepository] : Track caching with ISRC-based cross-service matching
//   - [PlaylistTrackRepository] : Junction table managing playlist track membership
//   - [MigrationJobRepository] : Migration history with status tracking
//
// Sequence numbers provide stable, human-readable ordering (e.g., user #42, playlist #15) independent of UUIDs and creation timestamps.
// The [NextSequence] function atomically increments per-table sequence counters in dedicated sequence tables.
package repositories
