package templater

import (
	"bytes"
	"fmt"
	"text/template"
)

// Render executes the provided template with the given data.
func Render(templateText string, data any) (string, error) {
	tmpl, err := template.New("prompt").Parse(templateText)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}

	return buf.String(), nil
}
