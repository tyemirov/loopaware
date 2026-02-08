package httpapi

import (
	"bytes"
	"encoding/base64"
	"html/template"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	landingTemplateName             = "landing"
	landingHTMLContentType          = "text/html; charset=utf-8"
	landingFooterElementID          = "landing-footer"
	landingFooterInnerID            = "landing-footer-inner"
	landingFooterToggleID           = "landing-footer-toggle"
	landingFooterBaseClass          = "mpr-footer landing-footer border-top mt-auto py-2"
	landingDemoWidgetSiteID         = "a3222433-92ec-473a-9255-0797226c2273"
	landingWidgetScriptPath         = "/widget.js"
	widgetScriptQueryParamSiteID    = "site_id"
	widgetScriptQueryParamAPIOrigin = "api_origin"
)

type landingTemplateData struct {
	SharedStyles    template.CSS
	FooterHTML      template.HTML
	HeaderHTML      template.HTML
	ThemeScript     template.JS
	AuthScript      template.JS
	FaviconDataURI  template.URL
	TauthScriptURL  template.URL
	WidgetScriptURL template.URL
}

// PublicPageCurrentUserProvider exposes the authenticated user when available.
type PublicPageCurrentUserProvider interface {
	CurrentUser(*gin.Context) (*CurrentUser, bool)
}

// LandingPageHandlers renders the public landing page.
type LandingPageHandlers struct {
	logger              *zap.Logger
	template            *template.Template
	currentUserProvider PublicPageCurrentUserProvider
	authConfig          AuthClientConfig
	apiBaseURL          string
}

// NewLandingPageHandlers constructs handlers that render the landing template.
func NewLandingPageHandlers(logger *zap.Logger, currentUserProvider PublicPageCurrentUserProvider, authConfig AuthClientConfig, apiBaseURL string) *LandingPageHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	compiledTemplate := template.Must(template.New(landingTemplateName).Parse(landingTemplateHTML))
	return &LandingPageHandlers{
		logger:              logger,
		template:            compiledTemplate,
		currentUserProvider: currentUserProvider,
		authConfig:          authConfig,
		apiBaseURL:          normalizeBaseURL(apiBaseURL),
	}
}

// RenderLandingPage writes the landing page response.
func (handlers *LandingPageHandlers) RenderLandingPage(context *gin.Context) {
	footerHTML, footerErr := renderFooterHTMLForVariant(footerVariantLanding)
	if footerErr != nil {
		handlers.logger.Error("render_landing_footer", zap.Error(footerErr))
		footerHTML = template.HTML("")
	}

	isAuthenticated := false
	if handlers.currentUserProvider != nil {
		_, isAuthenticated = handlers.currentUserProvider.CurrentUser(context)
	}

	headerHTML, headerErr := renderPublicHeader(landingLogoDataURI, isAuthenticated, publicPageLanding, handlers.authConfig, true)
	if headerErr != nil {
		handlers.logger.Error("render_landing_header", zap.Error(headerErr))
		headerHTML = template.HTML("")
	}

	themeScript, themeErr := renderPublicThemeScript()
	if themeErr != nil {
		handlers.logger.Error("render_public_theme_script", zap.Error(themeErr))
		themeScript = template.JS("")
	}
	authScript, authErr := renderPublicAuthScript()
	if authErr != nil {
		handlers.logger.Error("render_public_auth_script", zap.Error(authErr))
		authScript = template.JS("")
	}

	widgetQuery := url.Values{}
	widgetQuery.Set(widgetScriptQueryParamSiteID, landingDemoWidgetSiteID)
	if handlers.apiBaseURL != "" {
		widgetQuery.Set(widgetScriptQueryParamAPIOrigin, handlers.apiBaseURL)
	}
	widgetScriptURL := landingWidgetScriptPath + "?" + widgetQuery.Encode()

	data := landingTemplateData{
		SharedStyles:    sharedPublicStyles(),
		FooterHTML:      footerHTML,
		HeaderHTML:      headerHTML,
		ThemeScript:     themeScript,
		AuthScript:      authScript,
		FaviconDataURI:  template.URL(dashboardFaviconDataURI),
		TauthScriptURL:  template.URL(handlers.authConfig.TauthScriptURL),
		WidgetScriptURL: template.URL(widgetScriptURL),
	}

	var buffer bytes.Buffer
	executeErr := handlers.template.Execute(&buffer, data)
	if executeErr != nil {
		handlers.logger.Error("render_landing_page", zap.Error(executeErr))
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "landing_render_failed"})
		return
	}
	context.Data(http.StatusOK, landingHTMLContentType, buffer.Bytes())
}

var landingLogoDataURI = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(landingLogoImage))
