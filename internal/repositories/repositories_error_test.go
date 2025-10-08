package repositories

import (
	"fmt"
	"testing"

	"github.com/desertthunder/ytx/internal/models"
)

func TestUserRepositoryErrors(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		t.Run("ValidationError", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			repo := NewUserRepository(db)
			user := models.NewUser(0, "", "Test User")

			user.SetID("test-id")

			if err := repo.Create(user); err == nil {
				t.Fatal("expected validation error for empty email")
			}
		})

		t.Run("DuplicateEmail", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			repo := NewUserRepository(db)
			user1 := models.NewUser(0, "test@example.com", "User One")

			if err := repo.Create(user1); err != nil {
				t.Fatalf("failed to create first user: %v", err)
			}

			user2 := models.NewUser(0, "test@example.com", "User Two")
			err := repo.Create(user2)
			if err == nil {
				t.Fatal("expected error when creating user with duplicate email")
			}
		})

	})
	t.Run("Get", func(t *testing.T) {
		t.Run("NotFound", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			repo := NewUserRepository(db)

			_, err := repo.Get("nonexistent-id")
			if err == nil {
				t.Fatal("expected error when getting nonexistent user")
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("NotFound", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			repo := NewUserRepository(db)
			user := models.NewUser(0, "test@example.com", "Test User")
			user.SetID("nonexistent-id")

			err := repo.Update(user)
			if err == nil {
				t.Fatal("expected error when updating nonexistent user")
			}
		})

		t.Run("Deleted", func(t *testing.T) {
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

			err := repo.Update(user)
			if err == nil {
				t.Fatal("expected error when updating deleted user")
			}
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("NotFound", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			repo := NewUserRepository(db)

			err := repo.Delete("nonexistent-id")
			if err == nil {
				t.Fatal("expected error when deleting nonexistent user")
			}
		})

		t.Run("AlreadyDeleted", func(t *testing.T) {
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

			err := repo.Delete(user.ID())
			if err == nil {
				t.Fatal("expected error when deleting already deleted user")
			}
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("ExcludesDeleted", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			repo := NewUserRepository(db)

			user1 := models.NewUser(0, "user1@example.com", "User One")
			user2 := models.NewUser(0, "user2@example.com", "User Two")

			if err := repo.Create(user1); err != nil {
				t.Fatalf("failed to create user1: %v", err)
			}
			if err := repo.Create(user2); err != nil {
				t.Fatalf("failed to create user2: %v", err)
			}

			if err := repo.Delete(user1.ID()); err != nil {
				t.Fatalf("failed to delete user1: %v", err)
			}

			users, err := repo.List(map[string]any{})
			if err != nil {
				t.Fatalf("failed to list users: %v", err)
			}

			if len(users) != 1 {
				t.Errorf("expected 1 user (excluding deleted), got %d", len(users))
			}

			if len(users) > 0 && users[0].Email() != "user2@example.com" {
				t.Errorf("expected user2@example.com, got %s", users[0].Email())
			}
		})
	})
}

func TestTrackRepositoryErrors(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		t.Run("DuplicateServiceID", func(t *testing.T) {
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

			track1 := models.NewPersistedTrack(0, "spotify", "spotify123", trackDTO)
			if err := repo.Create(track1); err != nil {
				t.Fatalf("failed to create first track: %v", err)
			}

			// Try to create another track with same service+service_id
			track2 := models.NewPersistedTrack(0, "spotify", "spotify123", trackDTO)
			err := repo.Create(track2)
			if err == nil {
				t.Fatal("expected error when creating track with duplicate service+service_id")
			}
		})

		t.Run("ValidationError", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			repo := NewTrackRepository(db)

			trackDTO := models.Track{
				ID:     "spotify123",
				Title:  "",
				Artist: "",
			}
			track := models.NewPersistedTrack(0, "spotify", "spotify123", trackDTO)
			track.SetID("test-id")

			err := repo.Create(track)
			if err == nil {
				t.Fatal("expected validation error for track with empty title and artist")
			}
		})

	})

	t.Run("NotFound errors", func(t *testing.T) {
		t.Run("GetByServiceID", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			repo := NewTrackRepository(db)

			_, err := repo.GetByServiceID("spotify", "nonexistent")
			if err == nil {
				t.Fatal("expected error when getting nonexistent track")
			}
		})

		t.Run("GetByISRC", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			repo := NewTrackRepository(db)

			_, err := repo.GetByISRC("NONEXISTENT")
			if err == nil {
				t.Fatal("expected error when getting track by nonexistent ISRC")
			}
		})

		t.Run("Update", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			repo := NewTrackRepository(db)
			trackDTO := models.Track{
				ID:     "spotify123",
				Title:  "Test Song",
				Artist: "Test Artist",
			}
			track := models.NewPersistedTrack(0, "spotify", "spotify123", trackDTO)
			track.SetID("nonexistent-id")

			err := repo.Update(track)
			if err == nil {
				t.Fatal("expected error when updating nonexistent track")
			}
		})

		t.Run("Delete", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			repo := NewTrackRepository(db)

			err := repo.Delete("nonexistent-id")
			if err == nil {
				t.Fatal("expected error when deleting nonexistent track")
			}
		})
	})
}

