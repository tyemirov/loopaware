package httpapi

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	PrivacyPagePath      = "/privacy"
	privacyTemplateName  = "privacy"
	privacyContentType   = "text/html; charset=utf-8"
	privacyRenderFailure = "privacy_render_failed"
)

type PrivacyPageHandlers struct {
	template *template.Template
}

func NewPrivacyPageHandlers() *PrivacyPageHandlers {
	compiledTemplate := template.Must(template.New(privacyTemplateName).Parse(privacyTemplateHTML))
	return &PrivacyPageHandlers{
		template: compiledTemplate,
	}
}

func (handlers *PrivacyPageHandlers) RenderPrivacyPage(context *gin.Context) {
	var buffer bytes.Buffer
	if err := handlers.template.Execute(&buffer, nil); err != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": privacyRenderFailure})
		return
	}
	context.Data(http.StatusOK, privacyContentType, buffer.Bytes())
}
