package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

type SiteSubscribeTestHandlers struct {
	database                  *gorm.DB
	logger                    *zap.Logger
	template                  *template.Template
	eventBroadcaster          *SubscriptionTestEventBroadcaster
	subscriptionNotifier      SubscriptionNotifier
	subscriptionNotifications bool
	publicBaseURL             string
	subscriptionTokenSecret   string
	subscriptionTokenTTL      time.Duration
	confirmationEmailSender   EmailSender
	authConfig                AuthClientConfig
}

func NewSiteSubscribeTestHandlers(database *gorm.DB, logger *zap.Logger, broadcaster *SubscriptionTestEventBroadcaster, subscriptionNotifier SubscriptionNotifier, subscriptionNotificationsEnabled bool, publicBaseURL string, subscriptionTokenSecret string, confirmationEmailSender EmailSender, authConfig AuthClientConfig) *SiteSubscribeTestHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	baseTemplate := template.Must(template.New("subscribe_test").Parse(dashboardHeaderTemplateHTML))
	compiled := template.Must(baseTemplate.Parse(subscribeTestTemplateHTML))
	normalizedPublicBaseURL := strings.TrimSpace(publicBaseURL)
	normalizedTokenSecret := strings.TrimSpace(subscriptionTokenSecret)
	return &SiteSubscribeTestHandlers{
		database:                  database,
		logger:                    logger,
		template:                  compiled,
		eventBroadcaster:          broadcaster,
		subscriptionNotifier:      resolveSubscriptionNotifier(subscriptionNotifier),
		subscriptionNotifications: subscriptionNotificationsEnabled,
		publicBaseURL:             normalizedPublicBaseURL,
		subscriptionTokenSecret:   normalizedTokenSecret,
		subscriptionTokenTTL:      defaultSubscriptionConfirmationTokenTTL,
		confirmationEmailSender:   confirmationEmailSender,
		authConfig:                authConfig,
	}
}

