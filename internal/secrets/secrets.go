package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
)

const (
	EncryptedPrefix = "gurlenc:v1:"
	NonceSize       = 12
	KeySize         = 32
	gcmTagSize      = 16
)

func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func RandomBytes(size int) ([]byte, error) {
	value := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, value); err != nil {
		return nil, err
	}
	return value, nil
}

func GetOrCreateKeyAt(keyPath string) ([]byte, error) {
	if keyPath == "" {
		return nil, fmt.Errorf("key path is required")
	}

	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(keyPath)
	if err == nil && len(data) == KeySize {
		return data, nil
	}

	if !errors.Is(err, os.ErrNotExist) && err != nil {
		return nil, err
	}

	key, err := GenerateKey()
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, err
	}

	return key, nil
}

func Encrypt(key []byte, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return EncryptedPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(key []byte, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	encoded := ciphertext
	if len(encoded) >= len(EncryptedPrefix) && encoded[:len(EncryptedPrefix)] == EncryptedPrefix {
		encoded = encoded[len(EncryptedPrefix):]
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func IsEncryptedValue(value string) bool {
	if value == "" {
		return false
	}
	if len(value) >= len(EncryptedPrefix) && value[:len(EncryptedPrefix)] == EncryptedPrefix {
		data, err := base64.StdEncoding.DecodeString(value[len(EncryptedPrefix):])
		return err == nil && len(data) > NonceSize+gcmTagSize
	}

	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return false
	}
	return len(data) > NonceSize+gcmTagSize
}

func Mask(value string) string {
	if value == "" {
		return ""
	}
	return "*****"
}

func EncryptValueIfNeeded(key []byte, value string, isSecret bool) (string, error) {
	if !isSecret {
		return value, nil
	}
	return Encrypt(key, value)
}

func DecryptValueIfNeeded(key []byte, value string, isSecret bool) (string, error) {
	if !isSecret {
		return value, nil
	}
	return Decrypt(key, value)
}

func PBKDF2SHA256(passphrase string, salt []byte, iterations int, keyLen int) []byte {
	return pbkdf2([]byte(passphrase), salt, iterations, keyLen, sha256.New)
}

func pbkdf2(password []byte, salt []byte, iter int, keyLen int, h func() hash.Hash) []byte {
	if iter <= 0 {
		iter = 1
	}
	prf := hmac.New(h, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen
	var derived []byte
	var blockIndex [4]byte

	for block := 1; block <= numBlocks; block++ {
		blockIndex[0] = byte(block >> 24)
		blockIndex[1] = byte(block >> 16)
		blockIndex[2] = byte(block >> 8)
		blockIndex[3] = byte(block)

		prf.Reset()
		prf.Write(salt)
		prf.Write(blockIndex[:])
		u := prf.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)

		for i := 1; i < iter; i++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		derived = append(derived, t...)
	}

	return derived[:keyLen]
}
