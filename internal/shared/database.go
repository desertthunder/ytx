package shared

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// NewDatabase opens a connection to a SQLite database at the specified path.
// The path can be ":memory:" for an in-memory database.
// Returns an open database connection or an error if connection fails.
func NewDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// ConfigureDatabase sets connection pool settings for the database.
// Recommended for production use to limit connections and improve performance.
func ConfigureDatabase(db *sql.DB, maxOpenConns, maxIdleConns int) {
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
}
