package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/desertthunder/ytx/internal/models"
	"github.com/desertthunder/ytx/internal/shared"
)

// UserRepository implements [models.Repository] for user [models.User] persistence.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new [UserRepository] with the given database connection
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user into the database with generated ID and sequence
func (r *UserRepository) Create(user *models.User) error {
	sequence, err := NextSequence(r.db, "users")
	if err != nil {
		return fmt.Errorf("failed to generate sequence: %w", err)
	}

	id := shared.GenerateID()
	user.SetID(id)

	if err := user.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO users (id, sequence, email, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.Exec(query, id, sequence, user.Email(), user.Name(), user.CreatedAt(), user.UpdatedAt())
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

// Get retrieves a user by ID, excluding soft-deleted users
func (r *UserRepository) Get(id string) (*models.User, error) {
	query := `
		SELECT id, sequence, email, name, created_at, updated_at, deleted_at
		FROM users
		WHERE id = ? AND deleted_at IS NULL
	`

	var (
		userID    string
		sequence  int
		email     string
		name      string
		createdAt time.Time
		updatedAt time.Time
		deletedAt sql.NullTime
	)

	err := r.db.QueryRow(query, id).Scan(&userID, &sequence, &email, &name, &createdAt, &updatedAt, &deletedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	user := models.NewUser(sequence, email, name)
	user.SetID(userID)
	user.SetUpdatedAt(updatedAt)
	if deletedAt.Valid {
		user.SetDeletedAt(&deletedAt.Time)
	}

	return user, nil
}

// Update modifies an existing user in the database
func (r *UserRepository) Update(user *models.User) error {
	if err := user.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	now := time.Now()
	user.SetUpdatedAt(now)

	query := `
		UPDATE users
		SET email = ?, name = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, user.Email(), user.Name(), now, user.ID())
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found or already deleted: %s", user.ID())
	}

	return nil
}

// Delete soft-deletes a user by ID
func (r *UserRepository) Delete(id string) error {
	now := time.Now()

	query := `
		UPDATE users
		SET deleted_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, now, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found or already deleted: %s", id)
	}

	return nil
}

// List retrieves all users matching the given criteria, excluding soft-deleted users
func (r *UserRepository) List(criteria map[string]any) ([]*models.User, error) {
	query := `
		SELECT id, sequence, email, name, created_at, updated_at, deleted_at
		FROM users
		WHERE deleted_at IS NULL
	`

	args := []any{}

	if email, ok := criteria["email"].(string); ok && email != "" {
		query += " AND email = ?"
		args = append(args, email)
	}

	query += " ORDER BY sequence ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var (
			userID    string
			sequence  int
			email     string
			name      string
			createdAt time.Time
			updatedAt time.Time
			deletedAt sql.NullTime
		)

		err := rows.Scan(&userID, &sequence, &email, &name, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		user := models.NewUser(sequence, email, name)
		user.SetID(userID)
		user.SetUpdatedAt(updatedAt)
		if deletedAt.Valid {
			user.SetDeletedAt(&deletedAt.Time)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return users, nil
}
