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
	ExpiresAt int64             `json:"exp"`
	Data      map[string]string `json:"data,omitempty"`
}

func SignToken(secret string, claims Claims) (string, error) {
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
	if claims.ExpiresAt < time.Now().Unix() {
		return claims, fmt.Errorf("copytygo: token expired")
	}
	return claims, nil
}
