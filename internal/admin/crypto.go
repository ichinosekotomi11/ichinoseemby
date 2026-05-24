package admin

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/bcrypt"
)

const dataKeyEnv = "EDEN_DATA_KEY"

type FieldCipher struct {
	aead cipher.AEAD
}

var defaultFieldCipher *FieldCipher

func LoadFieldCipherFromEnv() (*FieldCipher, error) {
	raw := os.Getenv(dataKeyEnv)
	if raw == "" {
		return nil, fmt.Errorf("%s is required for encrypted fields", dataKeyEnv)
	}
	key, err := decodeDataKey(raw)
	if err != nil {
		return nil, err
	}
	return NewFieldCipher(key)
}

func NewFieldCipher(key []byte) (*FieldCipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("AES-256-GCM key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &FieldCipher{aead: aead}, nil
}

func SetDefaultFieldCipher(cipher *FieldCipher) {
	defaultFieldCipher = cipher
}

func (c *FieldCipher) EncryptString(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plain), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (c *FieldCipher) DecryptString(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	payload, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	nonceSize := c.aead.NonceSize()
	if len(payload) < nonceSize {
		return "", errors.New("encrypted payload is shorter than nonce")
	}
	plain, err := c.aead.Open(nil, payload[:nonceSize], payload[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

type EncryptedString struct {
	Plain string
}

func (s *EncryptedString) Scan(value any) error {
	if defaultFieldCipher == nil {
		return errors.New("default field cipher is not configured")
	}
	if value == nil {
		s.Plain = ""
		return nil
	}
	var encoded string
	switch v := value.(type) {
	case string:
		encoded = v
	case []byte:
		encoded = string(v)
	default:
		return fmt.Errorf("unsupported encrypted string type %T", value)
	}
	plain, err := defaultFieldCipher.DecryptString(encoded)
	if err != nil {
		return err
	}
	s.Plain = plain
	return nil
}

func (s EncryptedString) Value() (driver.Value, error) {
	if defaultFieldCipher == nil {
		return nil, errors.New("default field cipher is not configured")
	}
	return defaultFieldCipher.EncryptString(s.Plain)
}

func decodeDataKey(raw string) ([]byte, error) {
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(raw); err == nil {
		return decoded, nil
	}
	return []byte(raw), nil
}
