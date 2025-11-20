package httpapi

import (
	_ "embed"
	"text/template"
)

//go:embed assets/pixel.js
var pixelJavaScriptSource string

var pixelJavaScriptTemplate = template.Must(template.New("pixel.js").Parse(pixelJavaScriptSource))
