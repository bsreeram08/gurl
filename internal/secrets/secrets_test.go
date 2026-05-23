package secrets

import (
	"encoding/base64"
	"testing"
)

func TestIsEncryptedValueRequiresValidPrefixedPayload(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	ciphertext, err := Encrypt(key, "secret")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if !IsEncryptedValue(ciphertext) {
		t.Fatal("encrypted value should be detected")
	}

	tests := []string{
		"plain-text-value",
		EncryptedPrefix,
		EncryptedPrefix + "not-base64",
		EncryptedPrefix + base64.StdEncoding.EncodeToString([]byte("short")),
	}
	for _, value := range tests {
		if IsEncryptedValue(value) {
			t.Fatalf("expected %q to be treated as plaintext", value)
		}
	}
}
