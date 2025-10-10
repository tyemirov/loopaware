package httpapi

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	landingTemplateName    = "landing"
	landingHTMLContentType = "text/html; charset=utf-8"
	landingFooterElementID = "landing-footer"
	landingFooterInnerID   = "landing-footer-inner"
	landingFooterToggleID  = "landing-footer-toggle"
)

type landingTemplateData struct {
	FooterHTML     template.HTML
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
		BaseClass:         "landing-footer border-top mt-auto py-3",
		InnerClass:        "container d-flex flex-column flex-md-row align-items-center justify-content-center justify-content-md-end gap-3 text-center text-md-end",
		WrapperClass:      "dropup d-inline-flex align-items-center gap-2 text-body-secondary small",
		PrefixClass:       "text-body-secondary",
		PrefixText:        dashboardFooterBrandPrefix,
		ToggleButtonID:    landingFooterToggleID,
		ToggleButtonClass: "btn btn-link dropdown-toggle text-decoration-none px-0 fw-semibold",
		ToggleLabel:       dashboardFooterBrandName,
		MenuClass:         "dropdown-menu dropdown-menu-end shadow",
		MenuItemClass:     "dropdown-item",
	})
	if footerErr != nil {
		handlers.logger.Error("render_landing_footer", zap.Error(footerErr))
		footerHTML = template.HTML("")
	}

	data := landingTemplateData{
		FooterHTML:     footerHTML,
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
