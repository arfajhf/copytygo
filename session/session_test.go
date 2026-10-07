package session

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arfajhf/copytygo/v4/core"
	"github.com/arfajhf/copytygo/v4/security"
)

func TestSignedPayload(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	value := sign([]byte(`{"name":"copytygo"}`), key)
	data, ok := verify(value, key)
	if !ok || string(data) != `{"name":"copytygo"}` {
		t.Fatal("expected valid signed session")
	}
	if _, ok := verify(value+"x", key); ok {
		t.Fatal("tampered session must fail")
	}
}

func TestEncryptedSessionAndFlash(t *testing.T) {
	t.Setenv("APP_KEY", "12345678901234567890123456789012")
	t.Setenv("APP_ENV", "local")

	manager := New()
	app := core.New()

	app.Get("/write", func(ctx *core.Context) error {
		if err := manager.Put(ctx, "user_id", "42"); err != nil {
			return err
		}
		if err := manager.Flash(ctx, "status", "saved"); err != nil {
			return err
		}
		return ctx.Text("ok")
	})

	app.Get("/read", func(ctx *core.Context) error {
		flash, ok, err := manager.PullFlash(ctx, "status")
		if err != nil {
			return err
		}
		return ctx.JSON(core.Map{
			"user_id":  manager.Get(ctx, "user_id"),
			"flash":    flash,
			"flash_ok": ok,
		})
	})

	writeRec := httptest.NewRecorder()
	app.ServeHTTP(writeRec, httptest.NewRequest(http.MethodGet, "/write", nil))
	if writeRec.Code != http.StatusOK {
		t.Fatalf("write expected 200, got %d", writeRec.Code)
	}
	cookies := writeRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected encrypted session cookie")
	}
	// Random ciphertext can contain short strings such as "42" by chance.
	// Verify authenticated encryption rather than searching its encoded text.
	crypt, err := security.NewCrypt("12345678901234567890123456789012")
	if err != nil {
		t.Fatal(err)
	}
	encrypted := cookies[len(cookies)-1].Value
	decrypted, err := crypt.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("expected authenticated encrypted session: %v", err)
	}
	var state payload
	if err := json.Unmarshal([]byte(decrypted), &state); err != nil {
		t.Fatal(err)
	}
	if state.Values["user_id"] != "42" || state.Flash["status"] != "saved" {
		t.Fatal("encrypted cookie lost session or flash values")
	}
	wrongKey, err := security.NewCrypt("00000000000000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrongKey.DecryptString(encrypted); err == nil {
		t.Fatal("session cookie authenticated with the wrong key")
	}

	readReq := httptest.NewRequest(http.MethodGet, "/read", nil)
	readReq.AddCookie(cookies[len(cookies)-1])
	readRec := httptest.NewRecorder()
	app.ServeHTTP(readRec, readReq)

	body := readRec.Body.String()
	if !strings.Contains(body, `"user_id":"42"`) || !strings.Contains(body, `"flash":"saved"`) {
		t.Fatalf("unexpected session response %s", body)
	}
}
