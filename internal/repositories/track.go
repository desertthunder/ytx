package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/desertthunder/ytx/internal/models"
	"github.com/desertthunder/ytx/internal/shared"
)

// TrackRepository implements models.Repository[*models.PersistedTrack] for track caching.
//
// Handles automatic track caching with soft delete support and service-specific lookups.
// Tracks are automatically cached on every fetch to enable cross-service matching via ISRC.
type TrackRepository struct {
	db *sql.DB
}

// NewTrackRepository creates a new TrackRepository with the given database connection
func NewTrackRepository(db *sql.DB) *TrackRepository {
	return &TrackRepository{db: db}
}

// Create inserts a new [models.PersistedTrack] into the database with generated ID and sequence
func (r *TrackRepository) Create(track *models.PersistedTrack) error {
	sequence, err := NextSequence(r.db, "tracks")
	if err != nil {
		return fmt.Errorf("failed to generate sequence: %w", err)
	}

	id := shared.GenerateID()
	track.SetID(id)

	if err := track.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO tracks (id, sequence, service, service_id, title, artist, album, duration, isrc, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.Exec(query,
		id,
		sequence,
		track.Service(),
		track.ServiceID(),
		track.Title(),
		track.Artist(),
		track.Album(),
		track.Duration(),
		track.ISRC(),
		track.CreatedAt(),
		track.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert track: %w", err)
	}

	return nil
}

// Get retrieves a track by ID, excluding soft-deleted tracks
func (r *TrackRepository) Get(id string) (*models.PersistedTrack, error) {
	query := `
		SELECT id, sequence, service, service_id, title, artist, album, duration, isrc, created_at, updated_at, deleted_at
		FROM tracks
		WHERE id = ? AND deleted_at IS NULL
	`

	return r.scanOne(r.db.QueryRow(query, id))
}

// GetByServiceID retrieves a track by service and service_id
func (r *TrackRepository) GetByServiceID(service, serviceID string) (*models.PersistedTrack, error) {
	query := `
		SELECT id, sequence, service, service_id, title, artist, album, duration, isrc, created_at, updated_at, deleted_at
		FROM tracks
		WHERE service = ? AND service_id = ? AND deleted_at IS NULL
	`

	return r.scanOne(r.db.QueryRow(query, service, serviceID))
}

// GetByISRC retrieves a track by ISRC code across any service
func (r *TrackRepository) GetByISRC(isrc string) (*models.PersistedTrack, error) {
	query := `
		SELECT id, sequence, service, service_id, title, artist, album, duration, isrc, created_at, updated_at, deleted_at
		FROM tracks
		WHERE isrc = ? AND deleted_at IS NULL
		LIMIT 1
	`

	return r.scanOne(r.db.QueryRow(query, isrc))
}

// Update modifies an existing track in the database
func (r *TrackRepository) Update(track *models.PersistedTrack) error {
	if err := track.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	now := time.Now()
	track.SetUpdatedAt(now)

	query := `
		UPDATE tracks
		SET title = ?, artist = ?, album = ?, duration = ?, isrc = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query,
		track.Title(),
		track.Artist(),
		track.Album(),
		track.Duration(),
		track.ISRC(),
		now,
		track.ID(),
	)
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("track not found or already deleted: %s", track.ID())
	}

	return nil
}

// Delete soft-deletes a track by ID
func (r *TrackRepository) Delete(id string) error {
	now := time.Now()

	query := `
		UPDATE tracks
		SET deleted_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, now, id)
	if err != nil {
		return fmt.Errorf("failed to delete track: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("track not found or already deleted: %s", id)
	}

	return nil
}

// List retrieves all tracks matching the given criteria, excluding soft-deleted tracks
func (r *TrackRepository) List(criteria map[string]any) ([]*models.PersistedTrack, error) {
	query := `
		SELECT id, sequence, service, service_id, title, artist, album, duration, isrc, created_at, updated_at, deleted_at
		FROM tracks
		WHERE deleted_at IS NULL
	`

	args := []any{}

	if service, ok := criteria["service"].(string); ok && service != "" {
		query += " AND service = ?"
		args = append(args, service)
	}

	if isrc, ok := criteria["isrc"].(string); ok && isrc != "" {
		query += " AND isrc = ?"
		args = append(args, isrc)
	}

	query += " ORDER BY sequence ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tracks: %w", err)
	}
	defer rows.Close()

	var tracks []*models.PersistedTrack
	for rows.Next() {
		track, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return tracks, nil
}

// scanOne scans a single [sql.Row] into a [models.PersistedTrack]
func (r *TrackRepository) scanOne(row *sql.Row) (*models.PersistedTrack, error) {
	var (
		id        string
		sequence  int
		service   string
		serviceID string
		title     string
		artist    string
		album     string
		duration  int
		isrc      string
		createdAt time.Time
		updatedAt time.Time
		deletedAt sql.NullTime
	)

	err := row.Scan(&id, &sequence, &service, &serviceID, &title, &artist, &album, &duration, &isrc, &createdAt, &updatedAt, &deletedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("track not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan track: %w", err)
	}

	dto := models.Track{
		ID:       serviceID,
		Title:    title,
		Artist:   artist,
		Album:    album,
		Duration: duration,
		ISRC:     isrc,
	}

	track := models.NewPersistedTrack(sequence, service, serviceID, dto)
	track.SetID(id)
	track.SetUpdatedAt(updatedAt)
	if deletedAt.Valid {
		track.SetDeletedAt(&deletedAt.Time)
	}

	return track, nil
}

// scanRow scans a row from [sql.Rows] into a [models.PersistedTrack]
func (r *TrackRepository) scanRow(rows *sql.Rows) (*models.PersistedTrack, error) {
	var (
		id        string
		sequence  int
		service   string
		serviceID string
		title     string
		artist    string
		album     string
		duration  int
		isrc      string
		createdAt time.Time
		updatedAt time.Time
		deletedAt sql.NullTime
	)

	err := rows.Scan(&id, &sequence, &service, &serviceID, &title, &artist, &album, &duration, &isrc, &createdAt, &updatedAt, &deletedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan track: %w", err)
	}

	dto := models.Track{
		ID:       serviceID,
		Title:    title,
		Artist:   artist,
		Album:    album,
		Duration: duration,
		ISRC:     isrc,
	}

	track := models.NewPersistedTrack(sequence, service, serviceID, dto)
	track.SetID(id)
	track.SetUpdatedAt(updatedAt)
	if deletedAt.Valid {
		track.SetDeletedAt(&deletedAt.Time)
	}

	return track, nil
}
