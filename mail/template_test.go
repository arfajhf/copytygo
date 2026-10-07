package mail

import (
	"strings"
	"testing"
)

func TestTemplatedMessage(t *testing.T) {
	message, err := TemplatedMessage(
		[]string{"user@example.com"},
		"Welcome",
		"<h1>Hello {{.Name}}</h1>",
		"Hello {{.Name}}",
		map[string]any{"Name":"Fajar"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(message.HTML, "Hello Fajar") || message.Text != "Hello Fajar" {
		t.Fatalf("unexpected rendered message %#v", message)
	}
}
