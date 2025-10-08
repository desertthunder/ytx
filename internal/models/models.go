// package models defines the data model for the song migration web service
package models

import (
	"fmt"
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

// User represents a user account in the persistence layer with authentication tokens, preferences, and migration history.
type User struct {
	id        string
	sequence  int
	email     string
	name      string
	createdAt time.Time
	updatedAt time.Time
	deletedAt *time.Time
}

func (u *User) ID() string           { return u.id }
func (u *User) CreatedAt() time.Time { return u.createdAt }
func (u *User) UpdatedAt() time.Time { return u.updatedAt }

// Validate checks if the user's data is valid
func (u *User) Validate() error {
	if u.id == "" {
		return ErrInvalidModel
	}
	if u.email == "" {
		return ErrInvalidModel
	}
	return nil
}

// NewUser creates a new User instance with generated ID and timestamps
func NewUser(sequence int, email, name string) *User {
	now := time.Now()
	return &User{
		sequence:  sequence,
		email:     email,
		name:      name,
		createdAt: now,
		updatedAt: now,
	}
}

func (u *User) Email() string { return u.email }
func (u *User) Name() string  { return u.name }
func (u *User) Sequence() int { return u.sequence }

// DeletedAt returns when this user was soft deleted (nil if not deleted)
func (u *User) DeletedAt() *time.Time { return u.deletedAt }

// SetDeletedAt marks the user as soft deleted (used by repository during deletion)
func (u *User) SetDeletedAt(t *time.Time) { u.deletedAt = t }

// SetID sets the user's ID (used by repository during creation)
func (u *User) SetID(id string)          { u.id = id }
func (u *User) SetUpdatedAt(t time.Time) { u.updatedAt = t }

// PersistedPlaylist represents a playlist cached in the persistence layer with service metadata, user ownership, and track relationships.
type PersistedPlaylist struct {
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
	deletedAt   *time.Time
}

func (p *PersistedPlaylist) ID() string           { return p.id }
func (p *PersistedPlaylist) CreatedAt() time.Time { return p.createdAt }
func (p *PersistedPlaylist) UpdatedAt() time.Time { return p.updatedAt }
func (p *PersistedPlaylist) Validate() error {
	if p.id == "" {
		return ErrInvalidModel
	}
	if p.service == "" || p.serviceID == "" {
		return ErrInvalidModel
	}
	if p.userID == "" {
		return ErrInvalidModel
	}
	if p.name == "" {
		return ErrInvalidModel
	}
	return nil
}

// NewPersistedPlaylist creates a new PersistedPlaylist from a Playlist DTO
func NewPersistedPlaylist(sequence int, service, serviceID, userID string, playlist Playlist) *PersistedPlaylist {
	now := time.Now()
	return &PersistedPlaylist{
		sequence:    sequence,
		service:     service,
		serviceID:   serviceID,
		userID:      userID,
		name:        playlist.Name,
		description: playlist.Description,
		trackCount:  playlist.TrackCount,
		public:      playlist.Public,
		createdAt:   now,
		updatedAt:   now,
	}
}

// Service returns the music service name (spotify, youtube)
func (p *PersistedPlaylist) Service() string { return p.service }

// ServiceID returns the service-specific playlist ID
func (p *PersistedPlaylist) ServiceID() string { return p.serviceID }

// UserID returns the owning user's ID
func (p *PersistedPlaylist) UserID() string { return p.userID }

func (p *PersistedPlaylist) Name() string        { return p.name }
func (p *PersistedPlaylist) Description() string { return p.description }
func (p *PersistedPlaylist) TrackCount() int     { return p.trackCount }
func (p *PersistedPlaylist) Public() bool        { return p.public }
func (p *PersistedPlaylist) Sequence() int       { return p.sequence }

// DeletedAt returns when this playlist was soft deleted (nil if not deleted)
func (p *PersistedPlaylist) DeletedAt() *time.Time { return p.deletedAt }

func (p *PersistedPlaylist) SetID(id string)           { p.id = id }
func (p *PersistedPlaylist) SetUpdatedAt(t time.Time)  { p.updatedAt = t }
func (p *PersistedPlaylist) SetDeletedAt(t *time.Time) { p.deletedAt = t }

// ToPlaylist converts a PersistedPlaylist to a Playlist DTO
func (p *PersistedPlaylist) ToPlaylist() Playlist {
	return Playlist{
		ID:          p.serviceID,
		Name:        p.name,
		Description: p.description,
		TrackCount:  p.trackCount,
		Public:      p.public,
	}
}

// PersistedTrack represents a track cached in the persistence layer with service metadata and ISRC for cross-service matching.
type PersistedTrack struct {
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
	deletedAt *time.Time
}

func (t *PersistedTrack) ID() string           { return t.id }
func (t *PersistedTrack) CreatedAt() time.Time { return t.createdAt }
func (t *PersistedTrack) UpdatedAt() time.Time { return t.updatedAt }

// Validate checks if the track's data is valid
func (t *PersistedTrack) Validate() error {
	if t.id == "" {
		return ErrInvalidModel
	}
	if t.service == "" || t.serviceID == "" {
		return ErrInvalidModel
	}
	if t.title == "" || t.artist == "" {
		return ErrInvalidModel
	}
	return nil
}

// NewPersistedTrack creates a new PersistedTrack from a Track DTO
func NewPersistedTrack(sequence int, service, serviceID string, track Track) *PersistedTrack {
	now := time.Now()
	return &PersistedTrack{
		sequence:  sequence,
		service:   service,
		serviceID: serviceID,
		title:     track.Title,
		artist:    track.Artist,
		album:     track.Album,
		duration:  track.Duration,
		isrc:      track.ISRC,
		createdAt: now,
		updatedAt: now,
	}
}

// Service returns the music service name (spotify, youtube)
func (t *PersistedTrack) Service() string { return t.service }

// ServiceID returns the service-specific track ID
func (t *PersistedTrack) ServiceID() string { return t.serviceID }

func (t *PersistedTrack) Title() string  { return t.title }
func (t *PersistedTrack) Artist() string { return t.artist }
func (t *PersistedTrack) Album() string  { return t.album }
func (t *PersistedTrack) Duration() int  { return t.duration }
func (t *PersistedTrack) Sequence() int  { return t.sequence }

// ISRC returns the International Standard Recording Code
func (t *PersistedTrack) ISRC() string { return t.isrc }

// DeletedAt returns when this track was soft deleted (nil if not deleted)
func (t *PersistedTrack) DeletedAt() *time.Time { return t.deletedAt }

func (t *PersistedTrack) SetID(id string)            { t.id = id }
func (t *PersistedTrack) SetUpdatedAt(t2 time.Time)  { t.updatedAt = t2 }
func (t *PersistedTrack) SetDeletedAt(t2 *time.Time) { t.deletedAt = t2 }

// ToTrack converts a PersistedTrack to a Track DTO
func (t *PersistedTrack) ToTrack() Track {
	return Track{
		ID:       t.serviceID,
		Title:    t.title,
		Artist:   t.artist,
		Album:    t.album,
		Duration: t.duration,
		ISRC:     t.isrc,
	}
}

// PlaylistTrack represents a track within a playlist with ordering via position field.
type PlaylistTrack struct {
	id         string
	sequence   int
	playlistID string
	trackID    string
	position   int
	createdAt  time.Time
	deletedAt  *time.Time
}

func (pt *PlaylistTrack) ID() string           { return pt.id }
func (pt *PlaylistTrack) CreatedAt() time.Time { return pt.createdAt }
func (pt *PlaylistTrack) UpdatedAt() time.Time { return pt.createdAt }

// Validate checks if the playlist-track's data is valid
func (pt *PlaylistTrack) Validate() error {
	if pt.id == "" {
		return ErrInvalidModel
	}
	if pt.playlistID == "" || pt.trackID == "" {
		return ErrInvalidModel
	}
	return nil
}

func (pt *PlaylistTrack) PlaylistID() string { return pt.playlistID }
func (pt *PlaylistTrack) TrackID() string    { return pt.trackID }
func (pt *PlaylistTrack) Position() int      { return pt.position }
func (pt *PlaylistTrack) Sequence() int      { return pt.sequence }

func (pt *PlaylistTrack) SetID(id string) { pt.id = id }

// DeletedAt returns when this playlist-track was soft deleted (nil if not deleted)
func (pt *PlaylistTrack) DeletedAt() *time.Time { return pt.deletedAt }

// SetDeletedAt marks the playlist-track as soft deleted (used by repository during deletion)
func (pt *PlaylistTrack) SetDeletedAt(t *time.Time) { pt.deletedAt = t }

// NewPlaylistTrack creates a new PlaylistTrack junction record
func NewPlaylistTrack(sequence int, playlistID, trackID string, position int) *PlaylistTrack {
	return &PlaylistTrack{
		sequence:   sequence,
		playlistID: playlistID,
		trackID:    trackID,
		position:   position,
		createdAt:  time.Now(),
	}
}

// MigrationJob represents a playlist migration operation tracking source/target playlists, progress metrics, and status.
type MigrationJob struct {
	id               string
	sequence         int
	userID           string
	sourceService    string
	sourcePlaylistID string
	targetService    string
	targetPlaylistID string
	status           string
	tracksTotal      int
	tracksMigrated   int
	tracksFailed     int
	errorMessage     string
	startedAt        *time.Time
	completedAt      *time.Time
	createdAt        time.Time
	updatedAt        time.Time
	deletedAt        *time.Time
}

func (m *MigrationJob) ID() string           { return m.id }
func (m *MigrationJob) CreatedAt() time.Time { return m.createdAt }
func (m *MigrationJob) UpdatedAt() time.Time { return m.updatedAt }

// Validate checks if the migration's data is valid
func (m *MigrationJob) Validate() error {
	if m.id == "" {
		return ErrInvalidModel
	}
	if m.userID == "" {
		return ErrInvalidModel
	}
	if m.sourceService == "" || m.sourcePlaylistID == "" {
		return ErrInvalidModel
	}
	if m.targetService == "" {
		return ErrInvalidModel
	}
	return nil
}

// NewMigrationJob creates a new MigrationJob with pending status
func NewMigrationJob(sequence int, userID, sourceService, sourcePlaylistID, targetService string) *MigrationJob {
	now := time.Now()
	return &MigrationJob{
		sequence:         sequence,
		userID:           userID,
		sourceService:    sourceService,
		sourcePlaylistID: sourcePlaylistID,
		targetService:    targetService,
		status:           "pending",
		createdAt:        now,
		updatedAt:        now,
	}
}

// UserID returns the owning user's ID
func (m *MigrationJob) UserID() string { return m.userID }

// SourceService returns the source music service name
func (m *MigrationJob) SourceService() string { return m.sourceService }

// SourcePlaylistID returns the source playlist's ID
func (m *MigrationJob) SourcePlaylistID() string { return m.sourcePlaylistID }

// TargetService returns the target music service name
func (m *MigrationJob) TargetService() string { return m.targetService }

func (m *MigrationJob) TargetPlaylistID() string { return m.targetPlaylistID }
func (m *MigrationJob) Status() string           { return m.status }
func (m *MigrationJob) TracksTotal() int         { return m.tracksTotal }
func (m *MigrationJob) TracksMigrated() int      { return m.tracksMigrated }
func (m *MigrationJob) TracksFailed() int        { return m.tracksFailed }
func (m *MigrationJob) ErrorMessage() string     { return m.errorMessage }
func (m *MigrationJob) StartedAt() *time.Time    { return m.startedAt }
func (m *MigrationJob) CompletedAt() *time.Time  { return m.completedAt }
func (m *MigrationJob) Sequence() int            { return m.sequence }

// DeletedAt returns when this migration was soft deleted (nil if not deleted)
func (m *MigrationJob) DeletedAt() *time.Time { return m.deletedAt }

// SetDeletedAt marks the migration as soft deleted (used by repository during deletion)
func (m *MigrationJob) SetDeletedAt(t *time.Time) { m.deletedAt = t }

func (m *MigrationJob) SetID(id string)                { m.id = id }
func (m *MigrationJob) SetUpdatedAt(t time.Time)       { m.updatedAt = t }
func (m *MigrationJob) SetTargetPlaylistID(id string)  { m.targetPlaylistID = id }
func (m *MigrationJob) SetStatus(status string)        { m.status = status }
func (m *MigrationJob) SetTracksTotal(total int)       { m.tracksTotal = total }
func (m *MigrationJob) SetTracksMigrated(migrated int) { m.tracksMigrated = migrated }
func (m *MigrationJob) SetTracksFailed(failed int)     { m.tracksFailed = failed }
func (m *MigrationJob) SetErrorMessage(msg string)     { m.errorMessage = msg }
func (m *MigrationJob) SetStartedAt(t *time.Time)      { m.startedAt = t }
func (m *MigrationJob) SetCompletedAt(t *time.Time)    { m.completedAt = t }

// ErrInvalidModel is returned when a model fails validation
var ErrInvalidModel = fmt.Errorf("invalid model")
