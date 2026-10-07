package mail

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	texttemplate "text/template"
)

func RenderHTML(source string, data any) (string, error) {
	tpl, err := htmltemplate.New("mail").Parse(source)
	if err != nil {
		return "", fmt.Errorf("copytygo mail: parse HTML template: %w", err)
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("copytygo mail: render HTML template: %w", err)
	}
	return out.String(), nil
}

func RenderText(source string, data any) (string, error) {
	tpl, err := texttemplate.New("mail").Parse(source)
	if err != nil {
		return "", fmt.Errorf("copytygo mail: parse text template: %w", err)
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("copytygo mail: render text template: %w", err)
	}
	return out.String(), nil
}

func TemplatedMessage(
	to []string,
	subject string,
	htmlSource string,
	textSource string,
	data any,
) (Message, error) {
	message := Message{To: to, Subject: subject}

	if htmlSource != "" {
		rendered, err := RenderHTML(htmlSource, data)
		if err != nil {
			return Message{}, err
		}
		message.HTML = rendered
	}

	if textSource != "" {
		rendered, err := RenderText(textSource, data)
		if err != nil {
			return Message{}, err
		}
		message.Text = rendered
	}

	return message, nil
}
