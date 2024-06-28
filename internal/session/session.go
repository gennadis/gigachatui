package session

import (
	"time"

	"github.com/gennadis/gigachatui/internal/chat"
	"github.com/google/uuid"
)

// Session represents a chat session
type Session struct {
	ID        string
	Name      string
	CreatedAt int64
	Messages  []chat.Message
}

// NewSession creates a new Session instance
func NewSession(name string) *Session {
	return &Session{
		ID:        uuid.NewString(),
		Name:      name,
		CreatedAt: time.Now().Unix(),
		Messages:  []chat.Message{},
	}
}
