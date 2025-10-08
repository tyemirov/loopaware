package httpapi

import _ "embed"

//go:embed templates/dashboard.tmpl
var dashboardTemplateHTML string
