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
	template *template.Template
}

type privacyTemplateData struct {
	FooterHTML template.HTML
}

func NewPrivacyPageHandlers() *PrivacyPageHandlers {
	compiledTemplate := template.Must(template.New(privacyTemplateName).Parse(privacyTemplateHTML))
	return &PrivacyPageHandlers{
		template: compiledTemplate,
	}
}

func (handlers *PrivacyPageHandlers) RenderPrivacyPage(context *gin.Context) {
	footerHTML, footerErr := RenderFooterHTML(FooterConfig{
		ElementID:         privacyFooterElementID,
		InnerElementID:    privacyFooterInnerID,
		BaseClass:         landingFooterBaseClass,
		InnerClass:        landingFooterInnerClass,
		WrapperClass:      footerLayoutClass,
		BrandWrapperClass: footerBrandWrapperClass,
		MenuWrapperClass:  footerMenuWrapperClass,
		PrefixClass:       footerPrefixClass,
		PrefixText:        dashboardFooterBrandPrefix,
		ToggleButtonID:    dashboardFooterToggleButtonID,
		ToggleButtonClass: footerToggleButtonClass,
		ToggleLabel:       dashboardFooterBrandName,
		MenuClass:         footerMenuClass,
		MenuItemClass:     footerMenuItemClass,
		PrivacyLinkClass:  footerPrivacyLinkClass,
	})
	if footerErr != nil {
		footerHTML = template.HTML("")
	}

	payload := privacyTemplateData{
		FooterHTML: footerHTML,
	}

	var buffer bytes.Buffer
	if err := handlers.template.Execute(&buffer, payload); err != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": privacyRenderFailure})
		return
	}
	context.Data(http.StatusOK, privacyContentType, buffer.Bytes())
}
