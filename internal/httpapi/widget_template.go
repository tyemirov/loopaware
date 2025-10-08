package httpapi

import (
	_ "embed"
	"text/template"
)

//go:embed assets/widget.js
var widgetJavaScriptSource string

var widgetJavaScriptTemplate = template.Must(template.New("widget.js").Parse(widgetJavaScriptSource))
