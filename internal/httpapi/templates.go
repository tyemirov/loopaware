package httpapi

import _ "embed"

//go:embed templates/dashboard.tmpl
var dashboardTemplateHTML string

//go:embed templates/landing.tmpl
var landingTemplateHTML string

//go:embed templates/privacy.tmpl
var privacyTemplateHTML string

//go:embed templates/widget_test.tmpl
var widgetTestTemplateHTML string

//go:embed templates/logo.png
var landingLogoImage []byte
