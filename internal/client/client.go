package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gennadis/gigachatui/internal/auth"
	"github.com/gennadis/gigachatui/internal/chat"
	"github.com/gennadis/gigachatui/internal/config"
	"github.com/gennadis/gigachatui/internal/session"
)

const (
	JSONContentType       = "application/json"
	URLEncodedContentType = "application/x-www-form-urlencoded"
)

type ApiErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Client struct {
	httpClient  *http.Client
	Config      *config.Config
	AuthHandler *auth.AuthenticationHandler
	Session     *session.Session
}

func NewClient(ctx context.Context, chatName string, clientID string, clientSecret string, cfg config.Config) (*Client, error) {
	authHandler, err := auth.NewAuthenticationHandler(ctx, clientID, clientSecret)
	if err != nil {
		slog.Error("Failed to init authentication handler", "error", err)
	}
	newSession := session.NewSession(chatName)
	return &Client{
		httpClient:  &http.Client{Timeout: time.Second * 10},
		Config:      &cfg,
		AuthHandler: authHandler,
		Session:     newSession,
	}, nil
}

func (c *Client) GetCompletion(ctx context.Context, request *chat.ChatRequest) (*chat.ChatResponse, error) {
	reqBytes, _ := json.Marshal(request)
	completionsPath := c.Config.BaseURL + "/chat/completions"

	req, err := http.NewRequest("POST", completionsPath, bytes.NewBuffer(reqBytes))
	if err != nil {
		slog.Error("Failed to build send request", "error", err)
		return nil, err
	}

	authHeader := fmt.Sprintf("Bearer %s", c.AuthHandler.Token.AccessToken)
	req.Header.Set("Content-Type", JSONContentType)
	req.Header.Set("Accept", JSONContentType)
	req.Header.Set("Authorization", authHeader)

	res, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("Failed to send request", "error", err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("Failed to read response body", "error", err)
		return nil, err
	}

	if err := handleApiError(res, body); err != nil {
		slog.Error("Failed to send completion request", "error", err)
		return nil, err
	}

	chatResp := chat.ChatResponse{}
	if err := json.Unmarshal(body, &chatResp); err != nil {
		slog.Error("Failed to unmarshal chat response body", "error", err)
		return nil, err
	}

	return &chatResp, nil
}

func handleApiError(res *http.Response, body []byte) error {
	if res.StatusCode != http.StatusOK {
		authErr := ApiErrorResponse{}
		if err := json.Unmarshal(body, &authErr); err != nil {
			slog.Error("Failed to unmarshal auth error response", "error", err)
			return err
		}
		return fmt.Errorf("Api request failed: status code %d, message %s", res.StatusCode, authErr.Message)
	}
	return nil
}
