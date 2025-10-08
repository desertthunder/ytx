package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/desertthunder/ytx/internal/models"
	"github.com/desertthunder/ytx/internal/shared"
)

// PlaylistRepository implements models.Repository[*models.PersistedPlaylist] for playlist caching.
//
// Handles playlist CRUD operations with soft delete support and service-specific lookups.
type PlaylistRepository struct {
	db *sql.DB
}

// NewPlaylistRepository creates a new PlaylistRepository with the given database connection
func NewPlaylistRepository(db *sql.DB) *PlaylistRepository {
	return &PlaylistRepository{db: db}
}

// Create inserts a new playlist into the database with generated ID and sequence
func (r *PlaylistRepository) Create(playlist *models.PersistedPlaylist) error {
	sequence, err := NextSequence(r.db, "playlists")
	if err != nil {
		return fmt.Errorf("failed to generate sequence: %w", err)
	}

	id := shared.GenerateID()
	playlist.SetID(id)

	if err := playlist.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO playlists (id, sequence, service, service_id, user_id, name, description, track_count, public, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.Exec(query,
		id,
		sequence,
		playlist.Service(),
		playlist.ServiceID(),
		playlist.UserID(),
		playlist.Name(),
		playlist.Description(),
		playlist.TrackCount(),
		playlist.Public(),
		playlist.CreatedAt(),
		playlist.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert playlist: %w", err)
	}

	return nil
}

// Get retrieves a playlist by ID, excluding soft-deleted playlists
func (r *PlaylistRepository) Get(id string) (*models.PersistedPlaylist, error) {
	query := `
		SELECT id, sequence, service, service_id, user_id, name, description, track_count, public, created_at, updated_at, deleted_at
		FROM playlists
		WHERE id = ? AND deleted_at IS NULL
	`

	return r.scanOne(r.db.QueryRow(query, id))
}

// GetByServiceID retrieves a playlist by service and service_id
func (r *PlaylistRepository) GetByServiceID(service, serviceID string) (*models.PersistedPlaylist, error) {
	query := `
		SELECT id, sequence, service, service_id, user_id, name, description, track_count, public, created_at, updated_at, deleted_at
		FROM playlists
		WHERE service = ? AND service_id = ? AND deleted_at IS NULL
	`

	return r.scanOne(r.db.QueryRow(query, service, serviceID))
}

// Update modifies an existing playlist in the database
func (r *PlaylistRepository) Update(playlist *models.PersistedPlaylist) error {
	if err := playlist.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	now := time.Now()
	playlist.SetUpdatedAt(now)

	query := `
		UPDATE playlists
		SET name = ?, description = ?, track_count = ?, public = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query,
		playlist.Name(),
		playlist.Description(),
		playlist.TrackCount(),
		playlist.Public(),
		now,
		playlist.ID(),
	)
	if err != nil {
		return fmt.Errorf("failed to update playlist: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("playlist not found or already deleted: %s", playlist.ID())
	}

	return nil
}

// Delete soft-deletes a playlist by ID
func (r *PlaylistRepository) Delete(id string) error {
	now := time.Now()

	query := `
		UPDATE playlists
		SET deleted_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, now, id)
	if err != nil {
		return fmt.Errorf("failed to delete playlist: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("playlist not found or already deleted: %s", id)
	}

	return nil
}

// List retrieves all playlists matching the given criteria, excluding soft-deleted playlists
func (r *PlaylistRepository) List(criteria map[string]any) ([]*models.PersistedPlaylist, error) {
	query := `
		SELECT id, sequence, service, service_id, user_id, name, description, track_count, public, created_at, updated_at, deleted_at
		FROM playlists
		WHERE deleted_at IS NULL
	`

	args := []any{}

	if userID, ok := criteria["user_id"].(string); ok && userID != "" {
		query += " AND user_id = ?"
		args = append(args, userID)
	}

	if service, ok := criteria["service"].(string); ok && service != "" {
		query += " AND service = ?"
		args = append(args, service)
	}

	query += " ORDER BY sequence ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query playlists: %w", err)
	}
	defer rows.Close()

	var playlists []*models.PersistedPlaylist
	for rows.Next() {
		playlist, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		playlists = append(playlists, playlist)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return playlists, nil
}

// scanOne scans a single row into a [models.PersistedPlaylist]
func (r *PlaylistRepository) scanOne(row *sql.Row) (*models.PersistedPlaylist, error) {
	var (
		id          string
		sequence    int
		service     string
		serviceID   string
		userID      string
		name        string
		description string
		trackCount  int
		public      bool
		createdAt   time.Time
		updatedAt   time.Time
		deletedAt   sql.NullTime
	)

	err := row.Scan(&id, &sequence, &service, &serviceID, &userID, &name, &description, &trackCount, &public, &createdAt, &updatedAt, &deletedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("playlist not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan playlist: %w", err)
	}

	dto := models.Playlist{
		ID:          serviceID,
		Name:        name,
		Description: description,
		TrackCount:  trackCount,
		Public:      public,
	}

	playlist := models.NewPersistedPlaylist(sequence, service, serviceID, userID, dto)
	playlist.SetID(id)
	playlist.SetUpdatedAt(updatedAt)
	if deletedAt.Valid {
		playlist.SetDeletedAt(&deletedAt.Time)
	}

	return playlist, nil
}

// scanRow scans a row from [sql.Rows] into a [models.PersistedPlaylist]
func (r *PlaylistRepository) scanRow(rows *sql.Rows) (*models.PersistedPlaylist, error) {
	var (
		id          string
		sequence    int
		service     string
		serviceID   string
		userID      string
		name        string
		description string
		trackCount  int
		public      bool
		createdAt   time.Time
		updatedAt   time.Time
		deletedAt   sql.NullTime
	)

	err := rows.Scan(&id, &sequence, &service, &serviceID, &userID, &name, &description, &trackCount, &public, &createdAt, &updatedAt, &deletedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan playlist: %w", err)
	}

	dto := models.Playlist{
		ID:          serviceID,
		Name:        name,
		Description: description,
		TrackCount:  trackCount,
		Public:      public,
	}

	playlist := models.NewPersistedPlaylist(sequence, service, serviceID, userID, dto)
	playlist.SetID(id)
	playlist.SetUpdatedAt(updatedAt)
	if deletedAt.Valid {
		playlist.SetDeletedAt(&deletedAt.Time)
	}

	return playlist, nil
}
