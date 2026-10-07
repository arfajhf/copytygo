package mail

import "testing"

func TestFakeMailer(t *testing.T) {
	fake := &Fake{}
	err := fake.Send(Message{To:[]string{"user@example.com"},Subject:"Welcome",Text:"Hello"})
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.Messages) != 1 || fake.Messages[0].Subject != "Welcome" {
		t.Fatal("expected captured message")
	}
}

func TestSanitizeHeader(t *testing.T) {
	if got := sanitizeHeader("Hello\r\nBcc: bad@example.com"); got != "HelloBcc: bad@example.com" {
		t.Fatalf("unexpected sanitized header %q", got)
	}
}
