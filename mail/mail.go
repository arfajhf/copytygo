package mail

import (
	"bytes"
	"fmt"
	"net/smtp"
	"strings"
)

type Message struct {
	From    string
	To      []string
	Subject string
	HTML    string
	Text    string
}

type Mailer interface {
	Send(Message) error
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type SMTP struct {
	config SMTPConfig
}

func NewSMTP(config SMTPConfig) *SMTP {
	if config.Port == 0 {
		config.Port = 587
	}
	return &SMTP{config: config}
}

func (s *SMTP) Send(message Message) error {
	if len(message.To) == 0 {
		return fmt.Errorf("copytygo mail: recipient is required")
	}

	from := strings.TrimSpace(message.From)
	if from == "" {
		from = strings.TrimSpace(s.config.From)
	}
	if from == "" {
		return fmt.Errorf("copytygo mail: sender is required")
	}
	if strings.TrimSpace(s.config.Host) == "" {
		return fmt.Errorf("copytygo mail: SMTP host is required")
	}

	var body bytes.Buffer
	body.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	body.WriteString("To: " + sanitizeHeader(strings.Join(message.To, ", ")) + "\r\n")
	body.WriteString("Subject: " + sanitizeHeader(message.Subject) + "\r\n")
	body.WriteString("MIME-Version: 1.0\r\n")

	content := message.Text
	contentType := "text/plain"
	if message.HTML != "" {
		content = message.HTML
		contentType = "text/html"
	}
	body.WriteString("Content-Type: " + contentType + "; charset=UTF-8\r\n\r\n")
	body.WriteString(content)

	address := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	var auth smtp.Auth
	if s.config.Username != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}

	if err := smtp.SendMail(address, auth, from, message.To, body.Bytes()); err != nil {
		return fmt.Errorf("copytygo mail: send SMTP message: %w", err)
	}
	return nil
}

func sanitizeHeader(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	return value
}

type Fake struct {
	Messages []Message
}

func (f *Fake) Send(message Message) error {
	f.Messages = append(f.Messages, message)
	return nil
}

func (f *Fake) Reset() {
	f.Messages = nil
}
