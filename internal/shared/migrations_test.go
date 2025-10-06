package shared

import (
	"testing"
)

func TestMigrationRunner(t *testing.T) {
	t.Run("loadMigrations", func(t *testing.T) {
		migrations, err := loadMigrations()
		if err != nil {
			t.Fatalf("failed to load migrations: %v", err)
		}

		if len(migrations) == 0 {
			t.Fatal("expected at least one migration")
		}

		for i := 1; i < len(migrations); i++ {
			if migrations[i].Version <= migrations[i-1].Version {
				t.Errorf("migrations not sorted: version %d comes after %d", migrations[i].Version, migrations[i-1].Version)
			}
		}

		for _, m := range migrations {
			if m.Up == "" {
				t.Errorf("migration version %d missing up SQL", m.Version)
			}
			if m.Down == "" {
				t.Errorf("migration version %d missing down SQL", m.Version)
			}
		}
	})

	t.Run("RunMigrations And Rollback", func(t *testing.T) {
		db, err := NewDatabase(":memory:")
		if err != nil {
			t.Fatalf("failed to create database: %v", err)
		}
		defer db.Close()

		if err := RunMigrations(db); err != nil {
			t.Fatalf("failed to run migrations: %v", err)
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query schema_migrations: %v", err)
		}
		if count == 0 {
			t.Error("expected at least one migration to be applied")
		}

		_, err = db.Exec("SELECT 1 FROM users LIMIT 1")
		if err != nil {
			t.Errorf("users table should exist after migrations: %v", err)
		}

		if err := RollbackMigration(db); err != nil {
			t.Fatalf("failed to rollback migration: %v", err)
		}

		var newCount int
		err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&newCount)
		if err != nil {
			t.Fatalf("failed to query schema_migrations after rollback: %v", err)
		}
		if newCount >= count {
			t.Errorf("expected migration count to decrease after rollback, got %d (was %d)", newCount, count)
		}
	})

	t.Run("Idempotent Migrations", func(t *testing.T) {
		db, err := NewDatabase(":memory:")
		if err != nil {
			t.Fatalf("failed to create database: %v", err)
		}
		defer db.Close()

		if err := RunMigrations(db); err != nil {
			t.Fatalf("failed to run migrations first time: %v", err)
		}

		if err := RunMigrations(db); err != nil {
			t.Fatalf("failed to run migrations second time: %v", err)
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query schema_migrations: %v", err)
		}

		migrations, _ := loadMigrations()
		if count != len(migrations) {
			t.Errorf("expected %d migrations to be applied, got %d", len(migrations), count)
		}
	})
}
