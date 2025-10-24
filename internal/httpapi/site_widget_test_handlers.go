package httpapi

import (
	"bytes"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/temirov/GAuss/pkg/constants"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
)

type SiteWidgetTestHandlers struct {
	database            *gorm.DB
	logger              *zap.Logger
	widgetBaseURL       string
	template            *template.Template
	feedbackBroadcaster *FeedbackEventBroadcaster
}

func NewSiteWidgetTestHandlers(database *gorm.DB, logger *zap.Logger, widgetBaseURL string, feedbackBroadcaster *FeedbackEventBroadcaster) *SiteWidgetTestHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	baseTemplate := template.Must(template.New("widget_test").Parse(dashboardHeaderTemplateHTML))
	compiledTemplate := template.Must(baseTemplate.Parse(widgetTestTemplateHTML))
	return &SiteWidgetTestHandlers{
		database:            database,
		logger:              logger,
		widgetBaseURL:       normalizeWidgetBaseURL(widgetBaseURL),
		template:            compiledTemplate,
		feedbackBroadcaster: feedbackBroadcaster,
	}
}

type dashboardHeaderTemplateData struct {
	PageTitle                string
	HeaderLogoDataURI        template.URL
	HeaderLogoImageID        string
	SettingsButtonID         string
	SettingsButtonLabel      string
	SettingsAvatarImageID    string
	SettingsAvatarFallbackID string
	SettingsMenuID           string
	ThemeToggleLabel         string
	SettingsThemeToggleID    string
	LogoutButtonID           string
	LogoutLabel              string
}

type widgetTestTemplateData struct {
	PageTitle               string
	Header                  dashboardHeaderTemplateData
	LogoutPath              string
	LandingPath             string
	BootstrapIconsIntegrity template.HTMLAttr
	FaviconDataURI          template.URL
	SiteName                string
	SiteID                  string
	PlacementSide           string
	PlacementSideLabel      string
	PlacementOffset         int
	WidgetScriptURL         template.URL
	TestFeedbackEndpoint    template.URL
	WidgetUpdateEndpoint    template.URL
	SharedStyles            template.CSS
	FooterHTML              template.HTML
	FooterElementID         string
	FooterInnerElementID    string
	FooterBaseClass         string
	FooterThemeLightClass   string
	FooterThemeDarkClass    string
	ThemeStorageKey         string
	PublicThemeStorageKey   string
	LandingThemeStorageKey  string
	LegacyThemeStorageKey   string
}

