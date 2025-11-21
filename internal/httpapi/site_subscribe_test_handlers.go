package httpapi

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/temirov/GAuss/pkg/constants"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

type SiteSubscribeTestHandlers struct {
	database         *gorm.DB
	logger           *zap.Logger
	template         *template.Template
	eventBroadcaster *SubscriptionTestEventBroadcaster
}

func NewSiteSubscribeTestHandlers(database *gorm.DB, logger *zap.Logger, broadcaster *SubscriptionTestEventBroadcaster) *SiteSubscribeTestHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	baseTemplate := template.Must(template.New("subscribe_test").Parse(dashboardHeaderTemplateHTML))
	compiled := template.Must(baseTemplate.Parse(subscribeTestTemplateHTML))
	return &SiteSubscribeTestHandlers{
		database:         database,
		logger:           logger,
		template:         compiled,
		eventBroadcaster: broadcaster,
	}
}

type subscribeTestTemplateData struct {
	PageTitle               string
	Header                  dashboardHeaderTemplateData
	LogoutPath              string
	LandingPath             string
	BootstrapIconsIntegrity template.HTMLAttr
	FaviconDataURI          template.URL
	SiteName                string
	SiteID                  string
	PreviewBase             template.URL
	InlinePreviewTitle      string
	BubblePreviewTitle      string
	AccentInputID           string
	CTAInputID              string
	NameFieldInputID        string
	InlineFrameID           string
	BubbleFrameID           string
	StatusLogElementID      string
	StatusTextElementID     string
	ReloadInlineButtonID    string
	ReloadBubbleButtonID    string
	EventsEndpoint          template.URL
	DefaultAccent           string
	DefaultCTA              string
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
	DashboardPath           string
}

const (
	subscribeTestAccentDefault = "#0d6efd"
	subscribeTestCTADefault    = "Subscribe"
)

func (handlers *SiteSubscribeTestHandlers) RenderSubscribeTestPage(context *gin.Context) {
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

	headerData := dashboardHeaderTemplateData{
		PageTitle:                    dashboardPageTitle,
		HeaderLogoDataURI:            landingLogoDataURI,
		HeaderLogoImageID:            dashboardHeaderLogoElementID,
		SettingsButtonID:             settingsButtonElementID,
		SettingsButtonLabel:          navbarSettingsButtonLabel,
		SettingsAvatarImageID:        settingsAvatarImageElementID,
		SettingsAvatarFallbackID:     settingsAvatarFallbackElementID,
		SettingsMenuID:               settingsMenuElementID,
		SettingsMenuSettingsButtonID: settingsMenuSettingsButtonElementID,
		SettingsMenuSettingsLabel:    settingsMenuSettingsLabel,
		SettingsModalID:              settingsModalElementID,
		SettingsModalTitleID:         settingsModalTitleElementID,
		SettingsModalTitle:           settingsModalTitle,
		SettingsModalIntro:           settingsModalIntroText,
		SettingsModalCloseLabel:      settingsModalCloseButtonLabel,
		SettingsModalContentID:       settingsModalContentElementID,
		LogoutButtonID:               logoutButtonElementID,
		LogoutLabel:                  navbarLogoutLabel,
	}

	footerHTML, footerErr := renderFooterHTMLForVariant(footerVariantDashboard)
	if footerErr != nil && handlers.logger != nil {
		handlers.logger.Warn("render_subscribe_test_footer", zap.Error(footerErr))
		footerHTML = template.HTML("")
	}

	previewBase := "/subscribe-demo?site_id=" + url.QueryEscape(site.ID)
	eventsEndpoint := "/api/sites/" + site.ID + "/subscription-tests/events"

	data := subscribeTestTemplateData{
		PageTitle:               "Subscribe Widget Test — " + site.Name,
		Header:                  headerData,
		LogoutPath:              constants.LogoutPath,
		LandingPath:             constants.LoginPath,
		BootstrapIconsIntegrity: template.HTMLAttr(dashboardBootstrapIconsIntegrityAttr),
		FaviconDataURI:          template.URL(dashboardFaviconDataURI),
		SiteName:                site.Name,
		SiteID:                  site.ID,
		PreviewBase:             template.URL(previewBase),
		InlinePreviewTitle:      "Inline preview",
		BubblePreviewTitle:      "Bubble preview",
		AccentInputID:           "subscribe-test-accent",
		CTAInputID:              "subscribe-test-cta",
		NameFieldInputID:        "subscribe-test-name-field",
		InlineFrameID:           "subscribe-test-inline-frame",
		BubbleFrameID:           "subscribe-test-bubble-frame",
		StatusLogElementID:      "subscribe-test-log",
		StatusTextElementID:     "subscribe-test-status",
		ReloadInlineButtonID:    "subscribe-test-inline-reload",
		ReloadBubbleButtonID:    "subscribe-test-bubble-reload",
		EventsEndpoint:          template.URL(eventsEndpoint),
		DefaultAccent:           subscribeTestAccentDefault,
		DefaultCTA:              subscribeTestCTADefault,
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
		DashboardPath:           publicDashboardPath,
	}

	var buffer bytes.Buffer
	if err := handlers.template.Execute(&buffer, data); err != nil {
		if handlers.logger != nil {
			handlers.logger.Warn("render_subscribe_test_page", zap.Error(err))
		}
		context.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	context.Data(http.StatusOK, "text/html; charset=utf-8", buffer.Bytes())
}

func (handlers *SiteSubscribeTestHandlers) StreamSubscriptionTestEvents(context *gin.Context) {
	siteIdentifier := strings.TrimSpace(context.Param("id"))
	if siteIdentifier == "" {
		context.AbortWithStatus(http.StatusBadRequest)
		return
	}
	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
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

	if handlers.eventBroadcaster == nil {
		context.AbortWithStatus(http.StatusNoContent)
		return
	}

	subscription := handlers.eventBroadcaster.Subscribe()
	if subscription == nil {
		context.AbortWithStatus(http.StatusNoContent)
		return
	}
	defer subscription.Close()

	writer := context.Writer
	flusher, ok := writer.(http.Flusher)
	if !ok {
		context.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	context.Header("Content-Type", "text/event-stream")
	context.Header("Cache-Control", "no-cache")
	context.Header("Connection", "keep-alive")
	context.Status(http.StatusOK)

	for {
		select {
		case <-context.Request.Context().Done():
			return
		case event, open := <-subscription.Events():
			if !open {
				return
			}
			if event.SiteID != site.ID {
				continue
			}
			if event.Timestamp.IsZero() {
				event.Timestamp = time.Now().UTC()
			}
			payload, err := json.Marshal(event)
			if err != nil {
				if handlers.logger != nil {
					handlers.logger.Debug("subscribe_test_event_encode_failed", zap.Error(err))
				}
				continue
			}
			writer.Write([]byte("data: "))
			writer.Write(payload)
			writer.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}
