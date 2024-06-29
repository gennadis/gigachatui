package storage

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gennadis/gigachatui/internal/chat"
	"github.com/jmoiron/sqlx"
)

// Messages is a storage for messages
type Messages struct {
	db *sqlx.DB
}

// NewMessages creates a new Messages storage
func NewMessages(db *sqlx.DB) (*Messages, error) {
	createMessagesTable := `
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		content TEXT NOT NULL,
		role TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	)
	`
	if _, err := db.Exec(createMessagesTable); err != nil {
		return nil, fmt.Errorf("failed to create messages table: %w", err)
	}

	return &Messages{db: db}, nil
}

// Read returns all messages
func (m *Messages) Read() ([]chat.Message, error) {
	var messages []chat.Message
	err := m.db.Select(&messages, "SELECT id, session_id, content, role, timestamp FROM messages ORDER BY timestamp ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	slog.Debug("read messages",
		slog.Int("count", len(messages)),
	)
	return messages, nil
}

// ReadBySessionID returns messages for a specific session_id
func (m *Messages) ReadBySessionID(sessionID string) ([]chat.Message, error) {
	var messages []chat.Message
	err := m.db.Select(&messages, "SELECT id, session_id, content, role, timestamp FROM messages WHERE session_id = ? ORDER BY timestamp ASC", sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages for session_id %s: %w", sessionID, err)
	}

	slog.Debug("read messages by session_id",
		slog.String("session_id", sessionID),
		slog.Int("count", len(messages)),
	)
	return messages, nil
}

// Write writes new message to the storage
func (m *Messages) Write(message chat.Message) error {
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}
	// Prepare the query to insert a new record, ignoring if it already exists
	insertQuery := "INSERT OR IGNORE INTO messages (id, session_id, content, role, timestamp) VALUES (?, ?, ?, ?, ?)"
	if _, err := m.db.Exec(insertQuery, message.ID, message.SessionID, message.Content, message.Role, message.Timestamp); err != nil {
		return fmt.Errorf("failed to insert message %+v: %w", message, err)
	}

	slog.Debug("message added to messages",
		slog.String("id", message.ID),
		slog.String("session_id", message.SessionID),
		slog.String("content", message.Content),
		slog.String("role", string(message.Role)),
		slog.Time("timestamp", message.Timestamp),
	)
	return nil
}

// Delete deletes the given message by id from the storage
func (m *Messages) Delete(id string) error {
	var message chat.Message

	// retrieve the message's session_id and timestamp for logging purposes
	err := m.db.Get(&message, "SELECT id, session_id, timestamp FROM messages WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to get message for id %s: %w", id, err)
	}

	if _, err := m.db.Exec("DELETE FROM messages WHERE id = ?", id); err != nil {
		return fmt.Errorf("failed to delete message by id %s: %w", id, err)
	}

	slog.Debug("message deleted from messages",
		slog.String("id", message.ID),
		slog.String("session_id", message.SessionID),
		slog.Time("timestamp", message.Timestamp),
	)
	return nil
}
