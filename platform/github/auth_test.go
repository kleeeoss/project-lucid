package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func generateTestRSAKeyPEM(t *testing.T) []byte {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	}
	return pem.EncodeToMemory(privBlock)
}

func TestTokenManager_GenerateAppJWT(t *testing.T) {
	pemBytes := generateTestRSAKeyPEM(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tm, err := NewTokenManager("123456", pemBytes, "", logger)
	if err != nil {
		t.Fatalf("NewTokenManager failed: %v", err)
	}

	jwtStr, err := tm.GenerateAppJWT()
	if err != nil {
		t.Fatalf("GenerateAppJWT failed: %v", err)
	}

	// Parse and verify claims
	parsedToken, err := jwt.Parse(jwtStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			t.Fatalf("unexpected signing method: %v", token.Header["alg"])
		}
		rsaKey, _ := jwt.ParseRSAPrivateKeyFromPEM(pemBytes)
		return &rsaKey.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("Failed to parse generated JWT: %v", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		t.Fatalf("Invalid claims in token")
	}

	if claims["iss"] != "123456" {
		t.Errorf("expected iss '123456', got %v", claims["iss"])
	}
}

func TestTokenManager_GetInstallationToken_Caching(t *testing.T) {
	pemBytes := generateTestRSAKeyPEM(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	var apiCallCount atomic.Int32

	// Mock GitHub API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCallCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(installationTokenResponse{
			Token:     "ghs_mock_installation_token_xyz",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		})
	}))
	defer mockServer.Close()

	tm, err := NewTokenManager("123456", pemBytes, mockServer.URL, logger)
	if err != nil {
		t.Fatalf("NewTokenManager failed: %v", err)
	}

	ctx := context.Background()

	// 1. First call: Should hit the mock GitHub API server
	tok1, err := tm.GetInstallationToken(ctx, 999)
	if err != nil {
		t.Fatalf("GetInstallationToken failed: %v", err)
	}
	if tok1 != "ghs_mock_installation_token_xyz" {
		t.Errorf("Unexpected token: %s", tok1)
	}
	if count := apiCallCount.Load(); count != 1 {
		t.Errorf("Expected 1 API call, got %d", count)
	}

	// 2. Second call: Should read from memory cache, ZERO additional API calls
	tok2, err := tm.GetInstallationToken(ctx, 999)
	if err != nil {
		t.Fatalf("GetInstallationToken second call failed: %v", err)
	}
	if tok2 != tok1 {
		t.Errorf("Cached token mismatch: %s vs %s", tok2, tok1)
	}
	if count := apiCallCount.Load(); count != 1 {
		t.Errorf("Expected call count to remain 1 due to cache, got %d", count)
	}
}
