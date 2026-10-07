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
	"github.com/arfajhf/copytygo/v4/security"
)

const sessionContextPrefix = "copytygo.session.state:"

type Manager struct {
	Name     string
	TTL      time.Duration
	Secure   bool
	SameSite http.SameSite
}

type payload struct {
	Values map[string]string `json:"values,omitempty"`
	Flash  map[string]string `json:"flash,omitempty"`
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
	session := m.readPayload(ctx)
	return clone(session.Values)
}

func (m *Manager) Get(ctx *core.Context, key string) string {
	return m.readPayload(ctx).Values[key]
}

func (m *Manager) Put(ctx *core.Context, key, value string) error {
	session := m.readPayload(ctx)
	session.Values[key] = value
	return m.writePayload(ctx, session)
}

func (m *Manager) Remove(ctx *core.Context, key string) error {
	session := m.readPayload(ctx)
	delete(session.Values, key)
	return m.writePayload(ctx, session)
}

func (m *Manager) Write(ctx *core.Context, values map[string]string) error {
	session := m.readPayload(ctx)
	session.Values = clone(values)
	return m.writePayload(ctx, session)
}

func (m *Manager) Flash(ctx *core.Context, key, value string) error {
	session := m.readPayload(ctx)
	session.Flash[key] = value
	return m.writePayload(ctx, session)
}

func (m *Manager) PullFlash(ctx *core.Context, key string) (string, bool, error) {
	session := m.readPayload(ctx)
	value, ok := session.Flash[key]
	if !ok {
		return "", false, nil
	}
	delete(session.Flash, key)
	if err := m.writePayload(ctx, session); err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (m *Manager) Regenerate(ctx *core.Context) error {
	return m.writePayload(ctx, m.readPayload(ctx))
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

func (m *Manager) readPayload(ctx *core.Context) payload {
	if cached, ok := core.ContextValue[payload](ctx, sessionContextPrefix+m.Name); ok {
		normalize(&cached)
		return payload{
			Values: clone(cached.Values),
			Flash:  clone(cached.Flash),
		}
	}

	result := payload{
		Values: map[string]string{},
		Flash:  map[string]string{},
	}

	raw, ok := ctx.Cookie(m.Name)
	if !ok || raw == "" {
		ctx.Set(sessionContextPrefix+m.Name, result)
		return result
	}

	key := config.Get("APP_KEY")
	crypt, err := security.NewCrypt(key)
	if err == nil {
		if decrypted, decryptErr := crypt.DecryptString(raw); decryptErr == nil {
			if json.Unmarshal([]byte(decrypted), &result) == nil {
				normalize(&result)
				ctx.Set(sessionContextPrefix+m.Name, result)
				return payload{
					Values: clone(result.Values),
					Flash:  clone(result.Flash),
				}
			}
		}
	}

	// Backward-compatible reader for signed v4 development cookies.
	if legacy, valid := verify(raw, []byte(key)); valid {
		var values map[string]string
		if json.Unmarshal(legacy, &values) == nil && values != nil {
			result.Values = values
		}
	}
	ctx.Set(sessionContextPrefix+m.Name, result)
	return result
}

func (m *Manager) writePayload(ctx *core.Context, session payload) error {
	key := config.Get("APP_KEY")
	if len(key) < 32 {
		return errors.New("copytygo session: APP_KEY must be at least 32 characters")
	}
	normalize(&session)

	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}

	crypt, err := security.NewCrypt(key)
	if err != nil {
		return err
	}
	value, err := crypt.EncryptString(string(raw))
	if err != nil {
		return err
	}

	ctx.Set(sessionContextPrefix+m.Name, payload{
		Values: clone(session.Values),
		Flash:  clone(session.Flash),
	})
	ctx.SetCookie(&http.Cookie{
		Name:     m.Name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.Secure,
		SameSite: m.SameSite,
		MaxAge:   int(m.TTL.Seconds()),
	})
	return nil
}

func normalize(session *payload) {
	if session.Values == nil {
		session.Values = map[string]string{}
	}
	if session.Flash == nil {
		session.Flash = map[string]string{}
	}
}

func clone(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

// Legacy signing helpers are kept only for v4 development-cookie migration.
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
