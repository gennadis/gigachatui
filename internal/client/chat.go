package client

type ChatModel string

const (
	Lite ChatModel = "GigaChat"
	Pro  ChatModel = "GigaChat-Pro"
)

type ChatModelRole string

const (
	System        ChatModelRole = "system"
	ChatModelUser ChatModelRole = "user"
	Assistant     ChatModelRole = "assistant"
)

type ChatCompletionRequest struct {
	Model             ChatModel     `json:"model"`
	Messages          []ChatMessage `json:"messages"`
	Temperature       float64       `json:"temperature"`
	TopP              float64       `json:"top_p"`
	N                 int64         `json:"n"`
	MaxTokens         int32         `json:"max_tokens"`
	RepetitionPenalty float64       `json:"repetition_penalty"`
}

type ChatMessage struct {
	Role    ChatModelRole `json:"role"`
	Content string        `json:"content"`
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
