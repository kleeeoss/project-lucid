package ingestion

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
)

func computeTestSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	secret := "test_webhook_secret_12345"
	payload := []byte(`{"action":"opened","number":42}`)

	t.Run("valid signature passes", func(t *testing.T) {
		sig := computeTestSignature(payload, secret)
		err := VerifySignature(payload, sig, secret)
		if err != nil {
			t.Fatalf("expected nil error for valid signature, got: %v", err)
		}
	})

	t.Run("missing header returns ErrMissingSignature", func(t *testing.T) {
		err := VerifySignature(payload, "", secret)
		if !errors.Is(err, ErrMissingSignature) {
			t.Fatalf("expected ErrMissingSignature, got: %v", err)
		}
	})

	t.Run("invalid header format without sha256 prefix", func(t *testing.T) {
		err := VerifySignature(payload, "invalid_header", secret)
		if !errors.Is(err, ErrInvalidHeader) {
			t.Fatalf("expected ErrInvalidHeader, got: %v", err)
		}
	})

	t.Run("tampered payload fails", func(t *testing.T) {
		sig := computeTestSignature(payload, secret)
		tamperedPayload := []byte(`{"action":"opened","number":99}`)
		err := VerifySignature(tamperedPayload, sig, secret)
		if !errors.Is(err, ErrSignatureMismatch) {
			t.Fatalf("expected ErrSignatureMismatch, got: %v", err)
		}
	})

	t.Run("tampered secret fails", func(t *testing.T) {
		sig := computeTestSignature(payload, secret)
		err := VerifySignature(payload, sig, "different_secret")
		if !errors.Is(err, ErrSignatureMismatch) {
			t.Fatalf("expected ErrSignatureMismatch, got: %v", err)
		}
	})

	t.Run("non-hex characters in signature fail", func(t *testing.T) {
		err := VerifySignature(payload, "sha256=not_valid_hex!@#$", secret)
		if !errors.Is(err, ErrInvalidHeader) {
			t.Fatalf("expected ErrInvalidHeader, got: %v", err)
		}
	})
}
