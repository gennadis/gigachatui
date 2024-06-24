package chat

type ChatModel string

const (
	ChatModelLite ChatModel = "GigaChat"
	ChatModelPro  ChatModel = "GigaChat-Pro"
)

type ChatRole string

const (
	ChatRoleUser      ChatRole = "user"
	ChatRoleAssistant ChatRole = "assistant"
	ChatRoleSystem    ChatRole = "system"
)

type ChatRequest struct {
	Model    ChatModel     `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Options  ChatOptions
}

func NewDefaultChatRequest(messages []ChatMessage) *ChatRequest {
	return &ChatRequest{
		Model:    ChatModelLite,
		Messages: messages,
		Options: ChatOptions{
			Temperature:       0.87,
			N:                 1,
			Stream:            false,
			MaxTokens:         512,
			RepetitionPenalty: 1.07,
			UpdateInterval:    0,
		},
	}
}

type ChatMessage struct {
	Role    ChatRole `json:"role"`
	Content string   `json:"content"`
}

type ChatOptions struct {
	Temperature       float64 `json:"temperature"`        // [ 0 .. 2 ] Default: 0.87
	TopP              float64 `json:"top_p"`              // [ 0 .. 1 ] Default: 0.47
	N                 int64   `json:"n"`                  // [ 1 .. 4 ] Default: 1
	Stream            bool    `json:"stream"`             // Default: false
	MaxTokens         int64   `json:"max_tokens"`         // Default: 512
	RepetitionPenalty float64 `json:"repetition_penalty"` // Default: 1.07
	UpdateInterval    float64 `json:"update_interval"`    // Default: 0
}

type ChatResponse struct {
	Choices []ChatResponseChoice `json:"choices"`
	Created int64                `json:"created"`
	Model   ChatModel            `json:"model"`
	Usage   ChatResponseUsage    `json:"usage"`
	Object  string               `json:"object"`
}

type ChatResponseChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type ChatResponseUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}