func (handlers *SiteWidgetTestHandlers) RenderWidgetTestPage(context *gin.Context) {
	siteIdentifier := strings.TrimSpace(context.Param("id"))
	if siteIdentifier == "" {
		context.AbortWithStatus(http.StatusBadRequest)
		return
	}

	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.Redirect(http.StatusFound, constants.LoginPath)
		return
	}

	var site model.Site
	if handlers.database == nil || handlers.database.First(&site, "id = ?", siteIdentifier).Error != nil {
		context.AbortWithStatus(http.StatusNotFound)
		return
	}
	if !currentUser.canManageSite(site) {
		context.AbortWithStatus(http.StatusForbidden)
		return
	}

	ensureWidgetBubblePlacementDefaults(&site)
	widgetScriptURL := "/widget.js?site_id=" + url.QueryEscape(site.ID)
	if handlers.widgetBaseURL != "" {
		widgetScriptURL = handlers.widgetBaseURL + "/widget.js?site_id=" + url.QueryEscape(site.ID)
	}
	headerData := dashboardHeaderTemplateData{
		PageTitle:                dashboardPageTitle,
		HeaderLogoDataURI:        landingLogoDataURI,
		HeaderLogoImageID:        dashboardHeaderLogoElementID,
		SettingsButtonID:         settingsButtonElementID,
		SettingsButtonLabel:      navbarSettingsButtonLabel,
		SettingsAvatarImageID:    settingsAvatarImageElementID,
		SettingsAvatarFallbackID: settingsAvatarFallbackElementID,
		SettingsMenuID:           settingsMenuElementID,
		ThemeToggleLabel:         navbarThemeToggleLabel,
		SettingsThemeToggleID:    settingsThemeToggleElementID,
		LogoutButtonID:           logoutButtonElementID,
		LogoutLabel:              navbarLogoutLabel,
	}
	footerHTML, footerErr := renderFooterHTMLForVariant(footerVariantDashboard)
	if footerErr != nil {
		if handlers.logger != nil {
			handlers.logger.Warn("render_widget_test_footer", zap.Error(footerErr))
		}
		footerHTML = template.HTML("")
	}
	data := widgetTestTemplateData{
		PageTitle:               "Widget Test — " + site.Name,
		Header:                  headerData,
		LogoutPath:              constants.LogoutPath,
		LandingPath:             constants.LoginPath,
		BootstrapIconsIntegrity: template.HTMLAttr(dashboardBootstrapIconsIntegrityAttr),
		FaviconDataURI:          template.URL(dashboardFaviconDataURI),
		SiteName:                site.Name,
		SiteID:                  site.ID,
		PlacementSide:           strings.ToLower(strings.TrimSpace(site.WidgetBubbleSide)),
		PlacementSideLabel:      formatWidgetPlacementSide(site.WidgetBubbleSide),
		PlacementOffset:         site.WidgetBubbleBottomOffsetPx,
		WidgetScriptURL:         template.URL(widgetScriptURL),
		TestFeedbackEndpoint:    template.URL("/app/sites/" + site.ID + "/widget-test/feedback"),
		WidgetUpdateEndpoint:    template.URL("/api/sites/" + site.ID),
		SharedStyles:            sharedPublicStyles(),
		FooterHTML:              footerHTML,
		FooterElementID:         footerElementID,
		FooterInnerElementID:    footerInnerElementID,
		FooterBaseClass:         footerBaseClass,
		FooterThemeLightClass:   footerThemeLightClass,
		FooterThemeDarkClass:    footerThemeDarkClass,
		ThemeStorageKey:         themeStorageKey,
		PublicThemeStorageKey:   publicThemeStorageKey,
		LandingThemeStorageKey:  publicLandingThemeStorageKey,
		LegacyThemeStorageKey:   publicLegacyThemeStorageKey,
	}

	var buffer bytes.Buffer
	if err := handlers.template.Execute(&buffer, data); err != nil {
		if handlers.logger != nil {
			handlers.logger.Warn("render_widget_test_page", zap.Error(err))
		}
		context.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	context.Data(http.StatusOK, "text/html; charset=utf-8", buffer.Bytes())
}

type widgetTestFeedbackRequest struct {
	Contact string `json:"contact"`
	Message string `json:"message"`
}

func (handlers *SiteWidgetTestHandlers) SubmitWidgetTestFeedback(context *gin.Context) {
	siteIdentifier := strings.TrimSpace(context.Param("id"))
	if siteIdentifier == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingSite})
		return
	}

	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return
	}

	var site model.Site
	if handlers.database == nil || handlers.database.First(&site, "id = ?", siteIdentifier).Error != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSite})
		return
	}
	if !currentUser.canManageSite(site) {
		context.JSON(http.StatusForbidden, gin.H{jsonKeyError: errorValueNotAuthorized})
		return
	}

	var payload widgetTestFeedbackRequest
	if err := context.BindJSON(&payload); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}

	contact := strings.TrimSpace(payload.Contact)
	message := strings.TrimSpace(payload.Message)
	if contact == "" || message == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingFields})
		return
	}

	feedback := model.Feedback{
		ID:        storage.NewID(),
		SiteID:    site.ID,
		Contact:   truncate(contact, 320),
		Message:   truncate(message, 4000),
		IP:        context.ClientIP(),
		UserAgent: truncate(context.Request.UserAgent(), 400),
	}
	if err := handlers.database.Create(&feedback).Error; err != nil {
		if handlers.logger != nil {
			handlers.logger.Warn("create_widget_test_feedback", zap.Error(err))
		}
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}

	broadcastFeedbackEvent(handlers.database, handlers.logger, handlers.feedbackBroadcaster, context.Request.Context(), feedback)

	context.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func formatWidgetPlacementSide(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case widgetBubbleSideLeft:
		return "Left"
	case widgetBubbleSideRight:
		return "Right"
	default:
		return "Right"
	}
}
