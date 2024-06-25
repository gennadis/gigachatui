package chat

// https://developers.sber.ru/docs/ru/gigachat/api/reference/rest/post-chat#zapros
const (
	defaultTemperature       = 0.87 // [ 0 .. 2 ] Default: 0.87
	defaultTopP              = 0.47 // [ 0 .. 1 ] Default: 0.47
	defaultN                 = 1    // [ 1 .. 4 ] Default: 1
	defaultStream            = true // Default: false
	defaultMaxTokens         = 1024 // Default: 1024
	defaultRepetitionPenalty = 1.07 // Default: N/A
	defaultUpdateInterval    = 0.1  // in seconds
)

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
	ChatOptions
}

func NewDefaultChatRequest(messages []ChatMessage) *ChatRequest {
	return &ChatRequest{
		Model:    ChatModelLite,
		Messages: messages,
		ChatOptions: ChatOptions{
			Temperature:       defaultTemperature,
			N:                 defaultN,
			Stream:            defaultStream,
			MaxTokens:         defaultMaxTokens,
			RepetitionPenalty: defaultRepetitionPenalty,
			UpdateInterval:    defaultUpdateInterval,
		},
	}
}

type ChatMessage struct {
	Content string   `json:"content"`
	Role    ChatRole `json:"role"`
}

type ChatOptions struct {
	Temperature       float64 `json:"temperature"`
	TopP              float64 `json:"top_p"`
	N                 int64   `json:"n"`
	Stream            bool    `json:"stream"`
	MaxTokens         int64   `json:"max_tokens"`
	RepetitionPenalty float64 `json:"repetition_penalty"`
	UpdateInterval    float64 `json:"update_interval"`
}

type ChatResponseUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}

type ChatResponseChoice struct {
	Delta ChatMessage `json:"delta"`
	Index int32       `json:"index"`
}

type ChatResponseStreamChunk struct {
	Choices []ChatResponseChoice `json:"choices"`
	Created int64                `json:"created"`
	Model   ChatModel            `json:"model"`
	Object  string               `json:"object"`
	Usage   ChatResponseUsage    `json:"usage"`
	Final   bool
}
