package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Claims struct {
	Subject   string            `json:"sub"`
	Role      string            `json:"role,omitempty"`
	IssuedAt  int64             `json:"iat"`
	ExpiresAt int64             `json:"exp"`
	Data      map[string]string `json:"data,omitempty"`
}

func SignToken(secret string, claims Claims) (string, error) {
	if len(secret) < 32 {
		return "", fmt.Errorf("copytygo: token secret must be at least 32 characters")
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return "", fmt.Errorf("copytygo: token subject is required")
	}
	if claims.IssuedAt == 0 {
		claims.IssuedAt = time.Now().Unix()
	}
	if claims.ExpiresAt == 0 {
		claims.ExpiresAt = time.Now().Add(24 * time.Hour).Unix()
	}
	raw, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + sig, nil
}
func VerifyToken(secret, token string) (Claims, error) {
	var claims Claims
	if len(secret) < 32 {
		return claims, fmt.Errorf("copytygo: token secret must be at least 32 characters")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return claims, fmt.Errorf("copytygo: invalid token")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(expected, actual) {
		return claims, fmt.Errorf("copytygo: invalid token signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return claims, err
	}
	if err = json.Unmarshal(raw, &claims); err != nil {
		return claims, err
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return claims, fmt.Errorf("copytygo: invalid token subject")
	}
	if claims.IssuedAt > time.Now().Add(time.Minute).Unix() {
		return claims, fmt.Errorf("copytygo: token issued in the future")
	}
	if claims.ExpiresAt <= time.Now().Unix() {
		return claims, fmt.Errorf("copytygo: token expired")
	}
	return claims, nil
}
