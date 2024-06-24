package session

import (
	"time"

	"github.com/gennadis/gigachatui/internal/chat"
	"github.com/google/uuid"
)

type Session struct {
	ID        string
	Name      string
	CreatedAt int64
	Messages  []chat.ChatMessage
}

func NewSession(name string) *Session {
	return &Session{
		ID:        uuid.NewString(),
		Name:      name,
		CreatedAt: time.Now().Unix(),
	}
}
