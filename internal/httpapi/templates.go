package httpapi

import _ "embed"

//go:embed templates/dashboard.tmpl
var dashboardTemplateHTML string

//go:embed templates/landing.tmpl
var landingTemplateHTML string

//go:embed templates/privacy.tmpl
var privacyTemplateHTML string

//go:embed templates/example.tmpl
var exampleTemplateHTML string

//go:embed templates/logo.png
var landingLogoImage []byte
