package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const (
	authApiUrl       = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	personalApiScope = "scope=GIGACHAT_API_PERS"
)

type Token struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   uint64 `json:"expires_at"`
}

func generateAuthSecret(clientID, clientSecret string) string {
	authSecret := fmt.Sprintf("%s:%s", clientID, clientSecret)
	encodedAuthStr := base64.StdEncoding.EncodeToString([]byte(authSecret))
	return encodedAuthStr
}

func (c *Client) getAccessToken() (*Token, error) {
	payload := strings.NewReader(personalApiScope)
	req, err := http.NewRequest("POST", authApiUrl, payload)
	if err != nil {
		slog.Error("Failed to build auth request", "error", err)
		return nil, err
	}

	reqUUID := uuid.NewString()
	authSecret := generateAuthSecret(c.Config.ClientID, c.Config.ClientSecret)
	authHeader := fmt.Sprintf("Basic %s", authSecret)

	req.Header.Add("Content-Type", URLEncodedContentType)
	req.Header.Add("Accept", JSONContentType)
	req.Header.Add("RqUID", reqUUID)
	req.Header.Add("Authorization", authHeader)

	res, err := c.HttpClient.Do(req)
	if err != nil {
		slog.Error("Failed to send auth request", "error", err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("Failed to read auth response body", "error", err)
		return nil, err
	}

	if err := handleApiError(res, body); err != nil {
		slog.Error("Failed to get Access Token", "error", err)
		return nil, err
	}

	accessToken := Token{}
	if err := json.Unmarshal(body, &accessToken); err != nil {
		slog.Error("Failed to unmarshal auth response body", "error", err)
		return nil, err
	}
	return &accessToken, nil
}
