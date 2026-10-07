package session

import "testing"

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
