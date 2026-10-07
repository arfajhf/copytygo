package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/core"
)

type Manager struct {
	Name     string
	TTL      time.Duration
	Secure   bool
	SameSite http.SameSite
}

func New() *Manager {
	return &Manager{
		Name:     "copytygo_session",
		TTL:      24 * time.Hour,
		Secure:   strings.EqualFold(config.Get("APP_ENV", "local"), "production"),
		SameSite: http.SameSiteLaxMode,
	}
}

func (m *Manager) Read(ctx *core.Context) map[string]string {
	raw, ok := ctx.Cookie(m.Name)
	if !ok || raw == "" {
		return map[string]string{}
	}
	data, ok := verify(raw, []byte(config.Get("APP_KEY")))
	if !ok {
		return map[string]string{}
	}
	var values map[string]string
	if json.Unmarshal(data, &values) != nil || values == nil {
		return map[string]string{}
	}
	return values
}

func (m *Manager) Write(ctx *core.Context, values map[string]string) error {
	key := []byte(config.Get("APP_KEY"))
	if len(key) < 32 {
		return errors.New("copytygo session: APP_KEY must be at least 32 characters")
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return err
	}
	value := sign(raw, key)
	cookie := &http.Cookie{
		Name:     m.Name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.Secure,
		SameSite: m.SameSite,
		MaxAge:   int(m.TTL.Seconds()),
	}
	ctx.SetCookie(cookie)
	return nil
}

func (m *Manager) Forget(ctx *core.Context) {
	ctx.SetCookie(&http.Cookie{
		Name:     m.Name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   m.Secure,
		SameSite: m.SameSite,
		MaxAge:   -1,
	})
}

func sign(data, key []byte) string {
	payload := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + sig
}

func verify(value string, key []byte) ([]byte, bool) {
	if len(key) < 32 {
		return nil, false
	}
	payload, sig, ok := strings.Cut(value, ".")
	if !ok {
		return nil, false
	}
	expectedMAC := hmac.New(sha256.New, key)
	_, _ = expectedMAC.Write([]byte(payload))
	expected := expectedMAC.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(actual, expected) {
		return nil, false
	}
	data, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, false
	}
	return data, true
}
