package httpapi

import (
	_ "embed"
	"text/template"
)

//go:embed assets/subscribe.js
var subscribeJavaScriptSource string

var subscribeJavaScriptTemplate = template.Must(template.New("subscribe.js").Parse(subscribeJavaScriptSource))
