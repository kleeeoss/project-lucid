package ingestion

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
)

var (
	ErrMissingSignature  = errors.New("missing X-Hub-Signature-256 header")
	ErrInvalidHeader     = errors.New("invalid signature header format: expected sha256=<hex>")
	ErrSignatureMismatch = errors.New("HMAC signature mismatch")
)

// VerifySignature validates that the given payload bytes match the signature
// provided in the X-Hub-Signature-256 header using the shared secret.
// It uses constant-time comparison to prevent timing side-channel attacks.
func VerifySignature(payload []byte, signatureHeader, secret string) error {
	if signatureHeader == "" {
		return ErrMissingSignature
	}

	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return ErrInvalidHeader
	}

	receivedHexSig := strings.TrimPrefix(signatureHeader, "sha256=")
	receivedBytes, err := hex.DecodeString(receivedHexSig)
	if err != nil {
		return ErrInvalidHeader
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedBytes := mac.Sum(nil)

	// ConstantTimeCompare returns 1 if both slices are equal, 0 otherwise
	if subtle.ConstantTimeCompare(receivedBytes, expectedBytes) != 1 {
		return ErrSignatureMismatch
	}

	return nil
}