func TestPlaylistRepositoryErrors(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		t.Run("DuplicateServiceID", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			userRepo := NewUserRepository(db)
			user := models.NewUser(0, "test@example.com", "Test User")
			if err := userRepo.Create(user); err != nil {
				t.Fatalf("failed to create user: %v", err)
			}

			playlistRepo := NewPlaylistRepository(db)
			playlistDTO := models.Playlist{
				ID:          "spotify123",
				Name:        "Test Playlist",
				Description: "Test Description",
				TrackCount:  10,
				Public:      true,
			}

			playlist1 := models.NewPersistedPlaylist(0, "spotify", "spotify123", user.ID(), playlistDTO)
			if err := playlistRepo.Create(playlist1); err != nil {
				t.Fatalf("failed to create first playlist: %v", err)
			}

			playlist2 := models.NewPersistedPlaylist(0, "spotify", "spotify123", user.ID(), playlistDTO)
			err := playlistRepo.Create(playlist2)
			if err == nil {
				t.Fatal("expected error when creating playlist with duplicate service+service_id")
			}
		})

		t.Run("InvalidUserID", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			playlistRepo := NewPlaylistRepository(db)
			playlistDTO := models.Playlist{
				ID:          "spotify123",
				Name:        "Test Playlist",
				Description: "Test Description",
				TrackCount:  10,
				Public:      true,
			}

			playlist := models.NewPersistedPlaylist(0, "spotify", "spotify123", "nonexistent-user", playlistDTO)
			err := playlistRepo.Create(playlist)
			if err == nil {
				t.Fatal("expected error when creating playlist with invalid user_id")
			}
		})
	})

	t.Run("NotFound errors", func(t *testing.T) {
		t.Run("GetByServiceID", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			playlistRepo := NewPlaylistRepository(db)

			_, err := playlistRepo.GetByServiceID("spotify", "nonexistent")
			if err == nil {
				t.Fatal("expected error when getting nonexistent playlist")
			}
		})

		t.Run("Update", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			userRepo := NewUserRepository(db)
			user := models.NewUser(0, "test@example.com", "Test User")
			if err := userRepo.Create(user); err != nil {
				t.Fatalf("failed to create user: %v", err)
			}

			playlistRepo := NewPlaylistRepository(db)
			playlistDTO := models.Playlist{
				ID:          "spotify123",
				Name:        "Test Playlist",
				Description: "Test Description",
				TrackCount:  10,
				Public:      true,
			}
			playlist := models.NewPersistedPlaylist(0, "spotify", "spotify123", user.ID(), playlistDTO)
			playlist.SetID("nonexistent-id")

			err := playlistRepo.Update(playlist)
			if err == nil {
				t.Fatal("expected error when updating nonexistent playlist")
			}
		})

		t.Run("Delete", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			playlistRepo := NewPlaylistRepository(db)

			err := playlistRepo.Delete("nonexistent-id")
			if err == nil {
				t.Fatal("expected error when deleting nonexistent playlist")
			}
		})
	})
}

