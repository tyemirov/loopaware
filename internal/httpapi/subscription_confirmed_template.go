package httpapi

import (
	"embed"
	"html/template"
)

//go:embed templates/subscription_confirmed.tmpl
var subscriptionConfirmedTemplateFS embed.FS

var subscriptionConfirmedTemplate = template.Must(template.ParseFS(subscriptionConfirmedTemplateFS, "templates/subscription_confirmed.tmpl"))

type subscriptionConfirmedTemplateData struct {
	PageTitle      string
	SharedStyles   template.CSS
	ThemeScript    template.JS
	FaviconDataURI template.URL
	HeaderHTML     template.HTML
	FooterHTML     template.HTML
	TauthScriptURL template.URL
	LandingPath    string
	Heading        string
	Message        string
	OpenURL        template.URL
	OpenLabel      string
	UnsubscribeURL template.URL
}
