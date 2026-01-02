package httpapi

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	PrivacyPagePath        = "/privacy"
	privacyTemplateName    = "privacy"
	privacyContentType     = "text/html; charset=utf-8"
	privacyRenderFailure   = "privacy_render_failed"
	privacyFooterElementID = "privacy-footer"
	privacyFooterInnerID   = "privacy-footer-inner"
)

type PrivacyPageHandlers struct {
	template            *template.Template
	currentUserProvider PublicPageCurrentUserProvider
	authConfig          AuthClientConfig
}

type privacyTemplateData struct {
	SharedStyles   template.CSS
	PrivacyStyles  template.CSS
	FooterHTML     template.HTML
	HeaderHTML     template.HTML
	ThemeScript    template.JS
	TauthScriptURL template.URL
}

func NewPrivacyPageHandlers(currentUserProvider PublicPageCurrentUserProvider, authConfig AuthClientConfig) *PrivacyPageHandlers {
	compiledTemplate := template.Must(template.New(privacyTemplateName).Parse(privacyTemplateHTML))
	return &PrivacyPageHandlers{
		template:            compiledTemplate,
		currentUserProvider: currentUserProvider,
		authConfig:          authConfig,
	}
}

func (handlers *PrivacyPageHandlers) RenderPrivacyPage(context *gin.Context) {
	footerHTML, footerErr := renderFooterHTMLForVariant(footerVariantPrivacy)
	if footerErr != nil {
		footerHTML = template.HTML("")
	}

	isAuthenticated := false
	if handlers.currentUserProvider != nil {
		_, isAuthenticated = handlers.currentUserProvider.CurrentUser(context)
	}

	headerHTML, headerErr := renderPublicHeader(landingLogoDataURI, isAuthenticated, publicPagePrivacy, handlers.authConfig)
	if headerErr != nil {
		headerHTML = template.HTML("")
	}

	themeScript, themeErr := renderPublicThemeScript()
	if themeErr != nil {
		themeScript = template.JS("")
	}

	payload := privacyTemplateData{
		SharedStyles:   sharedPublicStyles(),
		PrivacyStyles:  privacyPageStyles(),
		FooterHTML:     footerHTML,
		HeaderHTML:     headerHTML,
		ThemeScript:    themeScript,
		TauthScriptURL: template.URL(handlers.authConfig.TauthScriptURL),
	}

	var buffer bytes.Buffer
	if err := handlers.template.Execute(&buffer, payload); err != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": privacyRenderFailure})
		return
	}
	context.Data(http.StatusOK, privacyContentType, buffer.Bytes())
}
