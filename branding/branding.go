// Package branding provides the framework logo in standalone HTML and starters.
package branding

import (
	_ "embed"
	"encoding/base64"
	"html/template"
)

//go:embed logo.png
var Logo []byte

var logoURL = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(Logo))

// LogoURL returns the trusted embedded framework image; no asset server is needed.
func LogoURL() template.URL { return logoURL }
