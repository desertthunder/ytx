package shared

import (
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

//go:embed sql/*.sql
var migrationFiles embed.FS

// Migration represents a database migration with up and down SQL.
type Migration struct {
	Version int
	Up      string
	Down    string
}

// loadMigrations reads all migration files from the embedded filesystem and returns them sorted by version.
func loadMigrations() ([]Migration, error) {
	entries, err := migrationFiles.ReadDir("sql")
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	migrationMap := make(map[int]*Migration)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// Parse version from filename (e.g., "0000_create_tables_up.sql" -> version 0)
		parts := strings.Split(name, "_")
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		content, err := migrationFiles.ReadFile(filepath.Join("sql", name))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", name, err)
		}

		if migrationMap[version] == nil {
			migrationMap[version] = &Migration{Version: version}
		}

		if strings.Contains(name, "_up.sql") {
			migrationMap[version].Up = string(content)
		} else if strings.Contains(name, "_down.sql") {
			migrationMap[version].Down = string(content)
		}
	}

	// Convert map to sorted slice
	var migrations []Migration
	for _, migration := range migrationMap {
		if migration.Up == "" || migration.Down == "" {
			return nil, fmt.Errorf("incomplete migration for version %d", migration.Version)
		}
		migrations = append(migrations, *migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// RunMigrations executes all pending migrations on the database.
// Creates a schema_migrations table to track applied migrations.
func RunMigrations(db *sql.DB) error {
	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	for _, migration := range migrations {
		// Check if this migration has already been applied
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)", migration.Version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if !exists {
			if err := applyMigration(db, migration); err != nil {
				return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
			}
		}
	}

	return nil
}

// RollbackMigration rolls back the most recent migration.
func RollbackMigration(db *sql.DB) error {
	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Check if there are any applied migrations
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check migrations: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	// Get the highest version number that's been applied
	currentVersion, err := getCurrentVersion(db)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	for _, migration := range migrations {
		if migration.Version == currentVersion {
			if err := rollbackMigration(db, migration); err != nil {
				return fmt.Errorf("failed to rollback migration %d: %w", migration.Version, err)
			}
			return nil
		}
	}

	return fmt.Errorf("migration version %d not found", currentVersion)
}

// createMigrationsTable creates the schema_migrations table if it doesn't exist.
func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := db.Exec(query)
	return err
}

// getCurrentVersion returns the current migration version.
func getCurrentVersion(db *sql.DB) (int, error) {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// applyMigration executes a migration's up SQL and records it.
func applyMigration(db *sql.DB, migration Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute each statement separately
	statements := strings.Split(migration.Up, ";")
	for _, stmt := range statements {
		// Remove comments and trim whitespace
		stmt = removeComments(stmt)
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %w\nStatement: %s", err, stmt)
		}
	}

	// Record the migration
	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migration.Version); err != nil {
		return err
	}

	return tx.Commit()
}

// removeComments removes SQL comments from a statement.
func removeComments(sql string) string {
	lines := strings.Split(sql, "\n")
	var result []string
	for _, line := range lines {
		// Remove single-line comments
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// rollbackMigration executes a migration's down SQL and removes the record.
func rollbackMigration(db *sql.DB, migration Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute each statement separately
	statements := strings.Split(migration.Down, ";")
	for _, stmt := range statements {
		// Remove comments and trim whitespace
		stmt = removeComments(stmt)
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %w\nStatement: %s", err, stmt)
		}
	}

	// Remove the migration record
	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = ?", migration.Version); err != nil {
		return err
	}

	return tx.Commit()
}