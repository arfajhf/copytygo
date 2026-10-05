package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
)

type Crypt struct{ aead cipher.AEAD }

func NewCrypt(key string) (*Crypt, error) {
	if len(key) < 32 {
		return nil, fmt.Errorf("copytygo: APP_KEY must contain at least 32 characters")
	}
	digest := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(digest[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Crypt{aead: aead}, nil
}
func (c *Crypt) EncryptString(value string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := c.aead.Seal(nil, nonce, []byte(value), nil)
	raw := append(nonce, sealed...)
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
func (c *Crypt) DecryptString(value string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("copytygo: invalid encrypted value")
	}
	n := c.aead.NonceSize()
	if len(raw) <= n {
		return "", fmt.Errorf("copytygo: invalid encrypted value")
	}
	plain, err := c.aead.Open(nil, raw[:n], raw[n:], nil)
	if err != nil {
		return "", fmt.Errorf("copytygo: encrypted value could not be authenticated")
	}
	return string(plain), nil
}
func (c *Crypt) UUIDCos(id int64) (string, error) { return c.EncryptString(strconv.FormatInt(id, 10)) }
func (c *Crypt) ResolveUUIDCos(value string) (int64, error) {
	plain, err := c.DecryptString(value)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(plain, 10, 64)
}
