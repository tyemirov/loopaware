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
)

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
	var buffer bytes.Buffer
	executeErr := handlers.template.Execute(&buffer, nil)
	if executeErr != nil {
		handlers.logger.Error("render_landing_page", zap.Error(executeErr))
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "landing_render_failed"})
		return
	}
	context.Data(http.StatusOK, landingHTMLContentType, buffer.Bytes())
}