type subscribeTestTemplateData struct {
	PageTitle               string
	Header                  dashboardHeaderTemplateData
	LandingPath             string
	BootstrapIconsIntegrity template.HTMLAttr
	FaviconDataURI          template.URL
	TauthScriptURL          template.URL
	SiteName                string
	SiteID                  string
	AccentInputID           string
	CTAInputID              string
	TargetInputID           string
	NameFieldInputID        string
	InlineFormTitle         string
	InlineFormContainerID   string
	StatusLogElementID      string
	StatusTextElementID     string
	EventsEndpoint          template.URL
	DefaultAccent           string
	DefaultCTA              string
	SharedStyles            template.CSS
	AuthScript              template.JS
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

	_, ok := CurrentUserFromContext(context)
	if !ok {
		context.Redirect(http.StatusFound, LandingPagePath)
		return
	}

	headerData := dashboardHeaderTemplateData{
		PageTitle:               dashboardPageTitle,
		HeaderLogoDataURI:       landingLogoDataURI,
		HeaderLogoImageID:       dashboardHeaderLogoElementID,
		HeaderGoogleClientID:    handlers.authConfig.GoogleClientID,
		HeaderTauthBaseURL:      handlers.authConfig.TauthBaseURL,
		HeaderTauthTenantID:     handlers.authConfig.TauthTenantID,
		HeaderTauthLoginPath:    TauthLoginPath,
		HeaderTauthLogoutPath:   TauthLogoutPath,
		HeaderTauthNoncePath:    TauthNoncePath,
		HeaderSignInLabel:       publicSignInLabel,
		HeaderSignOutLabel:      publicSignOutLabel,
		SettingsButtonID:        settingsButtonElementID,
		SettingsButtonLabel:     navbarSettingsButtonLabel,
		SettingsModalID:         settingsModalElementID,
		SettingsModalTitleID:    settingsModalTitleElementID,
		SettingsModalTitle:      settingsModalTitle,
		SettingsModalIntro:      settingsModalIntroText,
		SettingsModalCloseLabel: settingsModalCloseButtonLabel,
		SettingsModalContentID:  settingsModalContentElementID,
	}

	footerHTML, footerErr := renderFooterHTMLForVariant(footerVariantDashboard)
	if footerErr != nil && handlers.logger != nil {
		handlers.logger.Warn("render_subscribe_test_footer", zap.Error(footerErr))
		footerHTML = template.HTML("")
	}
	authScript, authErr := renderPublicAuthScript()
	if authErr != nil && handlers.logger != nil {
		handlers.logger.Warn("render_subscribe_test_auth_script", zap.Error(authErr))
		authScript = template.JS("")
	}

	data := subscribeTestTemplateData{
		PageTitle:               "Subscribe Widget Test — LoopAware",
		Header:                  headerData,
		LandingPath:             LandingPagePath,
		BootstrapIconsIntegrity: template.HTMLAttr(dashboardBootstrapIconsIntegrityAttr),
		FaviconDataURI:          template.URL(dashboardFaviconDataURI),
		TauthScriptURL:          template.URL(handlers.authConfig.TauthScriptURL),
		SiteName:                "Loading...",
		SiteID:                  siteIdentifier,
		InlineFormTitle:         "Subscribe form preview",
		InlineFormContainerID:   "subscribe-test-inline-preview",
		AccentInputID:           "subscribe-test-accent",
		CTAInputID:              "subscribe-test-cta",
		TargetInputID:           "subscribe-test-target",
		NameFieldInputID:        "subscribe-test-name-field",
		StatusLogElementID:      "subscribe-test-log",
		StatusTextElementID:     "subscribe-test-status",
		EventsEndpoint:          template.URL("/api/sites/" + siteIdentifier + "/subscribe-test/events"),
		DefaultAccent:           subscribeTestAccentDefault,
		DefaultCTA:              subscribeTestCTADefault,
		SharedStyles:            sharedPublicStyles(),
		AuthScript:              authScript,
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
	flusher.Flush()

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

func (handlers *SiteSubscribeTestHandlers) CreateSubscription(context *gin.Context) {
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

	var payload createSubscriptionRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	payload.Email = strings.TrimSpace(payload.Email)
	payload.Name = strings.TrimSpace(payload.Name)
	payload.SourceURL = strings.TrimSpace(payload.SourceURL)

	if payload.Email == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "missing_fields"})
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

	clientIP := context.ClientIP()
	existingSubscriber, err := findSubscriber(context.Request.Context(), handlers.database, site.ID, payload.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		context.JSON(http.StatusInternalServerError, gin.H{"error": errorValueSaveSubscriberFailed})
		return
	}
	if err == nil {
		if existingSubscriber.Status == model.SubscriberStatusUnsubscribed {
			now := time.Now().UTC()
			updateErr := handlers.database.Model(&existingSubscriber).Updates(map[string]any{
				"status":          model.SubscriberStatusPending,
				"unsubscribed_at": time.Time{},
				"confirmed_at":    time.Time{},
				"consent_at":      now,
				"name":            payload.Name,
				"source_url":      payload.SourceURL,
				"ip":              truncate(clientIP, subscriptionIPMaxLength),
				"user_agent":      truncate(context.Request.UserAgent(), subscriptionUserAgentMaxLength),
			}).Error
			if updateErr != nil {
				context.JSON(http.StatusInternalServerError, gin.H{"error": errorValueSaveSubscriberFailed})
				return
			}
			handlers.recordSubscriptionTestEvent(site, existingSubscriber, subscriptionEventTypeSubmission, subscriptionEventStatusSuccess, "")
			handlers.sendSubscriptionConfirmation(context.Request.Context(), site, existingSubscriber)
			context.JSON(http.StatusOK, gin.H{"status": "ok", "subscriber_id": existingSubscriber.ID})
			return
		}
		handlers.recordSubscriptionTestEvent(site, existingSubscriber, subscriptionEventTypeSubmission, subscriptionEventStatusError, errorValueDuplicateSubscriber)
		context.JSON(http.StatusConflict, gin.H{"error": errorValueDuplicateSubscriber})
		return
	}

	input := model.SubscriberInput{
		SiteID:    site.ID,
		Email:     payload.Email,
		Name:      payload.Name,
		SourceURL: payload.SourceURL,
		IP:        truncate(clientIP, subscriptionIPMaxLength),
		UserAgent: truncate(context.Request.UserAgent(), subscriptionUserAgentMaxLength),
		Status:    model.SubscriberStatusPending,
		ConsentAt: time.Now().UTC(),
	}

	subscriber, subscriberErr := model.NewSubscriber(input)
	if subscriberErr != nil {
		if errors.Is(subscriberErr, model.ErrInvalidSubscriberEmail) {
			context.JSON(http.StatusBadRequest, gin.H{"error": errorValueInvalidEmail})
			return
		}
		context.JSON(http.StatusBadRequest, gin.H{"error": errorValueInvalidEmail})
		return
	}

	if err := handlers.database.Create(&subscriber).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": errorValueSaveSubscriberFailed})
		return
	}

	handlers.recordSubscriptionTestEvent(site, subscriber, subscriptionEventTypeSubmission, subscriptionEventStatusSuccess, "")
	handlers.sendSubscriptionConfirmation(context.Request.Context(), site, subscriber)
	context.JSON(http.StatusOK, gin.H{"status": "ok", "subscriber_id": subscriber.ID})
}

func (handlers *SiteSubscribeTestHandlers) sendSubscriptionConfirmation(ctx context.Context, site model.Site, subscriber model.Subscriber) {
	if handlers == nil {
		return
	}
	sendSubscriptionConfirmationEmail(ctx, handlers.logger, handlers.recordSubscriptionTestEvent, handlers.confirmationEmailSender, handlers.publicBaseURL, handlers.subscriptionTokenSecret, handlers.subscriptionTokenTTL, site, subscriber)
}

func (handlers *SiteSubscribeTestHandlers) recordSubscriptionTestEvent(site model.Site, subscriber model.Subscriber, eventType, status, message string) {
	if handlers == nil || handlers.eventBroadcaster == nil {
		return
	}
	normalizedSiteID := strings.TrimSpace(site.ID)
	normalizedSubscriberID := strings.TrimSpace(subscriber.ID)
	if normalizedSiteID == "" || normalizedSubscriberID == "" {
		return
	}
	normalizedStatus := strings.TrimSpace(status)
	if normalizedStatus == "" {
		normalizedStatus = subscriptionEventStatusSuccess
	}
	normalizedMessage := strings.TrimSpace(message)
	event := SubscriptionTestEvent{
		SiteID:       normalizedSiteID,
		SubscriberID: normalizedSubscriberID,
		Email:        strings.ToLower(strings.TrimSpace(subscriber.Email)),
		EventType:    strings.TrimSpace(eventType),
		Status:       normalizedStatus,
		Error:        normalizedMessage,
		Timestamp:    time.Now().UTC(),
	}
	if event.EventType == "" {
		event.EventType = subscriptionEventTypeSubmission
	}
	handlers.eventBroadcaster.Broadcast(event)
}
