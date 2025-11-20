package httpapi

import (
	"embed"
	"html/template"
)

//go:embed templates/subscribe_demo.tmpl
var subscribeDemoTemplateFS embed.FS

var subscribeDemoTemplate = template.Must(template.ParseFS(subscribeDemoTemplateFS, "templates/subscribe_demo.tmpl"))
