package auth

import (
	"context"
	"crypto/tls"
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
	contentTypeJSON       = "application/json"
	contentTypeURLEncoded = "application/x-www-form-urlencoded"

	authAPIURL                = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	personalAPIScope          = "scope=GIGACHAT_API_PERS"
	rotateTokenTickerInterval = time.Minute * 20
)

// errorResponse represents an error response from the authentication API
type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Token represents an access token
type Token struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   uint64 `json:"expires_at"`
}

// Manager handles authentication and token rotation
type Manager struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
	Token        Token
	ErrorChan    chan error
}

// NewManager creates a new AuthenticationHandler instance
func NewManager(ctx context.Context, clientID, clientSecret string) (*Manager, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec

	m := &Manager{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{},
		ErrorChan:    make(chan error),
	}
	t, err := m.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial access token: %w", err)
	}
	m.Token = *t
	return m, nil
}

// getToken retrieves a new access token from the authentication API
func (m *Manager) getToken(ctx context.Context) (*Token, error) {
	payload := strings.NewReader(personalAPIScope)
	req, err := http.NewRequestWithContext(ctx, "POST", authAPIURL, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to build authentication request: %w", err)
	}

	reqUUID := uuid.NewString()
	authSecret := generateAuthSecret(m.clientID, m.clientSecret)
	authHeader := fmt.Sprintf("Basic %s", authSecret)

	req.Header.Add("Content-Type", contentTypeURLEncoded)
	req.Header.Add("Accept", contentTypeJSON)
	req.Header.Add("RqUID", reqUUID)
	req.Header.Add("Authorization", authHeader)

	res, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send authentication request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read authentication response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		var authErr errorResponse
		if err := json.Unmarshal(body, &authErr); err != nil {
			return nil, fmt.Errorf("failed to unmarshal authentication error response: %w", err)
		}
		return nil, fmt.Errorf("failed API request: status code %d, error code %d, message %s", res.StatusCode, authErr.Code, authErr.Message)
	}

	var t Token
	if err := json.Unmarshal(body, &t); err != nil {
		return nil, fmt.Errorf("failed to unmarshal authentication response: %w", err)
	}
	return &t, nil
}

// Run starts the token rotation process
func (m *Manager) Run(ctx context.Context) *sync.WaitGroup {
	t := time.NewTicker(rotateTokenTickerInterval)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer t.Stop()

		for {
			select {
			case <-t.C:
				m.rotateToken(ctx)

			case <-ctx.Done():
				m.rotateToken(context.Background())
				return

			case err := <-m.ErrorChan:
				slog.Error("token rotation error", "error", err)
			}
		}
	}()

	return wg
}

// rotateToken retrieves a new access token and updates the current token
func (m *Manager) rotateToken(ctx context.Context) {
	newToken, err := m.getToken(ctx)
	if err != nil {
		m.ErrorChan <- fmt.Errorf("failed to get new token for rotation: %w", err)
		return
	}

	m.Token = *newToken
	slog.Info("token rotated successfully", slog.Int("new token is valid to", int(newToken.ExpiresAt)))
}

// generateAuthSecret generates the base64 encoded authentication secret
func generateAuthSecret(clientID, clientSecret string) string {
	authSecret := fmt.Sprintf("%s:%s", clientID, clientSecret)
	return base64.StdEncoding.EncodeToString([]byte(authSecret))
}
