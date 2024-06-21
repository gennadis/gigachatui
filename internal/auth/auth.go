package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	JSONContentType       = "application/json"
	URLEncodedContentType = "application/x-www-form-urlencoded"
)

const (
	authApiUrl                = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	personalApiScope          = "scope=GIGACHAT_API_PERS"
	errorChanBufferSize       = 100
	rotateTokenTickerInterval = time.Minute * 20
)

type AuthErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Token struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   uint64 `json:"expires_at"`
}

type AuthenticationHandler struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
	Token        Token
	ErrorChan    chan error
}

func NewAuthenticationHandler(ctx context.Context, clientID, clientSecret string) (*AuthenticationHandler, error) {
	authHandler := &AuthenticationHandler{
		httpClient:   &http.Client{},
		ErrorChan:    make(chan error, errorChanBufferSize),
		clientID:     clientID,
		clientSecret: clientSecret,
	}
	initialToken, err := authHandler.getAccessToken(ctx)
	if err != nil {
		slog.Error("Failed to get Access Token", "error", err)
		return nil, err
	}
	authHandler.Token = *initialToken
	return authHandler, nil
}

func (ah *AuthenticationHandler) getAccessToken(ctx context.Context) (*Token, error) {
	payload := strings.NewReader(personalApiScope)
	req, err := http.NewRequestWithContext(ctx, "POST", authApiUrl, payload)
	if err != nil {
		slog.Error("Failed to build auth request", "error", err)
		return nil, err
	}

	reqUUID := uuid.NewString()
	authSecret := generateAuthSecret(ah.clientID, ah.clientSecret)
	authHeader := fmt.Sprintf("Basic %s", authSecret)

	req.Header.Add("Content-Type", URLEncodedContentType)
	req.Header.Add("Accept", JSONContentType)
	req.Header.Add("RqUID", reqUUID)
	req.Header.Add("Authorization", authHeader)

	res, err := ah.httpClient.Do(req)
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

	if res.StatusCode != http.StatusOK {
		authErr := AuthErrorResponse{}
		if err := json.Unmarshal(body, &authErr); err != nil {
			slog.Error("Failed to unmarshal auth error response", "error", err)
			return nil, err
		}
		return nil, fmt.Errorf("Api request failed: status code %d, error code %d, message %s", res.StatusCode, authErr.Code, authErr.Message)
	}

	accessToken := Token{}
	if err := json.Unmarshal(body, &accessToken); err != nil {
		slog.Error("Failed to unmarshal auth response body", "error", err)
		return nil, err
	}
	return &accessToken, nil
}

func (ah *AuthenticationHandler) Run(ctx context.Context) *sync.WaitGroup {
	ticker := time.NewTicker(rotateTokenTickerInterval)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ah.rotateToken(ctx)

			case <-ctx.Done():
				ah.rotateToken(context.Background())
				return

			case err := <-ah.ErrorChan:
				slog.Error("Access token rotation error", "error", err)
			}
		}
	}()

	return wg
}

func (ah *AuthenticationHandler) rotateToken(ctx context.Context) {
	newToken, err := ah.getAccessToken(ctx)
	if err != nil {
		slog.Error("Failed to get new access token for rotation", "error", err)
		ah.ErrorChan <- err
	}

	ah.Token = *newToken
	slog.Info("Access token rotated successfully", slog.Int("new access token is valid to", int(newToken.ExpiresAt)))
}

func generateAuthSecret(clientID, clientSecret string) string {
	authSecret := fmt.Sprintf("%s:%s", clientID, clientSecret)
	encodedAuthStr := base64.StdEncoding.EncodeToString([]byte(authSecret))
	return encodedAuthStr
}
