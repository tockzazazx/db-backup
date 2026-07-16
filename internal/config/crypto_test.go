package config

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	const secret = "example-secret-token-1234"

	enc, err := EncryptSecret(secret)
	if err != nil {
		t.Fatalf("EncryptSecret: %v", err)
	}
	if !strings.HasPrefix(enc, encPrefix) {
		t.Errorf("encrypted value missing prefix: %q", enc)
	}
	if strings.Contains(enc, secret) {
		t.Error("encrypted value leaks the plaintext")
	}

	dec, err := DecryptSecret(enc)
	if err != nil {
		t.Fatalf("DecryptSecret: %v", err)
	}
	if dec != secret {
		t.Errorf("round trip = %q; want %q", dec, secret)
	}

	// Two encryptions of the same value must differ (random nonce).
	enc2, _ := EncryptSecret(secret)
	if enc == enc2 {
		t.Error("two encryptions produced identical ciphertext")
	}
}

func TestDecryptSecretErrors(t *testing.T) {
	for _, bad := range []string{
		"plaintext-token",     // no prefix
		encPrefix + "!!!",     // not base64
		encPrefix + "c2hvcnQ", // too short
	} {
		if _, err := DecryptSecret(bad); err == nil {
			t.Errorf("DecryptSecret(%q) should fail", bad)
		}
	}
}
