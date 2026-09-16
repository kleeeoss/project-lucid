package github

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingAppID      = errors.New("GITHUB_APP_ID is required")
	ErrMissingPrivateKey = errors.New("GITHUB_PRIVATE_KEY or GITHUB_PRIVATE_KEY_PATH is required")
	ErrTokenExchange     = errors.New("failed to exchange JWT for installation access token")
)

// TokenManager coordinates GitHub App RS256 JWT signing and Installation Access Token caching.
type TokenManager interface {
	GetInstallationToken(ctx context.Context, installationID int64) (string, error)
	GenerateAppJWT() (string, error)
}

type cachedToken struct {
	token     string
	expiresAt time.Time
}

type tokenManager struct {
	appID      string
	privateKey *rsa.PrivateKey
	apiBaseURL string
	httpClient *http.Client
	logger     *slog.Logger
	cacheMu    sync.RWMutex
	cache      map[int64]*cachedToken
}

// NewTokenManager initializes the authentication manager with the RSA private key.
func NewTokenManager(appID string, privateKeyPEM []byte, apiBaseURL string, logger *slog.Logger) (TokenManager, error) {
	if appID == "" {
		return nil, ErrMissingAppID
	}
	if len(privateKeyPEM) == 0 {
		return nil, ErrMissingPrivateKey
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA private key PEM: %w", err)
	}

	if apiBaseURL == "" {
		apiBaseURL = "https://api.github.com"
	}

	return &tokenManager{
		appID:      appID,
		privateKey: key,
		apiBaseURL: apiBaseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
		cache:      make(map[int64]*cachedToken),
	}, nil
}

// NewTokenManagerFromEnv loads App credentials from environment variables.
func NewTokenManagerFromEnv(logger *slog.Logger) (TokenManager, error) {
	appID := os.Getenv("GITHUB_APP_ID")
	privKeyData := os.Getenv("GITHUB_PRIVATE_KEY")

	if privKeyData == "" {
		path := os.Getenv("GITHUB_PRIVATE_KEY_PATH")
		if path != "" {
			b, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read private key from path %s: %w", path, err)
			}
			privKeyData = string(b)
		}
	}

	apiURL := os.Getenv("GITHUB_API_URL")
	if apiURL == "" {
		apiURL = "https://api.github.com"
	}

	return NewTokenManager(appID, []byte(privKeyData), apiURL, logger)
}

// GenerateAppJWT creates a short-lived (9-minute) RS256 JWT signed with the App's private key.
func (tm *tokenManager) GenerateAppJWT() (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"iat": now.Add(-60 * time.Second).Unix(), // 60s in the past for clock drift
		"exp": now.Add(9 * time.Minute).Unix(),   // Max 10 minutes allowed by GitHub
		"iss": tm.appID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(tm.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signed, nil
}

type installationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GetInstallationToken returns a valid Installation Access Token, utilizing in-memory cache
// and proactively refreshing when less than 5 minutes remain before expiration.
func (tm *tokenManager) GetInstallationToken(ctx context.Context, installationID int64) (string, error) {
	tm.cacheMu.RLock()
	cached, exists := tm.cache[installationID]
	if exists && time.Until(cached.expiresAt) > 5*time.Minute {
		tm.cacheMu.RUnlock()
		return cached.token, nil
	}
	tm.cacheMu.RUnlock()

	tm.cacheMu.Lock()
	defer tm.cacheMu.Unlock()

	// Double-check cache after acquiring write lock
	cached, exists = tm.cache[installationID]
	if exists && time.Until(cached.expiresAt) > 5*time.Minute {
		return cached.token, nil
	}

	// 1. Generate fresh RS256 JWT
	jwtToken, err := tm.GenerateAppJWT()
	if err != nil {
		return "", fmt.Errorf("failed to create App JWT: %w", err)
	}

	// 2. Call GitHub API to exchange for Installation Access Token
	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", tm.apiBaseURL, installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create access token request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := tm.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("installation token exchange network failure: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		tm.logger.Error("GitHub token exchange rejected",
			"status_code", resp.StatusCode,
			"installation_id", installationID,
			"body", string(body),
		)
		return "", fmt.Errorf("%w: status %d", ErrTokenExchange, resp.StatusCode)
	}

	var tokenResp installationTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode installation token response: %w", err)
	}

	// 3. Cache the token
	tm.cache[installationID] = &cachedToken{
		token:     tokenResp.Token,
		expiresAt: tokenResp.ExpiresAt,
	}

	tm.logger.Info("Acquired fresh GitHub Installation Access Token",
		"installation_id", installationID,
		"expires_at", tokenResp.ExpiresAt,
	)

	return tokenResp.Token, nil
}
