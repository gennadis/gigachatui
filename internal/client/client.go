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
	streamDataPrefix    = "data: "
	streamDataDone      = "data: [DONE]"
	contentTypeJSON     = "application/json"
	completionsEndpoint = "/chat/completions"
)

// Client represents a client for interacting with the GigaChat API
type Client struct {
	Config             *config.Config
	AuthManager        *auth.Manager
	SessionStorage     *storage.Sessions
	MessageStorage     *storage.Messages
	StreamResponseChan chan chat.StreamChunk
	ErrorChan          chan error
	httpClient         *http.Client
}

// NewClient initializes a new Client instance
func NewClient(cfg config.Config, authManager auth.Manager, sessionStorage storage.Sessions, messagesStorage storage.Messages) (*Client, error) {
	return &Client{
		Config:             &cfg,
		AuthManager:        &authManager,
		SessionStorage:     &sessionStorage,
		MessageStorage:     &messagesStorage,
		StreamResponseChan: make(chan chat.StreamChunk),
		ErrorChan:          make(chan error),
		httpClient:         &http.Client{Timeout: time.Second * 10},
	}, nil
}

// RequestCompletion sends a question to the chat API and processes the response
func (c *Client) RequestCompletion(ctx context.Context, sessionID, question string) error {
	// Store the user's message in the message storage
	if err := c.storeUserMessage(sessionID, question); err != nil {
		return fmt.Errorf("failed to write user message to storage: %w", err)
	}

	// Read all messages for the given session from storage
	// This is necessary to provide context to the chat assistant
	sessionMessages, err := c.MessageStorage.ReadBySessionID(sessionID)
	if err != nil {
		return fmt.Errorf("failed to read session messages from storage: %w", err)
	}

	// Create a request with the session messages to send to the GigaChat API
	request := chat.NewRequest(sessionMessages)
	resp, err := c.sendCompletionRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to get chat completion: %w", err)
	}

	// Process the response from the chat API asynchronously
	go c.processResponseStream(resp)

	// Collect the response from the stream and store it
	if err := c.collectResponse(sessionID); err != nil {
		return fmt.Errorf("failed to collect completions API response: %w", err)
	}

	return nil
}

// sendCompletionRequest sends a request to the chat completions API
func (c *Client) sendCompletionRequest(ctx context.Context, request *chat.Request) (*http.Response, error) {
	// Marshal the request into JSON
	reqBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chat request: %w", err)
	}

	// Build the request
	completionsPath := c.Config.BaseURL + completionsEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", completionsPath, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build completion API request: %w", err)
	}

	// Set necessary headers
	authHeader := fmt.Sprintf("Bearer %s", c.AuthManager.Token.AccessToken)
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", contentTypeJSON)
	req.Header.Set("Authorization", authHeader)

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send completion request: %w", err)
	}

	// Handle non-200 responses
	if err := handleNonOKStatus(resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// processResponseStream processes the response from the chat completions API
func (c *Client) processResponseStream(r *http.Response) {
	defer r.Body.Close()
	sc := bufio.NewScanner(r.Body)
	for sc.Scan() {
		ln := sc.Text()
		if ln == streamDataDone {
			// Send a final chunk to indicate the end of the stream
			c.StreamResponseChan <- chat.StreamChunk{Final: true}
			return
		}

		if strings.HasPrefix(ln, streamDataPrefix) || ln == "\n" {
			// Extract the JSON string from the line
			jsonStr := strings.TrimPrefix(ln, streamDataPrefix)
			var respChunk chat.StreamChunk
			// Unmarshal the JSON string into a StreamChunk
			if err := json.Unmarshal([]byte(jsonStr), &respChunk); err != nil {
				c.ErrorChan <- fmt.Errorf("failed to unmarshal completion response stream chunk: %w", err)
				return
			}
			// Send the chunk to the StreamRespChan
			c.StreamResponseChan <- respChunk
		}
	}
	// Handle any errors that occurred during scanning
	if err := sc.Err(); err != nil {
		c.ErrorChan <- fmt.Errorf("failed to scan completion response stream chunk: %w", err)
	}
}

func (c *Client) collectResponse(sessionID string) error {
	// Buffer to build the assistant's response text incrementally
	var assistantRespTxt strings.Builder

	for {
		select {
		// Handle the streaming response from the completions API
		case chunk := <-c.StreamResponseChan:
			// If the chunk is marked as final, write the complete response to storage
			if chunk.Final {
				if err := c.storeAssistantMessage(sessionID, assistantRespTxt.String()); err != nil {
					return fmt.Errorf("failed to write assistant message to storage: %w", err)
				}
				return nil
			}

			// Ensure there are choices available in the chunk
			if len(chunk.Choices) == 0 {
				return fmt.Errorf("no choices found in completions API response")
			}

			// Append the chunk content to the response text
			content := chunk.Choices[0].Delta.Content
			assistantRespTxt.WriteString(content)
			fmt.Print(content)

		// Handle errors that may occur during the streaming response processing
		case err := <-c.ErrorChan:
			return fmt.Errorf("failed to process completions response stream: %w", err)
		}
	}
}

// storeUserMessage stores the user message in the message storage
func (c *Client) storeUserMessage(sessionID, question string) error {
	userMessage := chat.NewMessage(question, chat.RoleUser, sessionID)
	if err := c.MessageStorage.Write(*userMessage); err != nil {
		return fmt.Errorf("failed to write user message to storage: %w", err)
	}
	return nil
}

// storeAssistantMessage stores the assistant's response message in the message storage
func (c *Client) storeAssistantMessage(sessionID, response string) error {
	assistantMessage := chat.NewMessage(response, chat.RoleAssistant, sessionID)
	if err := c.MessageStorage.Write(*assistantMessage); err != nil {
		return fmt.Errorf("failed to write assistant response message to storage: %w", err)
	}
	return nil
}

// handleNonOKStatus handles non-OK response status
func handleNonOKStatus(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read completion stream response body: %w", err)
		}
		return fmt.Errorf("failed to process API response with body: %s: %w", string(bodyBytes), err)
	}
	return nil
}