func TestMigrationRepositoryErrors(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		t.Run("InvalidUserID", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			migrationRepo := NewMigrationRepository(db)

			migration := models.NewMigrationJob(0, "nonexistent-user", "spotify", "playlist123", "youtube")
			err := migrationRepo.Create(migration)
			if err == nil {
				t.Fatal("expected error when creating migration with invalid user_id")
			}
		})
	})

	t.Run("NotFound errors", func(t *testing.T) {
		t.Run("Get", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			migrationRepo := NewMigrationRepository(db)

			_, err := migrationRepo.Get("nonexistent-id")
			if err == nil {
				t.Fatal("expected error when getting nonexistent migration")
			}
		})

		t.Run("Update", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			userRepo := NewUserRepository(db)
			user := models.NewUser(0, "test@example.com", "Test User")
			if err := userRepo.Create(user); err != nil {
				t.Fatalf("failed to create user: %v", err)
			}

			migrationRepo := NewMigrationRepository(db)
			migration := models.NewMigrationJob(0, user.ID(), "spotify", "playlist123", "youtube")
			migration.SetID("nonexistent-id")

			err := migrationRepo.Update(migration)
			if err == nil {
				t.Fatal("expected error when updating nonexistent migration")
			}
		})

		t.Run("Delete", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			migrationRepo := NewMigrationRepository(db)

			err := migrationRepo.Delete("nonexistent-id")
			if err == nil {
				t.Fatal("expected error when deleting nonexistent migration")
			}
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("FilterByStatus", func(t *testing.T) {
			db := setupTestDB(t)
			defer db.Close()

			userRepo := NewUserRepository(db)
			user := models.NewUser(0, "test@example.com", "Test User")
			if err := userRepo.Create(user); err != nil {
				t.Fatalf("failed to create user: %v", err)
			}

			playlistRepo := NewPlaylistRepository(db)
			playlists := make([]*models.PersistedPlaylist, 3)
			for i := 0; i < 3; i++ {
				pl := models.NewPersistedPlaylist(0, "spotify", fmt.Sprintf("spotifyid%d", i+1), user.ID(), models.Playlist{
					ID:          fmt.Sprintf("spotifyid%d", i+1),
					Name:        fmt.Sprintf("Playlist %d", i+1),
					Description: "Test",
					TrackCount:  10,
					Public:      false,
				})
				if err := playlistRepo.Create(pl); err != nil {
					t.Fatalf("failed to create playlist%d: %v", i+1, err)
				}
				playlists[i] = pl
			}

			migrationRepo := NewMigrationRepository(db)

			migration1 := models.NewMigrationJob(0, user.ID(), "spotify", playlists[0].ID(), "youtube")
			migration1.SetStatus("pending")
			if err := migrationRepo.Create(migration1); err != nil {
				t.Fatalf("failed to create migration1: %v", err)
			}

			migration2 := models.NewMigrationJob(0, user.ID(), "spotify", playlists[1].ID(), "youtube")
			migration2.SetStatus("completed")
			if err := migrationRepo.Create(migration2); err != nil {
				t.Fatalf("failed to create migration2: %v", err)
			}

			migration3 := models.NewMigrationJob(0, user.ID(), "spotify", playlists[2].ID(), "youtube")
			migration3.SetStatus("completed")
			if err := migrationRepo.Create(migration3); err != nil {
				t.Fatalf("failed to create migration3: %v", err)
			}

			completed, err := migrationRepo.List(map[string]any{"status": "completed"})
			if err != nil {
				t.Fatalf("failed to list completed migrations: %v", err)
			}

			if len(completed) != 2 {
				t.Errorf("expected 2 completed migrations, got %d", len(completed))
			}

			pending, err := migrationRepo.List(map[string]any{"status": "pending"})
			if err != nil {
				t.Fatalf("failed to list pending migrations: %v", err)
			}

			if len(pending) != 1 {
				t.Errorf("expected 1 pending migration, got %d", len(pending))
			}
		})
	})
}

func TestTrackCacheAdapter_CacheTrack_InvalidTrack(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTrackRepository(db)
	adapter := NewTrackCacheAdapter(repo)

	trackDTO := models.Track{
		ID:     "spotify123",
		Title:  "",
		Artist: "",
	}

	if err := adapter.CacheTrack("spotify", "spotify123", trackDTO); err == nil {
		t.Fatal("expected error when caching invalid track")
	}
}
