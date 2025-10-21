package httpapi

import (
	"bytes"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	exampleTemplateName             = "example"
	exampleHTMLContentType          = "text/html; charset=utf-8"
	examplePageTitle                = "LoopAware Widget Example"
	exampleHeadingText              = "Widget example"
	exampleDescriptionText          = "Enter a site identifier to embed the LoopAware widget on this page."
	exampleSiteIDLabelText          = "Site ID"
	exampleSiteIDPlaceholderText    = "00000000-0000-0000-0000-000000000000"
	exampleSubmitButtonLabel        = "Load widget"
	examplePreviewHeadingText       = "Embed snippet"
	examplePreviewDescriptionText   = "Copy this script tag into any page served from the allowed origin."
	exampleEmptyStateMessageText    = "Provide a site identifier to render the widget preview."
	exampleFormActionPath           = "/example"
	exampleSiteIDQueryParameter     = "site_id"
	exampleWidgetScriptRelativePath = "/widget.js"
	exampleDemoSiteID               = "__loopaware_widget_demo__"
	exampleDemoSiteName             = "LoopAware Widget Demo"
)

type exampleTemplateData struct {
	PageTitle            string
	Heading              string
	Description          string
	SharedStyles         template.CSS
	ThemeScript          template.JS
	SiteIDLabel          string
	SiteIDPlaceholder    string
	SiteIDValue          string
	SiteName             string
	SubmitLabel          string
	FormAction           string
	SiteIDQueryParameter string
	HasSiteID            bool
	WidgetScriptURL      template.URL
	PreviewHeading       string
	PreviewDescription   string
	EmptyStateMessage    string
	DemoMode             bool
}

// ExamplePageHandlers renders the widget example page.
type ExamplePageHandlers struct {
	logger   *zap.Logger
	template *template.Template
	database *gorm.DB
}

// NewExamplePageHandlers constructs handlers for the example page.
func NewExamplePageHandlers(logger *zap.Logger, database *gorm.DB) *ExamplePageHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	compiledTemplate := template.Must(template.New(exampleTemplateName).Parse(exampleTemplateHTML))
	return &ExamplePageHandlers{
		logger:   logger,
		template: compiledTemplate,
		database: database,
	}
}

// RenderExamplePage writes the example page response.
func (handlers *ExamplePageHandlers) RenderExamplePage(context *gin.Context) {
	themeScript, themeErr := renderPublicThemeScript()
	if themeErr != nil {
		handlers.logger.Error("render_example_theme_script", zap.Error(themeErr))
		themeScript = template.JS("")
	}

	siteIdentifier := strings.TrimSpace(context.Query(exampleSiteIDQueryParameter))
	siteName := ""
	demoMode := false

	if siteIdentifier == "" {
		siteIdentifier = exampleDemoSiteID
		siteName = exampleDemoSiteName
		demoMode = true
	} else if handlers.database != nil {
		if resolvedSite, lookupErr := handlers.lookupSiteByID(siteIdentifier); lookupErr == nil && resolvedSite != nil {
			siteName = resolvedSite.Name
		}
	}

	var widgetScriptURL template.URL
	var hasSiteIdentifier bool
	if siteIdentifier != "" {
		widgetScriptURL = template.URL(exampleWidgetScriptRelativePath + "?" + exampleSiteIDQueryParameter + "=" + url.QueryEscape(siteIdentifier))
		hasSiteIdentifier = true
	}

	data := exampleTemplateData{
		PageTitle:            examplePageTitle,
		Heading:              exampleHeadingText,
		Description:          exampleDescriptionText,
		SharedStyles:         sharedPublicStyles(),
		ThemeScript:          themeScript,
		SiteIDLabel:          exampleSiteIDLabelText,
		SiteIDPlaceholder:    exampleSiteIDPlaceholderText,
		SiteIDValue:          siteIdentifier,
		SiteName:             siteName,
		SubmitLabel:          exampleSubmitButtonLabel,
		FormAction:           exampleFormActionPath,
		SiteIDQueryParameter: exampleSiteIDQueryParameter,
		HasSiteID:            hasSiteIdentifier,
		WidgetScriptURL:      widgetScriptURL,
		PreviewHeading:       examplePreviewHeadingText,
		PreviewDescription:   examplePreviewDescriptionText,
		EmptyStateMessage:    exampleEmptyStateMessageText,
		DemoMode:             demoMode,
	}

	var buffer bytes.Buffer
	if executeErr := handlers.template.Execute(&buffer, data); executeErr != nil {
		handlers.logger.Error("render_example_page", zap.Error(executeErr))
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "example_render_failed"})
		return
	}

	context.Data(http.StatusOK, exampleHTMLContentType, buffer.Bytes())
}

func (handlers *ExamplePageHandlers) lookupSiteByID(identifier string) (*model.Site, error) {
	if handlers.database == nil {
		return nil, nil
	}
	var site model.Site
	if err := handlers.database.First(&site, "id = ?", identifier).Error; err != nil {
		return nil, err
	}
	return &site, nil
}
