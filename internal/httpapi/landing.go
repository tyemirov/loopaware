package httpapi

import (
	"bytes"
	"encoding/base64"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	landingTemplateName     = "landing"
	landingHTMLContentType  = "text/html; charset=utf-8"
	landingFooterElementID  = "landing-footer"
	landingFooterInnerID    = "landing-footer-inner"
	landingFooterToggleID   = "landing-footer-toggle"
	landingFooterBaseClass  = "landing-footer border-top mt-auto py-2"
	landingFooterInnerClass = "container py-2"
)

type landingTemplateData struct {
	SharedStyles   template.CSS
	FooterHTML     template.HTML
	HeaderHTML     template.HTML
	ThemeScript    template.JS
	FaviconDataURI template.URL
}

// LandingPageHandlers renders the public landing page.
type LandingPageHandlers struct {
	logger   *zap.Logger
	template *template.Template
}

// NewLandingPageHandlers constructs handlers that render the landing template.
func NewLandingPageHandlers(logger *zap.Logger) *LandingPageHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	compiledTemplate := template.Must(template.New(landingTemplateName).Parse(landingTemplateHTML))
	return &LandingPageHandlers{
		logger:   logger,
		template: compiledTemplate,
	}
}

// RenderLandingPage writes the landing page response.
func (handlers *LandingPageHandlers) RenderLandingPage(context *gin.Context) {
	footerHTML, footerErr := RenderFooterHTML(FooterConfig{
		ElementID:         landingFooterElementID,
		InnerElementID:    landingFooterInnerID,
		BaseClass:         landingFooterBaseClass,
		InnerClass:        landingFooterInnerClass,
		WrapperClass:      footerLayoutClass,
		BrandWrapperClass: footerBrandWrapperClass,
		MenuWrapperClass:  footerMenuWrapperClass,
		PrefixClass:       footerPrefixClass,
		PrefixText:        dashboardFooterBrandPrefix,
		ToggleButtonID:    landingFooterToggleID,
		ToggleButtonClass: footerToggleButtonClass,
		ToggleLabel:       dashboardFooterBrandName,
		MenuClass:         footerMenuClass,
		MenuItemClass:     footerMenuItemClass,
		PrivacyLinkClass:  footerPrivacyLinkClass,
	})
	if footerErr != nil {
		handlers.logger.Error("render_landing_footer", zap.Error(footerErr))
		footerHTML = template.HTML("")
	}

	headerHTML, headerErr := renderPublicHeader(landingLogoDataURI)
	if headerErr != nil {
		handlers.logger.Error("render_landing_header", zap.Error(headerErr))
		headerHTML = template.HTML("")
	}

	themeScript, themeErr := renderPublicThemeScript()
	if themeErr != nil {
		handlers.logger.Error("render_public_theme_script", zap.Error(themeErr))
		themeScript = template.JS("")
	}

	data := landingTemplateData{
		SharedStyles:   sharedPublicStyles(),
		FooterHTML:     footerHTML,
		HeaderHTML:     headerHTML,
		ThemeScript:    themeScript,
		FaviconDataURI: template.URL(dashboardFaviconDataURI),
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
