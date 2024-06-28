package chat

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

// NewDefaultRequest creates a new Request with default options
func NewDefaultRequest(messages []Message) *Request {
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

// Message represents a message in the chat
type Message struct {
	Content string `json:"content"`
	Role    Role   `json:"role"`
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

// ResponseUsage represents the usage details of a chat response
type ResponseUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}

// ResponseChoice represents a choice in the chat response
type ResponseChoice struct {
	Delta Message `json:"delta"`
	Index int32   `json:"index"`
}

// ResponseStreamChunk represents a chunk of the chat response stream
type ResponseStreamChunk struct {
	Choices []ResponseChoice `json:"choices"`
	Created int64            `json:"created"`
	Model   Model            `json:"model"`
	Object  string           `json:"object"`
	Usage   ResponseUsage    `json:"usage"`
	Final   bool             `json:"-"`
}
