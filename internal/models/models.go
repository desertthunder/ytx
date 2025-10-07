// package models defines the data model for the song migration web service
package models

import (
	"time"
)

// Model defines the base interface for all persistent models in the song migration service.
// Implementations include User, Migration, MigrationJob, etc.
type Model interface {
	ID() string           // ID returns the unique identifier for this model
	CreatedAt() time.Time // CreatedAt returns when this model was created
	UpdatedAt() time.Time // UpdatedAt returns when this model was last updated
	Validate() error      // Validate checks if the model's data is valid and returns an error if not
}

// Repository defines the interface for data access operations.
// Implementations handle database interactions for specific model types.
type Repository[T Model] interface {
	Create(model T) error                      // Create inserts a new model into the database
	Get(id string) (T, error)                  // Get retrieves a model by its ID
	Update(model T) error                      // Update modifies an existing model in the database
	Delete(id string) error                    // Delete removes a model from the database by its ID
	List(criteria map[string]any) ([]T, error) // List retrieves all models matching the given criteria
}

// Playlist represents a music playlist from any service
type Playlist struct {
	ID          string
	Name        string
	Description string
	TrackCount  int
	Public      bool
}

// PlaylistExport represents a playlist with all its [Track] objects for migration
type PlaylistExport struct {
	Playlist Playlist
	Tracks   []Track
}

// Track represents a music track from any service
type Track struct {
	ID       string
	Title    string
	Artist   string
	Album    string
	Duration int    // Duration in seconds
	ISRC     string // International Standard Recording Code for matching
}
