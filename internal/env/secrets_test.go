package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	// Test that encrypting then decrypting returns the original value
	testCases := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple value",
			plaintext: "secret-api-key-12345",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "unicode value",
			plaintext: "密码🔐",
		},
		{
			name:      "long value",
			plaintext: "a very long secret value with lots of characters that should still encrypt and decrypt correctly",
		},
		{
			name:      "special characters",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get or create machine key
			key, err := GetOrCreateMachineKey()
			if err != nil {
				t.Fatalf("GetOrCreateMachineKey failed: %v", err)
			}

			// Encrypt
			ciphertext, err := EncryptSecret(key, tc.plaintext)
			if err != nil {
				t.Fatalf("EncryptSecret failed: %v", err)
			}

			// Ciphertext should be different from plaintext (unless empty)
			if tc.plaintext != "" && ciphertext == tc.plaintext {
				t.Error("ciphertext should differ from plaintext")
			}

			// Decrypt
			decrypted, err := DecryptSecret(key, ciphertext)
			if err != nil {
				t.Fatalf("DecryptSecret failed: %v", err)
			}

			// Should match original
			if decrypted != tc.plaintext {
				t.Errorf("round-trip failed: got %q, want %q", decrypted, tc.plaintext)
			}
		})
	}
}

func TestEncryptSecretProducesDifferentOutput(t *testing.T) {
	// Same plaintext should produce different ciphertext due to random nonce
	key, err := GetOrCreateMachineKey()
	if err != nil {
		t.Fatalf("GetOrCreateMachineKey failed: %v", err)
	}

	plaintext := "same-secret"

	ciphertext1, err := EncryptSecret(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	ciphertext2, err := EncryptSecret(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	// Due to random nonce, same plaintext should produce different ciphertext
	if ciphertext1 == ciphertext2 {
		t.Error("ciphertexts should differ due to random nonce")
	}

	// But both should decrypt to same value
	decrypted1, err := DecryptSecret(key, ciphertext1)
	if err != nil {
		t.Fatalf("DecryptSecret failed: %v", err)
	}
	decrypted2, err := DecryptSecret(key, ciphertext2)
	if err != nil {
		t.Fatalf("DecryptSecret failed: %v", err)
	}

	if decrypted1 != decrypted2 || decrypted1 != plaintext {
		t.Errorf("both should decrypt to %q, got %q and %q", plaintext, decrypted1, decrypted2)
	}
}

func TestGetOrCreateMachineKey(t *testing.T) {
	// Use a temp directory to avoid conflicts
	tmpDir, err := os.MkdirTemp("", "gurl-test-key-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	keyPath := filepath.Join(tmpDir, ".secret-key")

	// First call should create the key file
	key1, err := getOrCreateMachineKeyAt(keyPath)
	if err != nil {
		t.Fatalf("first GetOrCreateMachineKey failed: %v", err)
	}

	if len(key1) != 32 {
		t.Errorf("key should be 32 bytes, got %d", len(key1))
	}

	// Key file should exist
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("key file should exist after first call")
	}

	// Second call should return the same key
	key2, err := getOrCreateMachineKeyAt(keyPath)
	if err != nil {
		t.Fatalf("second GetOrCreateMachineKey failed: %v", err)
	}

	if string(key1) != string(key2) {
		t.Error("subsequent calls should return the same key")
	}
}

func TestMaskSecret(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "non-empty secret",
			input:    "my-secret-value",
			expected: "*****",
		},
		{
			name:     "empty secret",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := MaskSecret(tc.input)
			if result != tc.expected {
				t.Errorf("MaskSecret(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestIsEncryptedValue(t *testing.T) {
	// This is a helper to check if a stored value looks encrypted
	// Encrypted values with AES-GCM have a specific format: nonce (12 bytes) + ciphertext + tag (16 bytes)

	key, _ := GetOrCreateMachineKey()
	ciphertext, _ := EncryptSecret(key, "secret")

	if !IsEncryptedValue(ciphertext) {
		t.Error("encrypted value should be detected as encrypted")
	}

	if IsEncryptedValue("plain-text-value") {
		t.Error("plain text should not be detected as encrypted")
	}
}

func TestEncryptDecryptWithDifferentKeys(t *testing.T) {
	// Create two different key files
	tmpDir1, _ := os.MkdirTemp("", "gurl-test-key1-*")
	tmpDir2, _ := os.MkdirTemp("", "gurl-test-key2-*")
	defer os.RemoveAll(tmpDir1)
	defer os.RemoveAll(tmpDir2)

	keyPath1 := filepath.Join(tmpDir1, ".secret-key")
	keyPath2 := filepath.Join(tmpDir2, ".secret-key")

	key1, _ := getOrCreateMachineKeyAt(keyPath1)
	key2, _ := getOrCreateMachineKeyAt(keyPath2)

	// Encrypt with key1
	plaintext := "secret-message"
	ciphertext, err := EncryptSecret(key1, plaintext)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	// Decrypting with wrong key should fail
	_, err = DecryptSecret(key2, ciphertext)
	if err == nil {
		t.Error("decrypting with wrong key should fail")
	}

	// But decrypting with correct key should work
	decrypted, err := DecryptSecret(key1, ciphertext)
	if err != nil {
		t.Fatalf("DecryptSecret with correct key failed: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("got %q, want %q", decrypted, plaintext)
	}
}
