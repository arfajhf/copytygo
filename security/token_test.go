package security

import (
	"testing"
	"time"
)

const testTokenSecret = "copytygo-test-secret-0123456789abcdef"

func TestSignedTokenRoundTrip(t *testing.T) {
	token, err := SignToken(testTokenSecret, Claims{
		Subject: "42",
		Role: "admin",
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}

	claims, err := VerifyToken(testTokenSecret, token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "42" || claims.Role != "admin" || claims.IssuedAt == 0 {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestSignedTokenRejectsWeakSecret(t *testing.T) {
	if _, err := SignToken("short", Claims{Subject: "42"}); err == nil {
		t.Fatal("expected weak secret error")
	}
}

func TestSignedTokenRejectsExpiredToken(t *testing.T) {
	token, err := SignToken(testTokenSecret, Claims{
		Subject: "42",
		ExpiresAt: time.Now().Add(-time.Minute).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyToken(testTokenSecret, token); err == nil {
		t.Fatal("expected expired token error")
	}
}
