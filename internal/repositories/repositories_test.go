package repositories

import (
	"database/sql"
	"testing"

	"github.com/desertthunder/ytx/internal/models"
	"github.com/desertthunder/ytx/internal/shared"
)

// setupTestDB creates an in-memory SQLite database with migrations applied
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := shared.NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	if err := shared.RunMigrations(db); err != nil {
		db.Close()
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

func TestUserRepository(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		repo := NewUserRepository(db)
		user := models.NewUser(0, "test@example.com", "Test User")

		err := repo.Create(user)
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		if user.ID() == "" {
			t.Error("user ID should be set after creation")
		}
	})

	t.Run("Get", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		repo := NewUserRepository(db)
		user := models.NewUser(0, "test@example.com", "Test User")

		if err := repo.Create(user); err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		retrieved, err := repo.Get(user.ID())
		if err != nil {
			t.Fatalf("failed to get user: %v", err)
		}

		if retrieved.ID() != user.ID() {
			t.Errorf("expected ID %s, got %s", user.ID(), retrieved.ID())
		}

		if retrieved.Email() != user.Email() {
			t.Errorf("expected email %s, got %s", user.Email(), retrieved.Email())
		}
	})

	t.Run("Update", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		repo := NewUserRepository(db)
		user := models.NewUser(0, "test@example.com", "Test User")

		if err := repo.Create(user); err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		retrieved, err := repo.Get(user.ID())
		if err != nil {
			t.Fatalf("failed to get user: %v", err)
		}

		if err := repo.Update(retrieved); err != nil {
			t.Fatalf("failed to update user: %v", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		repo := NewUserRepository(db)
		user := models.NewUser(0, "test@example.com", "Test User")

		if err := repo.Create(user); err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		if err := repo.Delete(user.ID()); err != nil {
			t.Fatalf("failed to delete user: %v", err)
		}

		_, err := repo.Get(user.ID())
		if err == nil {
			t.Error("expected error when getting deleted user")
		}
	})

	t.Run("List", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		users := []*models.User{
			models.NewUser(0, "user1@example.com", "User One"),
			models.NewUser(0, "user2@example.com", "User Two"),
			models.NewUser(0, "user3@example.com", "User Three"),
		}

		for _, user := range users {
			if err := repo.Create(user); err != nil {
				t.Fatalf("failed to create user: %v", err)
			}
		}

		retrieved, err := repo.List(map[string]any{})
		if err != nil {
			t.Fatalf("failed to list users: %v", err)
		}

		if len(retrieved) != 3 {
			t.Errorf("expected 3 users, got %d", len(retrieved))
		}

		filtered, err := repo.List(map[string]any{"email": "user2@example.com"})
		if err != nil {
			t.Fatalf("failed to list filtered users: %v", err)
		}

		if len(filtered) != 1 {
			t.Errorf("expected 1 user, got %d", len(filtered))
		}

		if len(filtered) > 0 && filtered[0].Email() != "user2@example.com" {
			t.Errorf("expected user2@example.com, got %s", filtered[0].Email())
		}
	})
}

func TestTrackRepository(t *testing.T) {
	t.Run("Create & Get", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		repo := NewTrackRepository(db)
		trackDTO := models.Track{
			ID:       "spotify123",
			Title:    "Test Song",
			Artist:   "Test Artist",
			Album:    "Test Album",
			Duration: 180,
			ISRC:     "USTEST1234567",
		}

		track := models.NewPersistedTrack(0, "spotify", "spotify123", trackDTO)

		if err := repo.Create(track); err != nil {
			t.Fatalf("failed to create track: %v", err)
		}

		retrieved, err := repo.GetByServiceID("spotify", "spotify123")
		if err != nil {
			t.Fatalf("failed to get track: %v", err)
		}

		if retrieved.Title() != "Test Song" {
			t.Errorf("expected title 'Test Song', got %s", retrieved.Title())
		}

		if retrieved.ISRC() != "USTEST1234567" {
			t.Errorf("expected ISRC 'USTEST1234567', got %s", retrieved.ISRC())
		}
	})

	t.Run("GetByISRC", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		repo := NewTrackRepository(db)

		spotifyTrack := models.NewPersistedTrack(0, "spotify", "spotify123", models.Track{
			ID:     "spotify123",
			Title:  "Test Song",
			Artist: "Test Artist",
			ISRC:   "USTEST1234567",
		})

		if err := repo.Create(spotifyTrack); err != nil {
			t.Fatalf("failed to create Spotify track: %v", err)
		}

		retrieved, err := repo.GetByISRC("USTEST1234567")
		if err != nil {
			t.Fatalf("failed to get track by ISRC: %v", err)
		}

		if retrieved.ISRC() != "USTEST1234567" {
			t.Errorf("expected ISRC 'USTEST1234567', got %s", retrieved.ISRC())
		}
	})
}

