package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/desertthunder/ytx/internal/models"
	"github.com/desertthunder/ytx/internal/shared"
)

// MigrationRepository implements models.Repository[*models.MigrationJob] for migration tracking.
//
// Handles migration job CRUD operations with soft delete support and status-based queries.
type MigrationRepository struct {
	db *sql.DB
}

// NewMigrationRepository creates a new MigrationRepository with the given database connection
func NewMigrationRepository(db *sql.DB) *MigrationRepository {
	return &MigrationRepository{db: db}
}

// Create inserts a new migration job into the database with generated ID and sequence
func (r *MigrationRepository) Create(migration *models.MigrationJob) error {
	sequence, err := NextSequence(r.db, "migrations")
	if err != nil {
		return fmt.Errorf("failed to generate sequence: %w", err)
	}

	id := shared.GenerateID()
	migration.SetID(id)

	if err := migration.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO migrations (
			id, sequence, user_id, source_service, source_playlist_id,
			target_service, target_playlist_id, status, tracks_total,
			tracks_migrated, tracks_failed, error_message, started_at,
			completed_at, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var targetPlaylistID any = migration.TargetPlaylistID()
	if targetPlaylistID == "" {
		targetPlaylistID = nil
	}

	var errorMessage any = migration.ErrorMessage()
	if errorMessage == "" {
		errorMessage = nil
	}

	_, err = r.db.Exec(query,
		id,
		sequence,
		migration.UserID(),
		migration.SourceService(),
		migration.SourcePlaylistID(),
		migration.TargetService(),
		targetPlaylistID,
		migration.Status(),
		migration.TracksTotal(),
		migration.TracksMigrated(),
		migration.TracksFailed(),
		errorMessage,
		migration.StartedAt(),
		migration.CompletedAt(),
		migration.CreatedAt(),
		migration.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert migration: %w", err)
	}

	return nil
}

// Get retrieves a migration job by ID, excluding soft-deleted migrations
func (r *MigrationRepository) Get(id string) (*models.MigrationJob, error) {
	query := `
		SELECT
			id, sequence, user_id, source_service, source_playlist_id,
			target_service, target_playlist_id, status, tracks_total,
			tracks_migrated, tracks_failed, error_message, started_at,
			completed_at, created_at, updated_at, deleted_at
		FROM migrations
		WHERE id = ? AND deleted_at IS NULL
	`

	return r.scanOne(r.db.QueryRow(query, id))
}

// Update modifies an existing migration job in the database
func (r *MigrationRepository) Update(migration *models.MigrationJob) error {
	if err := migration.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	now := time.Now()
	migration.SetUpdatedAt(now)

	query := `
		UPDATE migrations
		SET target_playlist_id = ?, status = ?, tracks_total = ?,
			tracks_migrated = ?, tracks_failed = ?, error_message = ?,
			started_at = ?, completed_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	var targetPlaylistID any = migration.TargetPlaylistID()
	if targetPlaylistID == "" {
		targetPlaylistID = nil
	}

	var errorMessage any = migration.ErrorMessage()
	if errorMessage == "" {
		errorMessage = nil
	}

	result, err := r.db.Exec(query,
		targetPlaylistID,
		migration.Status(),
		migration.TracksTotal(),
		migration.TracksMigrated(),
		migration.TracksFailed(),
		errorMessage,
		migration.StartedAt(),
		migration.CompletedAt(),
		now,
		migration.ID(),
	)
	if err != nil {
		return fmt.Errorf("failed to update migration: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("migration not found or already deleted: %s", migration.ID())
	}

	return nil
}

// Delete soft-deletes a migration job by ID
func (r *MigrationRepository) Delete(id string) error {
	now := time.Now()

	query := `
		UPDATE migrations
		SET deleted_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, now, id)
	if err != nil {
		return fmt.Errorf("failed to delete migration: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("migration not found or already deleted: %s", id)
	}

	return nil
}

// List retrieves all migration jobs matching the given criteria, excluding soft-deleted migrations
func (r *MigrationRepository) List(criteria map[string]any) ([]*models.MigrationJob, error) {
	query := `
		SELECT
			id, sequence, user_id, source_service, source_playlist_id,
			target_service, target_playlist_id, status, tracks_total,
			tracks_migrated, tracks_failed, error_message, started_at,
			completed_at, created_at, updated_at, deleted_at
		FROM migrations
		WHERE deleted_at IS NULL
	`

	args := []any{}

	if userID, ok := criteria["user_id"].(string); ok && userID != "" {
		query += " AND user_id = ?"
		args = append(args, userID)
	}

	if status, ok := criteria["status"].(string); ok && status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	if sourceService, ok := criteria["source_service"].(string); ok && sourceService != "" {
		query += " AND source_service = ?"
		args = append(args, sourceService)
	}

	if targetService, ok := criteria["target_service"].(string); ok && targetService != "" {
		query += " AND target_service = ?"
		args = append(args, targetService)
	}

	query += " ORDER BY sequence DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var migrations []*models.MigrationJob
	for rows.Next() {
		migration, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return migrations, nil
}

// scanOne scans a single [sql.Row] into a [models.MigrationJob]
func (r *MigrationRepository) scanOne(row *sql.Row) (*models.MigrationJob, error) {
	var (
		id               string
		sequence         int
		userID           string
		sourceService    string
		sourcePlaylistID string
		targetService    string
		targetPlaylistID sql.NullString
		status           string
		tracksTotal      int
		tracksMigrated   int
		tracksFailed     int
		errorMessage     sql.NullString
		startedAt        sql.NullTime
		completedAt      sql.NullTime
		createdAt        time.Time
		updatedAt        time.Time
		deletedAt        sql.NullTime
	)

	err := row.Scan(
		&id, &sequence, &userID, &sourceService, &sourcePlaylistID,
		&targetService, &targetPlaylistID, &status, &tracksTotal,
		&tracksMigrated, &tracksFailed, &errorMessage, &startedAt,
		&completedAt, &createdAt, &updatedAt, &deletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("migration not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan migration: %w", err)
	}

	migration := models.NewMigrationJob(sequence, userID, sourceService, sourcePlaylistID, targetService)
	migration.SetID(id)
	migration.SetUpdatedAt(updatedAt)

	if targetPlaylistID.Valid {
		migration.SetTargetPlaylistID(targetPlaylistID.String)
	}
	migration.SetStatus(status)
	migration.SetTracksTotal(tracksTotal)
	migration.SetTracksMigrated(tracksMigrated)
	migration.SetTracksFailed(tracksFailed)
	if errorMessage.Valid {
		migration.SetErrorMessage(errorMessage.String)
	}
	if startedAt.Valid {
		migration.SetStartedAt(&startedAt.Time)
	}
	if completedAt.Valid {
		migration.SetCompletedAt(&completedAt.Time)
	}
	if deletedAt.Valid {
		migration.SetDeletedAt(&deletedAt.Time)
	}

	return migration, nil
}

// scanRow scans a row from [sql.Rows] into a [models.MigrationJob]
func (r *MigrationRepository) scanRow(rows *sql.Rows) (*models.MigrationJob, error) {
	var (
		id               string
		sequence         int
		userID           string
		sourceService    string
		sourcePlaylistID string
		targetService    string
		targetPlaylistID sql.NullString
		status           string
		tracksTotal      int
		tracksMigrated   int
		tracksFailed     int
		errorMessage     sql.NullString
		startedAt        sql.NullTime
		completedAt      sql.NullTime
		createdAt        time.Time
		updatedAt        time.Time
		deletedAt        sql.NullTime
	)

	err := rows.Scan(
		&id, &sequence, &userID, &sourceService, &sourcePlaylistID,
		&targetService, &targetPlaylistID, &status, &tracksTotal,
		&tracksMigrated, &tracksFailed, &errorMessage, &startedAt,
		&completedAt, &createdAt, &updatedAt, &deletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan migration: %w", err)
	}

	migration := models.NewMigrationJob(sequence, userID, sourceService, sourcePlaylistID, targetService)
	migration.SetID(id)
	migration.SetUpdatedAt(updatedAt)

	if targetPlaylistID.Valid {
		migration.SetTargetPlaylistID(targetPlaylistID.String)
	}
	migration.SetStatus(status)
	migration.SetTracksTotal(tracksTotal)
	migration.SetTracksMigrated(tracksMigrated)
	migration.SetTracksFailed(tracksFailed)
	if errorMessage.Valid {
		migration.SetErrorMessage(errorMessage.String)
	}
	if startedAt.Valid {
		migration.SetStartedAt(&startedAt.Time)
	}
	if completedAt.Valid {
		migration.SetCompletedAt(&completedAt.Time)
	}
	if deletedAt.Valid {
		migration.SetDeletedAt(&deletedAt.Time)
	}

	return migration, nil
}
