package chat

import (
	"time"

	"github.com/google/uuid"
)

// Default values for chat options. More info can be found here:
// https://developers.sber.ru/docs/ru/gigachat/api/reference/rest/post-chat#zapros
const (
	defaultTemperature       = 0.87 // [0 .. 2] Default: 0.87
	defaultTopP              = 0.47 // [0 .. 1] Default: 0.47
	defaultN                 = 1    // [1 .. 4] Default: 1
	defaultStream            = true // Default: false
	defaultMaxTokens         = 1024 // Default: 1024
	defaultRepetitionPenalty = 1.07 // Default: 1.07
	defaultUpdateInterval    = 0.1  // in seconds
)

// Message represents a message in the chat
type Message struct {
	ID        string    `db:"id" json:"-"`
	Content   string    `db:"content"`
	Role      Role      `db:"role"`
	SessionID string    `db:"session_id" json:"-"`
	Timestamp time.Time `db:"timestamp" json:"-"`
}

// NewMessage creates a new Message
func NewMessage(content string, role Role, sessionID string) *Message {
	return &Message{
		ID:        uuid.NewString(),
		Content:   content,
		Role:      role,
		SessionID: sessionID,
		Timestamp: time.Now(),
	}
}

// Model represents the model type for the chat
type Model string

const (
	// ChatModelLite represents GigaChat Lite Model
	ChatModelLite Model = "GigaChat"
	// ChatModelPro represents GigaChat Pro Model
	ChatModelPro Model = "GigaChat-Pro"
)

// Role represents the role of a message in the chat
type Role string

const (
	// RoleUser represents user propmt
	RoleUser Role = "user"
	// RoleAssistant represents assistant response
	RoleAssistant Role = "assistant"
	// RoleSystem represents system prompt
	RoleSystem Role = "system"
)

// Request represents a request to the chat API
type Request struct {
	Model    Model     `json:"model"`
	Messages []Message `json:"messages"`
	Options
}

// NewRequest creates a new Request with default options
func NewRequest(messages []Message) *Request {
	return &Request{
		Model:    ChatModelLite,
		Messages: messages,
		Options: Options{
			Temperature:       defaultTemperature,
			N:                 defaultN,
			Stream:            defaultStream,
			MaxTokens:         defaultMaxTokens,
			RepetitionPenalty: defaultRepetitionPenalty,
			UpdateInterval:    defaultUpdateInterval,
		},
	}
}

// Options represents the options for a chat request
type Options struct {
	Temperature       float64 `json:"temperature"`
	TopP              float64 `json:"top_p"`
	N                 int64   `json:"n"`
	Stream            bool    `json:"stream"`
	MaxTokens         int64   `json:"max_tokens"`
	RepetitionPenalty float64 `json:"repetition_penalty"`
	UpdateInterval    float64 `json:"update_interval"`
}

// Usage represents the usage details of a chat response
type Usage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}

// Choice represents a choice in the chat response
type Choice struct {
	Delta Message `json:"delta"`
	Index int32   `json:"index"`
}

// StreamChunk represents a chunk of the chat response stream
type StreamChunk struct {
	Choices []Choice `json:"choices"`
	Created int64    `json:"created"`
	Model   Model    `json:"model"`
	Object  string   `json:"object"`
	Usage   Usage    `json:"usage"`
	Final   bool     `json:"-"`
}
