package chat

import (
	"time"

	"github.com/google/uuid"
)

// Session represents a chat session
type Session struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Timestamp time.Time `db:"timestamp"`
}

// NewSession creates a new Session instance
func NewSession(name string) *Session {
	return &Session{
		ID:        uuid.NewString(),
		Name:      name,
		Timestamp: time.Now(),
	}
}
