package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gennadis/gigachatui/internal/auth"
	"github.com/gennadis/gigachatui/internal/chat"
	"github.com/gennadis/gigachatui/internal/config"
	"github.com/gennadis/gigachatui/internal/session"
)

const (
	streamResponseChunkStartLine = "data: "
	streamResponseFinalLine      = "data: [DONE]"
	JSONContentType              = "application/json"
)

type Client struct {
	httpClient         *http.Client
	Config             *config.Config
	AuthHandler        *auth.AuthenticationHandler
	Session            *session.Session
	StreamResponseChan chan chat.ChatResponseStreamChunk
	ErrorChan          chan error
}

func NewClient(ctx context.Context, chatName string, clientID string, clientSecret string, cfg config.Config) (*Client, error) {
	authHandler, err := auth.NewAuthenticationHandler(ctx, clientID, clientSecret)
	if err != nil {
		slog.Error("Failed to init authentication handler", "error", err)
	}
	newSession := session.NewSession(chatName)
	return &Client{
		httpClient:         &http.Client{Timeout: time.Second * 10},
		Config:             &cfg,
		AuthHandler:        authHandler,
		Session:            newSession,
		StreamResponseChan: make(chan chat.ChatResponseStreamChunk),
		ErrorChan:          make(chan error),
	}, nil
}

func (c *Client) GetCompletion(ctx context.Context, question string) error {
	userMsg := chat.ChatMessage{Role: chat.ChatRoleUser, Content: question}
	c.Session.Messages = append(c.Session.Messages, userMsg)
	request := chat.NewDefaultChatRequest(c.Session.Messages)

	resp, err := c.callCompletionsAPI(ctx, request)
	if err != nil {
		return fmt.Errorf("Failed to get chat completion: %w", err)
	}

	go c.processCompletionsAPIResponse(resp)

	var assistantRespTxt strings.Builder

	for {
		select {
		case chunk := <-c.StreamResponseChan:
			if chunk.Final {
				assistantRespMsg := chat.ChatMessage{Role: chat.ChatRoleAssistant, Content: assistantRespTxt.String()}
				c.Session.Messages = append(c.Session.Messages, assistantRespMsg)
				return nil
			}
			chunkContent := chunk.Choices[0].Delta.Content
			assistantRespTxt.WriteString(chunkContent)
			fmt.Print(chunkContent)

		case err := <-c.ErrorChan:
			slog.Error("Failed to process completions response stream", "error", err)
			return err
		}
	}
}

func (c *Client) callCompletionsAPI(ctx context.Context, request *chat.ChatRequest) (*http.Response, error) {
	reqBytes, _ := json.Marshal(request)
	completionsPath := c.Config.BaseURL + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, "POST", completionsPath, bytes.NewBuffer(reqBytes))
	if err != nil {
		slog.Error("Failed to build completion API request", "error", err)
		return nil, err
	}

	authHeader := fmt.Sprintf("Bearer %s", c.AuthHandler.Token.AccessToken)
	req.Header.Set("Content-Type", JSONContentType)
	req.Header.Set("Accept", JSONContentType)
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("Failed to send request", "error", err)
		return nil, err
	}

	return resp, nil
}

func (c *Client) processCompletionsAPIResponse(resp *http.Response) {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("Failed to read completion stream response body", "error", err)
			c.ErrorChan <- err
			return
		}
		slog.Error("Failed to process API request response", "response_body", string(bodyBytes))
		c.ErrorChan <- fmt.Errorf(string(bodyBytes))
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		ln := scanner.Text()
		// fmt.Println("Received line:", ln) // Debugging line to see the raw response

		if ln == streamResponseFinalLine {
			finalRespChunk := chat.ChatResponseStreamChunk{Final: true}
			c.StreamResponseChan <- finalRespChunk
			return
		}

		if strings.HasPrefix(ln, streamResponseChunkStartLine) || ln == "\n" {
			jsonStr := strings.TrimPrefix(ln, streamResponseChunkStartLine)
			var respChunk chat.ChatResponseStreamChunk
			if err := json.Unmarshal([]byte(jsonStr), &respChunk); err != nil {
				slog.Error("Failed to unmarshal completion stream response chunk", "error", err)
				c.ErrorChan <- err
				return
			}
			respChunk.Final = false
			c.StreamResponseChan <- respChunk
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Failed to scan completion stream response chunk", "error", err)
		c.ErrorChan <- err
	}
}