func TestTrackCacheAdapter_CacheTrack(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTrackRepository(db)
	adapter := NewTrackCacheAdapter(repo)

	trackDTO := models.Track{
		ID:       "spotify123",
		Title:    "Test Song",
		Artist:   "Test Artist",
		Album:    "Test Album",
		Duration: 180,
		ISRC:     "USTEST1234567",
	}

	if err := adapter.CacheTrack("spotify", "spotify123", trackDTO); err != nil {
		t.Fatalf("failed to cache track: %v", err)
	}

	if err := adapter.CacheTrack("spotify", "spotify123", trackDTO); err != nil {
		t.Fatalf("caching duplicate track should not error: %v", err)
	}

	retrieved, err := repo.GetByServiceID("spotify", "spotify123")
	if err != nil {
		t.Fatalf("failed to retrieve cached track: %v", err)
	}

	if retrieved.Title() != "Test Song" {
		t.Errorf("expected title 'Test Song', got %s", retrieved.Title())
	}
}

func TestPlaylistRepository_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	user := models.NewUser(0, "test@example.com", "Test User")
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create playlist
	playlistRepo := NewPlaylistRepository(db)
	playlistDTO := models.Playlist{
		ID:          "spotify123",
		Name:        "Test Playlist",
		Description: "Test Description",
		TrackCount:  10,
		Public:      true,
	}

	playlist := models.NewPersistedPlaylist(0, "spotify", "spotify123", user.ID(), playlistDTO)

	if err := playlistRepo.Create(playlist); err != nil {
		t.Fatalf("failed to create playlist: %v", err)
	}

	retrieved, err := playlistRepo.GetByServiceID("spotify", "spotify123")
	if err != nil {
		t.Fatalf("failed to get playlist: %v", err)
	}

	if retrieved.Name() != "Test Playlist" {
		t.Errorf("expected name 'Test Playlist', got %s", retrieved.Name())
	}

	if retrieved.UserID() != user.ID() {
		t.Errorf("expected user ID %s, got %s", user.ID(), retrieved.UserID())
	}
}

func TestMigrationRepository_CreateAndUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	user := models.NewUser(0, "test@example.com", "Test User")
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	playlistRepo := NewPlaylistRepository(db)
	sourcePlaylist := models.NewPersistedPlaylist(0, "spotify", "spotifyid123", user.ID(), models.Playlist{
		ID:          "spotifyid123",
		Name:        "Source Playlist",
		Description: "Test source",
		TrackCount:  10,
		Public:      false,
	})
	if err := playlistRepo.Create(sourcePlaylist); err != nil {
		t.Fatalf("failed to create source playlist: %v", err)
	}

	migrationRepo := NewMigrationRepository(db)
	migration := models.NewMigrationJob(0, user.ID(), "spotify", sourcePlaylist.ID(), "youtube")

	if err := migrationRepo.Create(migration); err != nil {
		t.Fatalf("failed to create migration: %v", err)
	}

	if migration.Status() != "pending" {
		t.Errorf("expected status 'pending', got %s", migration.Status())
	}

	migration.SetStatus("in_progress")
	migration.SetTracksTotal(10)
	migration.SetTracksMigrated(5)

	if err := migrationRepo.Update(migration); err != nil {
		t.Fatalf("failed to update migration: %v", err)
	}

	retrieved, err := migrationRepo.Get(migration.ID())
	if err != nil {
		t.Fatalf("failed to get migration: %v", err)
	}

	if retrieved.Status() != "in_progress" {
		t.Errorf("expected status 'in_progress', got %s", retrieved.Status())
	}

	if retrieved.TracksTotal() != 10 {
		t.Errorf("expected 10 total tracks, got %d", retrieved.TracksTotal())
	}

	if retrieved.TracksMigrated() != 5 {
		t.Errorf("expected 5 migrated tracks, got %d", retrieved.TracksMigrated())
	}
}

func TestNextSequence(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	seq1, err := NextSequence(db, "users")
	if err != nil {
		t.Fatalf("failed to get first sequence: %v", err)
	}

	if seq1 != 1 {
		t.Errorf("expected first sequence to be 1, got %d", seq1)
	}

	// Get second sequence
	seq2, err := NextSequence(db, "users")
	if err != nil {
		t.Fatalf("failed to get second sequence: %v", err)
	}

	if seq2 != 2 {
		t.Errorf("expected second sequence to be 2, got %d", seq2)
	}

	trackSeq, err := NextSequence(db, "tracks")
	if err != nil {
		t.Fatalf("failed to get track sequence: %v", err)
	}

	if trackSeq != 1 {
		t.Errorf("expected first track sequence to be 1, got %d", trackSeq)
	}
}
