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
	"github.com/gennadis/gigachatui/internal/session"
)

const (
	streamRespChunkStartLine = "data: "
	streamRespFinalLine      = "data: [DONE]"
	contentTypeJSON          = "application/json"
)

// Client represents a client for interacting with the GigaChat API
type Client struct {
	httpClient     *http.Client
	Config         *config.Config
	AuthManager    *auth.Manager
	Session        *session.Session
	StreamRespChan chan chat.ResponseStreamChunk
	ErrorChan      chan error
}

// NewClient initializes a new Client instance
func NewClient(ctx context.Context, cfg config.Config, chatName, clientID, clientSecret string) (*Client, error) {
	m, err := auth.NewManager(ctx, clientID, clientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to init authentication manager: %w", err)
	}

	s := session.NewSession(chatName)
	return &Client{
		httpClient:     &http.Client{Timeout: time.Second * 10},
		Config:         &cfg,
		AuthManager:    m,
		Session:        s,
		StreamRespChan: make(chan chat.ResponseStreamChunk),
		ErrorChan:      make(chan error),
	}, nil
}

// GetCompletion sends a question to the chat API and processes the response
func (c *Client) GetCompletion(ctx context.Context, question string) error {
	userMsg := chat.Message{Role: chat.RoleUser, Content: question}
	c.Session.Messages = append(c.Session.Messages, userMsg)
	request := chat.NewDefaultRequest(c.Session.Messages)

	resp, err := c.callCompletionsAPI(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to get chat completion: %w", err)
	}

	go c.processCompletionsAPIResponse(resp)

	var assistantRespTxt strings.Builder
	for {
		select {
		case chunk := <-c.StreamRespChan:
			if chunk.Final {
				assistantRespMsg := chat.Message{Role: chat.RoleAssistant, Content: assistantRespTxt.String()}
				c.Session.Messages = append(c.Session.Messages, assistantRespMsg)
				return nil
			}
			chunkContent := chunk.Choices[0].Delta.Content
			assistantRespTxt.WriteString(chunkContent)
			fmt.Print(chunkContent)

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
