package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// InitDB initializes the SQLite database and creates all tables.
func InitDB(dbPath string) (*sql.DB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := createTables(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating tables: %w", err)
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS actors (
			id TEXT PRIMARY KEY,
			actor_type TEXT NOT NULL,
			name TEXT NOT NULL,
			grade TEXT,
			subject TEXT,
			family_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			actor_id TEXT NOT NULL,
			actor_type TEXT NOT NULL,
			messages TEXT NOT NULL DEFAULT '[]',
			summary TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS learning_events (
			id TEXT PRIMARY KEY,
			student_id TEXT NOT NULL,
			subject TEXT NOT NULL,
			kp_id TEXT NOT NULL,
			kp_name TEXT NOT NULL,
			is_correct INTEGER NOT NULL DEFAULT 0,
			note TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS knowledge_states (
			id TEXT PRIMARY KEY,
			student_id TEXT NOT NULL,
			subject TEXT NOT NULL,
			kp_id TEXT NOT NULL,
			kp_name TEXT NOT NULL,
			correct_count INTEGER NOT NULL DEFAULT 0,
			total_count INTEGER NOT NULL DEFAULT 0,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(student_id, subject, kp_id)
		)`,
		`CREATE TABLE IF NOT EXISTS rendered_contents (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			actor_id TEXT NOT NULL,
			content_type TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("executing query %q: %w", q[:50], err)
		}
	}

	if err := ensureSessionSchema(db); err != nil {
		return err
	}

	return nil
}

func ensureSessionSchema(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(sessions)`)
	if err != nil {
		return fmt.Errorf("querying sessions schema: %w", err)
	}
	defer rows.Close()

	hasSummary := false
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scanning sessions schema: %w", err)
		}
		if name == "summary" {
			hasSummary = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("reading sessions schema: %w", err)
	}

	if !hasSummary {
		if _, err := db.Exec(`ALTER TABLE sessions ADD COLUMN summary TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("adding sessions.summary column: %w", err)
		}
	}
	return nil
}
