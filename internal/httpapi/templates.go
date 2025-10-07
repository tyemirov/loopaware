package httpapi

import _ "embed"

//go:embed templates/dashboard.gotmpl
var dashboardTemplateHTML string
