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

	"github.com/gennadis/gigachatui/internal/config"
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
	HttpClient *http.Client
	Config     *config.Config
	Token      Token
}

func NewClient(cfg config.Config) (*Client, error) {
	apiClient := &Client{
		HttpClient: &http.Client{Timeout: time.Second * 10},
		Config:     &cfg,
	}
	accessToken, err := apiClient.getAccessToken()
	if err != nil {
		slog.Error("Failed to get Access Token", "error", err)
		return nil, err
	}

	apiClient.Token = *accessToken
	return apiClient, nil
}

func (c *Client) GetComplition(ctx context.Context, request *ChatCompletionRequest) (*ChatResponse, error) {
	reqBytes, _ := json.Marshal(request)
	completionsPath := c.Config.BaseURL + "/chat/completions"

	req, err := http.NewRequest("POST", completionsPath, bytes.NewBuffer(reqBytes))
	if err != nil {
		slog.Error("Failed to build send request", "error", err)
		return nil, err
	}

	authHeader := fmt.Sprintf("Bearer %s", c.Token.AccessToken)
	req.Header.Set("Content-Type", JSONContentType)
	req.Header.Set("Accept", JSONContentType)
	req.Header.Set("Authorization", authHeader)

	res, err := c.HttpClient.Do(req)
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

	chatResp := ChatResponse{}
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