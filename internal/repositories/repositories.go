// package repositories provides persistence layer implementations for all model types.
//
// Each repository implements models.Repository[T] for a specific entity type,
// handling CRUD operations, soft deletes, and sequence generation.
package repositories

import (
	"database/sql"
	"fmt"
)

// NextSequence atomically increments and returns the next sequence number for the given table.
//
// Sequence numbers provide human-readable ordering for entities (e.g., user #42, playlist #15).
// They are NOT exposed in CLI output but used internally for sorting and debugging.
func NextSequence(db *sql.DB, table string) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	sequenceTable := table + "_sequence"

	_, err = tx.Exec(fmt.Sprintf("UPDATE %s SET value = value + 1 WHERE id = 1", sequenceTable))
	if err != nil {
		return 0, fmt.Errorf("failed to increment sequence: %w", err)
	}

	var sequence int
	err = tx.QueryRow(fmt.Sprintf("SELECT value FROM %s WHERE id = 1", sequenceTable)).Scan(&sequence)
	if err != nil {
		return 0, fmt.Errorf("failed to get sequence value: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit sequence transaction: %w", err)
	}

	return sequence, nil
}
