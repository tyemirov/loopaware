package httpapi

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/temirov/GAuss/pkg/constants"
	"go.uber.org/zap"
)

const (
	landingTemplateName             = "landing"
	landingHTMLContentType          = "text/html; charset=utf-8"
	landingTemplateFilePattern      = "loopaware_landing_*.tmpl"
	landingFooterElementID          = "landing-footer"
	landingFooterInnerElementID     = "landing-footer-inner"
	landingFooterBaseClass          = "mt-auto py-5 bg-dark text-light border-top border-light-subtle"
	landingFooterInnerClass         = "container py-4 d-flex flex-column flex-md-row align-items-start align-items-md-center gap-3 justify-content-between text-center text-md-end"
	landingFooterDropupClass        = footerDropupWrapperClass + " justify-content-center justify-content-md-end"
	landingFooterPrefixClass        = "text-light-emphasis"
	landingFaviconPlaceholderToken  = "__FAVICON_DATA_URI__"
	landingErrorMessageIntro        = "We could not complete Google sign-in. Please try again."
	landingBrandName                = "LoopAware"
	landingNavigationApplicationURL = "/app"
	landingPrimaryCTAURL            = constants.GoogleAuthPath
	landingSecondaryCTAURL          = "/app"
)

// LandingPageHandlers renders the public landing page that also powers the GAuss login experience.
type LandingPageHandlers struct {
	logger         *zap.Logger
	template       *template.Template
	templateSource string
}

// NewLandingPageHandlers constructs LandingPageHandlers with a compiled landing page template.
func NewLandingPageHandlers(logger *zap.Logger) *LandingPageHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}

	templateSource, sourceErr := landingTemplateSource()
	if sourceErr != nil {
		panic(sourceErr)
	}

	compiledTemplate := template.Must(template.New(landingTemplateName).Parse(templateSource))

	return &LandingPageHandlers{
		logger:         logger,
		template:       compiledTemplate,
		templateSource: templateSource,
	}
}

// RenderLandingPage writes the rendered landing page HTML to the response writer.
func (handlers *LandingPageHandlers) RenderLandingPage(context *gin.Context) {
	errorParam := strings.TrimSpace(context.Query("error"))
	data := map[string]any{"error": errorParam}

	var buffer bytes.Buffer
	executeErr := handlers.template.Execute(&buffer, data)
	if executeErr != nil {
		handlers.logger.Error("render_landing_page", zap.Error(executeErr))
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "landing_render_failed"})
		return
	}

	context.Data(http.StatusOK, landingHTMLContentType, buffer.Bytes())
}

// WriteLandingTemplateFile persists the hydrated landing template to a temporary file for GAuss to consume.
func WriteLandingTemplateFile(directory string) (string, error) {
	templateSource, sourceErr := landingTemplateSource()
	if sourceErr != nil {
		return "", sourceErr
	}

	tempFile, createErr := os.CreateTemp(directory, landingTemplateFilePattern)
	if createErr != nil {
		return "", fmt.Errorf("create landing template file: %w", createErr)
	}
	defer func() {
		_ = tempFile.Close()
	}()

	if _, writeErr := tempFile.WriteString(templateSource); writeErr != nil {
		return "", fmt.Errorf("write landing template file: %w", writeErr)
	}

	if syncErr := tempFile.Sync(); syncErr != nil {
		return "", fmt.Errorf("sync landing template file: %w", syncErr)
	}

	return tempFile.Name(), nil
}

func landingTemplateSource() (string, error) {
	footerConfig := defaultFooterConfig(landingFooterElementID, landingFooterInnerElementID, landingFooterBaseClass)
	footerConfig.InnerContainerClass = landingFooterInnerClass
	footerConfig.DropupClass = landingFooterDropupClass
	footerConfig.PrefixClass = landingFooterPrefixClass
	footerHTML, footerErr := RenderFooterHTML(footerConfig)
	if footerErr != nil {
		return "", footerErr
	}

	templateWithFooter := strings.Replace(landingTemplateHTML, footerPlaceholderToken, footerHTML, 1)
	templateWithFavicon := strings.ReplaceAll(templateWithFooter, landingFaviconPlaceholderToken, dashboardFaviconDataURI)
	templateWithBrand := strings.ReplaceAll(templateWithFavicon, landingBrandPlaceholder(), landingBrandName)
	templateWithAppLink := strings.ReplaceAll(templateWithBrand, landingAppLinkPlaceholder(), landingNavigationApplicationURL)
	templateWithPrimaryCTA := strings.ReplaceAll(templateWithAppLink, landingPrimaryCTAPlaceholder(), landingPrimaryCTAURL)
	templateWithSecondaryCTA := strings.ReplaceAll(templateWithPrimaryCTA, landingSecondaryCTAPlaceholder(), landingSecondaryCTAURL)
	templateWithIntro := strings.ReplaceAll(templateWithSecondaryCTA, landingErrorIntroPlaceholder(), landingErrorMessageIntro)

	return templateWithIntro, nil
}

func landingBrandPlaceholder() string {
	return "__LANDING_BRAND_NAME__"
}

func landingAppLinkPlaceholder() string {
	return "__LANDING_APP_URL__"
}

func landingPrimaryCTAPlaceholder() string {
	return "__LANDING_PRIMARY_CTA__"
}

func landingSecondaryCTAPlaceholder() string {
	return "__LANDING_SECONDARY_CTA__"
}

func landingErrorIntroPlaceholder() string {
	return "__LANDING_ERROR_INTRO__"
}
