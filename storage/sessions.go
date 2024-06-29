package storage

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gennadis/gigachatui/internal/chat"
	"github.com/jmoiron/sqlx"
)

// Sessions is a storage for sessions
type Sessions struct {
	db *sqlx.DB
}

// NewSessions creates a new Sessions storage
func NewSessions(db *sqlx.DB) (*Sessions, error) {
	createSessionsTable := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	)
	`
	if _, err := db.Exec(createSessionsTable); err != nil {
		return nil, fmt.Errorf("failed to create sessions table: %w", err)
	}

	return &Sessions{db: db}, nil
}

// Read returns all sessions
func (s *Sessions) Read() ([]chat.Session, error) {
	var sessions []chat.Session
	err := s.db.Select(&sessions, "SELECT id, name, timestamp FROM sessions ORDER BY timestamp DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	slog.Debug("read sessions",
		slog.Int("count", len(sessions)),
	)
	return sessions, nil
}

// Write writes new session to the storage
func (s *Sessions) Write(session chat.Session) error {
	if session.Timestamp.IsZero() {
		session.Timestamp = time.Now()
	}
	// Prepare the query to insert a new record, ignoring if it already exists
	insertQuery := "INSERT OR IGNORE INTO sessions (id, name, timestamp) VALUES (?, ?, ?)"
	if _, err := s.db.Exec(insertQuery, session.ID, session.Name, session.Timestamp); err != nil {
		return fmt.Errorf("failed to insert session %+v: %w", session, err)
	}

	slog.Debug("session added to sessions",
		slog.String("id", session.ID),
		slog.String("name", session.Name),
		slog.Time("timestamp", session.Timestamp),
	)
	return nil
}

// Delete deletes the given session by id from the storage
func (s *Sessions) Delete(id string) error {
	var session chat.Session

	// retrieve the session's name and timestamp for logging purposes
	err := s.db.Get(&session, "SELECT id, name, timestamp FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to get session for id %s: %w", id, err)
	}

	if _, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id); err != nil {
		return fmt.Errorf("failed to delete session by id %s: %w", id, err)
	}

	slog.Debug("session deleted from sessions",
		slog.String("id", session.ID),
		slog.String("name", session.Name),
	)
	return nil
}
