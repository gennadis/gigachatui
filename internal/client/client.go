package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gennadis/gigachatui/internal/auth"
	"github.com/gennadis/gigachatui/internal/chat"
	"github.com/gennadis/gigachatui/internal/config"
	"github.com/gennadis/gigachatui/storage"
)

const (
	streamRespChunkStartLine = "data: "
	streamRespFinalLine      = "data: [DONE]"
	contentTypeJSON          = "application/json"
)

// Client represents a client for interacting with the GigaChat API
type Client struct {
	Config         *config.Config
	AuthManager    *auth.Manager
	SessionStorage *storage.Sessions
	MessageStorage *storage.Messages
	StreamRespChan chan chat.ResponseStreamChunk
	ErrorChan      chan error
	httpClient     *http.Client
}

// NewClient initializes a new Client instance
func NewClient(ctx context.Context, cfg config.Config, sessionStorage storage.Sessions, messagesStorage storage.Messages, clientID, clientSecret string) (*Client, error) {
	m, err := auth.NewManager(ctx, clientID, clientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to init authentication manager: %w", err)
	}

	return &Client{
		Config:         &cfg,
		AuthManager:    m,
		SessionStorage: &sessionStorage,
		MessageStorage: &messagesStorage,
		StreamRespChan: make(chan chat.ResponseStreamChunk),
		ErrorChan:      make(chan error),
		httpClient:     &http.Client{Timeout: time.Second * 10},
	}, nil
}

// GetCompletion sends a question to the chat API and processes the response
func (c *Client) GetCompletion(ctx context.Context, sessionID, question string) error {
	// Create a new message for the user's question and write it to the storage
	userMsg := chat.NewMessage(question, chat.RoleUser, sessionID)
	if err := c.MessageStorage.Write(*userMsg); err != nil {
		return fmt.Errorf("failed to write user message to storage: %w", err)
	}

	// Read all messages for the given session from storage
	// This is necessary to provide context to the chat assistant
	sessionMessages, err := c.MessageStorage.ReadBySessionID(sessionID)
	if err != nil {
		return fmt.Errorf("failed to read session messages from storage: %w", err)
	}

	// Create a request with the session messages to send to the GigaChat API
	request := chat.NewDefaultRequest(sessionMessages)
	resp, err := c.callCompletionsAPI(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to get chat completion: %w", err)
	}

	// Process the response from the chat API asynchronously
	go c.processCompletionsAPIResponse(resp)

	// Buffer to build the assistant's response text incrementally
	var assistantRespTxt strings.Builder

	for {
		select {
		// Handle the streaming response from the completions API
		case chunk := <-c.StreamRespChan:
			// If the chunk is marked as final, write the complete response to storage
			if chunk.Final {
				assistantRespMsg := chat.NewMessage(assistantRespTxt.String(), chat.RoleAssistant, sessionID)
				if err := c.MessageStorage.Write(*assistantRespMsg); err != nil {
					return fmt.Errorf("failed to write assistant response message to storage: %w", err)
				}
				return nil
			}

			// Ensure there are choices available in the chunk
			if len(chunk.Choices) == 0 {
				return fmt.Errorf("no choices found in completions API response")
			}

			// Append the chunk content to the response text
			chunkContent := chunk.Choices[0].Delta.Content
			assistantRespTxt.WriteString(chunkContent)
			fmt.Print(chunkContent)

		// Handle errors that may occur during the streaming response processing
		case err := <-c.ErrorChan:
			return fmt.Errorf("failed to process completions response stream: %w", err)
		}
	}
}

// callCompletionsAPI sends a request to the chat completions API
func (c *Client) callCompletionsAPI(ctx context.Context, request *chat.Request) (*http.Response, error) {
	reqBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chat request: %w", err)
	}

	completionsPath := c.Config.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", completionsPath, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build completion API request: %w", err)
	}

	authHeader := fmt.Sprintf("Bearer %s", c.AuthManager.Token.AccessToken)
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", contentTypeJSON)
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send completion request: %w", err)
	}

	return resp, nil
}

// processCompletionsAPIResponse processes the response from the chat completions API
func (c *Client) processCompletionsAPIResponse(r *http.Response) {
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			c.ErrorChan <- fmt.Errorf("failed to read completion stream response body: %w", err)
			return
		}
		c.ErrorChan <- fmt.Errorf("failed to process API response with body: %s: %w", string(bodyBytes), err)
		return
	}

	sc := bufio.NewScanner(r.Body)
	for sc.Scan() {
		ln := sc.Text()
		if ln == streamRespFinalLine {
			finalRespChunk := chat.ResponseStreamChunk{Final: true}
			c.StreamRespChan <- finalRespChunk
			return
		}

		if strings.HasPrefix(ln, streamRespChunkStartLine) || ln == "\n" {
			jsonStr := strings.TrimPrefix(ln, streamRespChunkStartLine)
			var respChunk chat.ResponseStreamChunk
			if err := json.Unmarshal([]byte(jsonStr), &respChunk); err != nil {
				c.ErrorChan <- fmt.Errorf("failed to unmarshal completion response stream chunk: %w", err)
				return
			}
			respChunk.Final = false
			c.StreamRespChan <- respChunk
		}
	}

	if err := sc.Err(); err != nil {
		c.ErrorChan <- fmt.Errorf("failed to scan completion response stream chunk: %w", err)
	}
}
