package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"pathpad/internal/models"
)

const currentSchemaVersion = 1

// SQLiteStore provides persistent storage using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) the SQLite database and runs migrations.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Ensure the parent directory exists.
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create database directory %q: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Verify the connection works.
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return store, nil
}

// migrate runs schema migrations.
func (s *SQLiteStore) migrate() error {
	// Create schema_version table if not exists.
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`)
	if err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	// Get current version.
	var version int
	err = s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&version)
	if err != nil {
		return fmt.Errorf("get schema version: %w", err)
	}

	if version < 1 {
		log.Println("[db] Running migration v1: create pads table")
		_, err = s.db.Exec(`
			CREATE TABLE IF NOT EXISTS pads (
				path TEXT PRIMARY KEY,
				content TEXT NOT NULL DEFAULT '',
				parent_path TEXT NOT NULL DEFAULT '',
				updated_at INTEGER NOT NULL,
				created_at INTEGER NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_parent_path ON pads(parent_path);
			CREATE INDEX IF NOT EXISTS idx_updated_at ON pads(updated_at);
			INSERT OR REPLACE INTO schema_version (version) VALUES (1);
		`)
		if err != nil {
			return fmt.Errorf("migration v1: %w", err)
		}
	}

	log.Printf("[db] Schema at version %d\n", currentSchemaVersion)
	return nil
}

// Ping checks database connectivity.
func (s *SQLiteStore) Ping() error {
	return s.db.Ping()
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// GetPad retrieves a pad by path. Returns an empty pad (with zero timestamps)
// if the pad doesn't exist in the database (implicit pad).
func (s *SQLiteStore) GetPad(path string) (*models.Pad, error) {
	pad := &models.Pad{Path: path}
	err := s.db.QueryRow(
		`SELECT content, parent_path, updated_at, created_at FROM pads WHERE path = ?`,
		path,
	).Scan(&pad.Content, &pad.ParentPath, &pad.UpdatedAt, &pad.CreatedAt)

	if err == sql.ErrNoRows {
		// Implicit pad: exists conceptually but not in DB.
		pad.Content = ""
		pad.ParentPath = models.ParentPath(path)
		pad.UpdatedAt = 0
		pad.CreatedAt = 0
		return pad, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get pad %q: %w", path, err)
	}
	return pad, nil
}

// SavePad upserts a pad's content. Creates the row if it doesn't exist,
// updates it if it does. Returns the saved pad.
func (s *SQLiteStore) SavePad(path, content string) (*models.Pad, error) {
	now := time.Now().Unix()
	parentPath := models.ParentPath(path)

	_, err := s.db.Exec(`
		INSERT INTO pads (path, content, parent_path, updated_at, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			content = excluded.content,
			updated_at = excluded.updated_at
	`, path, content, parentPath, now, now)
	if err != nil {
		return nil, fmt.Errorf("save pad %q: %w", path, err)
	}

	// Retrieve the saved pad (to get the correct created_at for existing pads).
	return s.GetPad(path)
}

// DeletePad deletes a pad and all its descendants. Returns the count of deleted rows.
func (s *SQLiteStore) DeletePad(path string) (int64, error) {
	var result sql.Result
	var err error

	if path == "" {
		// Root: delete everything.
		result, err = s.db.Exec(`DELETE FROM pads`)
	} else {
		// Delete the pad itself and all descendants.
		// Descendants have path starting with "path/" or parent_path starting with "path".
		result, err = s.db.Exec(
			`DELETE FROM pads WHERE path = ? OR path LIKE ? || '/%'`,
			path, path,
		)
	}
	if err != nil {
		return 0, fmt.Errorf("delete pad %q: %w", path, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return count, nil
}

// GetChildren returns all direct children of a given path that have content.
// Children are sorted alphabetically by path.
func (s *SQLiteStore) GetChildren(parentPath string) ([]models.ChildPad, error) {
	rows, err := s.db.Query(
		`SELECT path, updated_at FROM pads WHERE parent_path = ? AND path != ? ORDER BY path ASC`,
		parentPath, parentPath,
	)
	if err != nil {
		return nil, fmt.Errorf("get children of %q: %w", parentPath, err)
	}
	defer rows.Close()

	var children []models.ChildPad
	for rows.Next() {
		var child models.ChildPad
		if err := rows.Scan(&child.Path, &child.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan child: %w", err)
		}
		children = append(children, child)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate children: %w", err)
	}

	// Return empty slice instead of nil for consistent JSON serialization.
	if children == nil {
		children = []models.ChildPad{}
	}
	return children, nil
}

// PathExists checks if a pad with content exists in the database.
func (s *SQLiteStore) PathExists(path string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pads WHERE path = ?`, path).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("path exists %q: %w", path, err)
	}
	return count > 0, nil
}
