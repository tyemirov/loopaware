package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
)

// PublicHandlers serves unauthenticated public API endpoints.
type PublicHandlers struct {
	database                   *gorm.DB
	logger                     *zap.Logger
	rateWindow                 time.Duration
	maxRequestsPerKeyPerWindow int
	rateCountersByKey          map[string]publicRateCounter
	rateCountersMutex          sync.Mutex
	visitRateWindow            time.Duration
	visitMaxRequestsPerWindow  int
	visitRateCountersByKey     map[string]publicRateCounter
	visitRateCountersMutex     sync.Mutex
	feedbackBroadcaster        *FeedbackEventBroadcaster
	subscriptionEvents         *SubscriptionTestEventBroadcaster
	feedbackNotifier           FeedbackNotifier
	subscriptionNotifier       SubscriptionNotifier
	subscriptionNotifications  bool
	publicBaseURL              string
	subscriptionTokenSecret    string
	subscriptionTokenTTL       time.Duration
	confirmationEmailSender    EmailSender
}

type publicRateCounter struct {
	windowStartedAt time.Time
	count           int
}

const (
	demoWidgetSiteID   = "__loopaware_widget_demo__"
	demoWidgetSiteName = "LoopAware Widget Demo"

	errorValueInvalidEmail         = "invalid_email"
	errorValueInvalidContact       = "invalid_contact"
	errorValueUnknownSubscription  = "unknown_subscription"
	errorValueDuplicateSubscriber  = "duplicate_subscription"
	errorValueSaveSubscriberFailed = "save_failed"
	errorValueInvalidAudience      = "invalid_audience"
	errorValueInvalidSite          = "unknown_site"
	errorValueInvalidSentiment     = "invalid_sentiment"
	errorValueInvalidVisitorID     = "invalid_visitor"
	errorValueInvalidURL           = "invalid_url"
	errorValueInvalidMobileClient  = "invalid_mobile_client"
	errorValueInvalidContext       = "invalid_context"

	subscriptionIPMaxLength        = 64
	subscriptionUserAgentMaxLength = 400
	mobileFeedbackContextMaxDepth  = 4

	subscriptionEventTypeSubmission   = "subscription"
	subscriptionEventTypeNotification = "notification"
	subscriptionEventTypeConfirmation = "confirmation"
	subscriptionEventStatusSuccess    = "ok"
	subscriptionEventStatusError      = "error"
	subscriptionEventStatusSkipped    = "skipped"

	defaultSubscriptionConfirmationTokenTTL = 48 * time.Hour
	publicMaxRateCounterEntries             = 4096
	publicRateScopeFeedback                 = "feedback"
	publicRateScopeMobileFeedback           = "mobile-feedback"
	publicRateScopeSubscription             = "subscription"
	publicRateScopeVisit                    = "visit"
	publicVisitMaxRequestsPerWindow         = 120
	publicVisitRateWindow                   = 30 * time.Second
)

// NewPublicHandlers constructs a PublicHandlers instance with the provided dependencies.
func NewPublicHandlers(database *gorm.DB, logger *zap.Logger, feedbackBroadcaster *FeedbackEventBroadcaster, subscriptionEvents *SubscriptionTestEventBroadcaster, notifier FeedbackNotifier, subscriptionNotifier SubscriptionNotifier, subscriptionNotificationsEnabled bool, publicBaseURL string, subscriptionTokenSecret string, confirmationEmailSender EmailSender) *PublicHandlers {
	normalizedPublicBaseURL := strings.TrimSpace(publicBaseURL)
	normalizedTokenSecret := strings.TrimSpace(subscriptionTokenSecret)
	return &PublicHandlers{
		database:                   database,
		logger:                     logger,
		rateWindow:                 30 * time.Second,
		maxRequestsPerKeyPerWindow: 6,
		rateCountersByKey:          make(map[string]publicRateCounter),
		visitRateWindow:            publicVisitRateWindow,
		visitMaxRequestsPerWindow:  publicVisitMaxRequestsPerWindow,
		visitRateCountersByKey:     make(map[string]publicRateCounter),
		feedbackBroadcaster:        feedbackBroadcaster,
		subscriptionEvents:         subscriptionEvents,
		feedbackNotifier:           resolveFeedbackNotifier(notifier),
		subscriptionNotifier:       resolveSubscriptionNotifier(subscriptionNotifier),
		subscriptionNotifications:  subscriptionNotificationsEnabled,
		publicBaseURL:              normalizedPublicBaseURL,
		subscriptionTokenSecret:    normalizedTokenSecret,
		subscriptionTokenTTL:       defaultSubscriptionConfirmationTokenTTL,
		confirmationEmailSender:    confirmationEmailSender,
	}
}

type createFeedbackRequest struct {
	SiteID      string `json:"site_id"`
	ContactInfo string `json:"contact"`
	MessageBody string `json:"message"`
	Sentiment   string `json:"sentiment"`
	SourceURL   string `json:"source_url"`
}

type createMobileFeedbackRequest struct {
	SiteID         string                      `json:"site_id"`
	MobileClientID string                      `json:"mobile_client_id"`
	ContactInfo    string                      `json:"contact"`
	MessageBody    string                      `json:"message"`
	Sentiment      string                      `json:"sentiment"`
	Screen         mobileFeedbackScreenRequest `json:"screen"`
	App            mobileFeedbackAppRequest    `json:"app"`
	Context        json.RawMessage             `json:"context"`
}

type mobileFeedbackScreenRequest struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type mobileFeedbackAppRequest struct {
	Platform      string `json:"platform"`
	ApplicationID string `json:"application_id"`
	Version       string `json:"version"`
	Build         string `json:"build"`
	Environment   string `json:"environment"`
}

type createSubscriptionRequest struct {
	SiteID      string `json:"site_id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	SourceURL   string `json:"source_url"`
	AudienceKey string `json:"audience_key"`
}

type widgetConfigResponse struct {
	SiteID                   string `json:"site_id"`
	WidgetBubbleSide         string `json:"widget_bubble_side"`
	WidgetBubbleBottomOffset int    `json:"widget_bubble_bottom_offset"`
	WidgetAccentColor        string `json:"widget_accent_color"`
	WidgetShowMessageInput   bool   `json:"widget_show_message_input"`
	WidgetShowSentiment      bool   `json:"widget_show_sentiment_buttons"`
}

type subscriptionLinkResponse struct {
	Heading        string `json:"heading"`
	Message        string `json:"message"`
	OpenURL        string `json:"open_url"`
	OpenLabel      string `json:"open_label"`
	UnsubscribeURL string `json:"unsubscribe_url"`
}

type feedbackRecordInput struct {
	Contact        string
	Message        string
	Sentiment      string
	SourceKind     string
	MobileClientID string
	ScreenName     string
	ScreenPath     string
	AppPlatform    string
	AppIdentifier  string
	AppVersion     string
	AppBuild       string
	AppEnvironment string
	ContextJSON    string
	SourceURL      string
}

func buildSubscriptionLinkResponse(heading string, message string, site model.Site, subscriber model.Subscriber, confirmationToken string) subscriptionLinkResponse {
	openURL := subscriptionConfirmationOpenURL(site, subscriber)
	openLabel := "Open site"
	trimmedSiteName := strings.TrimSpace(site.Name)
	if trimmedSiteName != "" {
		openLabel = "Open " + trimmedSiteName
	}

	unsubscribeURLValue := ""
	if strings.TrimSpace(confirmationToken) != "" && subscriber.Status == model.SubscriberStatusConfirmed {
		query := url.Values{}
		query.Set("token", confirmationToken)
		unsubscribeURLValue = "/subscriptions/unsubscribe?" + query.Encode()
	}

	return subscriptionLinkResponse{
		Heading:        heading,
		Message:        message,
		OpenURL:        openURL,
		OpenLabel:      openLabel,
		UnsubscribeURL: unsubscribeURLValue,
	}
}

// CreateFeedback accepts feedback submissions from the public widget.
func (h *PublicHandlers) CreateFeedback(context *gin.Context) {
	var payload createFeedbackRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(400, gin.H{"error": "invalid_json"})
		return
	}

	payload.SiteID = strings.TrimSpace(payload.SiteID)
	payload.ContactInfo = strings.TrimSpace(payload.ContactInfo)
	payload.MessageBody = strings.TrimSpace(payload.MessageBody)
	payload.Sentiment = strings.TrimSpace(payload.Sentiment)
	payload.SourceURL = strings.TrimSpace(payload.SourceURL)

	if payload.SiteID == "" || payload.ContactInfo == "" {
		context.JSON(400, gin.H{"error": "missing_fields"})
		return
	}

	normalizedContact, contactErr := normalizeFeedbackContact(payload.ContactInfo)
	if contactErr != nil {
		context.JSON(400, gin.H{"error": errorValueInvalidContact})
		return
	}

	normalizedSentiment, sentimentErr := model.NormalizeFeedbackSentiment(payload.Sentiment)
	if sentimentErr != nil {
		context.JSON(400, gin.H{"error": errorValueInvalidSentiment})
		return
	}

	if payload.MessageBody == "" && normalizedSentiment == "" {
		context.JSON(400, gin.H{"error": "missing_fields"})
		return
	}

	sourceURL, sourceURLErr := model.NormalizeFeedbackSourceURL(payload.SourceURL)
	if sourceURLErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": errorValueInvalidURL})
		return
	}

	var site model.Site
	if err := h.database.First(&site, "id = ?", payload.SiteID).Error; err != nil {
		context.JSON(404, gin.H{"error": "unknown_site"})
		return
	}

	originHeader := strings.TrimSpace(context.GetHeader("Origin"))
	refererHeader := strings.TrimSpace(context.GetHeader("Referer"))
	allowedOrigins := mergedAllowedOrigins(site.AllowedOrigin, site.WidgetAllowedOrigins)
	if !isOriginAllowed(allowedOrigins, originHeader, refererHeader, "") {
		context.JSON(403, gin.H{"error": "origin_forbidden"})
		return
	}
	if sourceURL != "" && !isOriginAllowed(allowedOrigins, "", "", sourceURL) {
		context.JSON(http.StatusForbidden, gin.H{"error": "origin_forbidden"})
		return
	}
	if h.isRateLimited(publicRateKey(publicRateScopeFeedback, site.ID, context.ClientIP())) {
		context.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
		return
	}

	h.storeFeedback(context, site, feedbackRecordInput{
		Contact:    normalizedContact,
		Message:    payload.MessageBody,
		Sentiment:  normalizedSentiment,
		SourceKind: model.FeedbackSourceWebWidget,
		SourceURL:  sourceURL,
	})
}

// CreateMobileFeedback accepts feedback submissions from native mobile apps.
func (h *PublicHandlers) CreateMobileFeedback(context *gin.Context) {
	if hasBrowserOriginHeaders(context) {
		context.JSON(http.StatusForbidden, gin.H{"error": errorValueOriginForbidden})
		return
	}

	var payload createMobileFeedbackRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(400, gin.H{"error": "invalid_json"})
		return
	}

	payload.SiteID = strings.TrimSpace(payload.SiteID)
	payload.MobileClientID = strings.TrimSpace(payload.MobileClientID)
	payload.ContactInfo = strings.TrimSpace(payload.ContactInfo)
	payload.MessageBody = strings.TrimSpace(payload.MessageBody)
	payload.Sentiment = strings.TrimSpace(payload.Sentiment)
	payload.Screen.Name = strings.TrimSpace(payload.Screen.Name)
	payload.Screen.Path = strings.TrimSpace(payload.Screen.Path)
	payload.App.ApplicationID = strings.TrimSpace(payload.App.ApplicationID)
	payload.App.Version = strings.TrimSpace(payload.App.Version)
	payload.App.Build = strings.TrimSpace(payload.App.Build)
	payload.App.Environment = strings.TrimSpace(payload.App.Environment)

	if payload.SiteID == "" || payload.MobileClientID == "" || payload.ContactInfo == "" || payload.Screen.Name == "" || payload.App.ApplicationID == "" {
		context.JSON(400, gin.H{"error": "missing_fields"})
		return
	}

	normalizedContact, contactErr := normalizeFeedbackContact(payload.ContactInfo)
	if contactErr != nil {
		context.JSON(400, gin.H{"error": errorValueInvalidContact})
		return
	}

	normalizedSentiment, sentimentErr := model.NormalizeFeedbackSentiment(payload.Sentiment)
	if sentimentErr != nil {
		context.JSON(400, gin.H{"error": errorValueInvalidSentiment})
		return
	}

	if payload.MessageBody == "" && normalizedSentiment == "" {
		context.JSON(400, gin.H{"error": "missing_fields"})
		return
	}

	normalizedPlatform, platformErr := model.NormalizeMobilePlatform(payload.App.Platform)
	if platformErr != nil {
		context.JSON(400, gin.H{"error": errorValueInvalidMobileApp})
		return
	}

	contextJSON, contextErr := normalizeMobileFeedbackContext(payload.Context)
	if contextErr != nil {
		context.JSON(400, gin.H{"error": errorValueInvalidContext})
		return
	}

	var site model.Site
	if err := h.database.First(&site, "id = ?", payload.SiteID).Error; err != nil {
		context.JSON(404, gin.H{"error": "unknown_site"})
		return
	}

	var mobileApp model.SiteMobileApp
	if err := h.database.
		Where("site_id = ? AND client_id = ? AND enabled = ?", site.ID, payload.MobileClientID, true).
		First(&mobileApp).Error; err != nil {
		context.JSON(403, gin.H{"error": errorValueInvalidMobileClient})
		return
	}
	if mobileApp.Platform != normalizedPlatform || mobileApp.AppIdentifier != payload.App.ApplicationID {
		context.JSON(403, gin.H{"error": errorValueInvalidMobileClient})
		return
	}
	if h.isRateLimited(publicRateKey(publicRateScopeMobileFeedback, site.ID, context.ClientIP())) {
		context.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
		return
	}

	h.storeFeedback(context, site, feedbackRecordInput{
		Contact:        normalizedContact,
		Message:        payload.MessageBody,
		Sentiment:      normalizedSentiment,
		SourceKind:     model.FeedbackSourceMobileApp,
		MobileClientID: payload.MobileClientID,
		ScreenName:     payload.Screen.Name,
		ScreenPath:     payload.Screen.Path,
		AppPlatform:    normalizedPlatform,
		AppIdentifier:  payload.App.ApplicationID,
		AppVersion:     payload.App.Version,
		AppBuild:       payload.App.Build,
		AppEnvironment: payload.App.Environment,
		ContextJSON:    contextJSON,
	})
}

func (h *PublicHandlers) storeFeedback(context *gin.Context, site model.Site, input feedbackRecordInput) {
	sourceKind, sourceErr := model.NormalizeFeedbackSource(input.SourceKind)
	if sourceErr != nil {
		context.JSON(400, gin.H{"error": errorValueInvalidMobileApp})
		return
	}
	if contextErr := model.ValidateFeedbackContextJSON(input.ContextJSON); contextErr != nil {
		context.JSON(400, gin.H{"error": errorValueInvalidContext})
		return
	}

	feedback := model.Feedback{
		ID:             storage.NewID(),
		SiteID:         site.ID,
		Contact:        truncate(input.Contact, 320),
		Message:        truncate(input.Message, 4000),
		Sentiment:      truncate(input.Sentiment, 16),
		IP:             context.ClientIP(),
		UserAgent:      truncate(context.Request.UserAgent(), 400),
		Delivery:       model.FeedbackDeliveryNone,
		SourceKind:     sourceKind,
		SourceURL:      input.SourceURL,
		MobileClientID: truncate(input.MobileClientID, 80),
		ScreenName:     model.TruncateFeedbackScreenName(input.ScreenName),
		ScreenPath:     model.TruncateFeedbackScreenPath(input.ScreenPath),
		AppPlatform:    model.TruncateFeedbackAppPlatform(input.AppPlatform),
		AppIdentifier:  model.TruncateFeedbackAppIdentifier(input.AppIdentifier),
		AppVersion:     model.TruncateFeedbackAppVersion(input.AppVersion),
		AppBuild:       model.TruncateFeedbackAppBuild(input.AppBuild),
		AppEnvironment: model.TruncateFeedbackAppEnvironment(input.AppEnvironment),
		ContextJSON:    input.ContextJSON,
	}
	if err := h.database.Create(&feedback).Error; err != nil {
		h.logger.Warn("save_feedback", zap.Error(err))
		context.JSON(500, gin.H{"error": "save_failed"})
		return
	}

	h.applyFeedbackNotification(context.Request.Context(), site, &feedback)

	h.broadcastFeedbackCreated(context.Request.Context(), feedback)
	context.JSON(200, gin.H{"status": "ok"})
}

func hasBrowserOriginHeaders(context *gin.Context) bool {
	return strings.TrimSpace(context.GetHeader("Origin")) != "" || strings.TrimSpace(context.GetHeader("Referer")) != ""
}

func normalizeMobileFeedbackContext(rawContext json.RawMessage) (string, error) {
	trimmedContext := bytes.TrimSpace(rawContext)
	if len(trimmedContext) == 0 || bytes.Equal(trimmedContext, []byte("null")) {
		return "", nil
	}
	if len(trimmedContext) > 8192 {
		return "", model.ErrInvalidFeedbackContext
	}
	if trimmedContext[0] != '{' {
		return "", model.ErrInvalidFeedbackContext
	}

	var decodedContext any
	decoder := json.NewDecoder(bytes.NewReader(trimmedContext))
	decoder.UseNumber()
	if decodeErr := decoder.Decode(&decodedContext); decodeErr != nil {
		return "", decodeErr
	}
	if _, validObject := decodedContext.(map[string]any); !validObject {
		return "", model.ErrInvalidFeedbackContext
	}
	if !mobileFeedbackContextWithinDepth(decodedContext, 0) {
		return "", model.ErrInvalidFeedbackContext
	}

	normalizedContext, marshalErr := json.Marshal(decodedContext)
	if marshalErr != nil {
		return "", marshalErr
	}
	if len(normalizedContext) > 8192 {
		return "", model.ErrInvalidFeedbackContext
	}
	return string(normalizedContext), nil
}

func mobileFeedbackContextWithinDepth(value any, depth int) bool {
	if depth > mobileFeedbackContextMaxDepth {
		return false
	}
	switch typedValue := value.(type) {
	case map[string]any:
		for _, nestedValue := range typedValue {
			if !mobileFeedbackContextWithinDepth(nestedValue, depth+1) {
				return false
			}
		}
	case []any:
		for _, nestedValue := range typedValue {
			if !mobileFeedbackContextWithinDepth(nestedValue, depth+1) {
				return false
			}
		}
	}
	return true
}

func (h *PublicHandlers) applyFeedbackNotification(ctx context.Context, site model.Site, feedback *model.Feedback) {
	applyFeedbackNotification(ctx, h.database, h.logger, h.feedbackNotifier, site, feedback)
}

func (h *PublicHandlers) broadcastFeedbackCreated(ctx context.Context, feedback model.Feedback) {
	broadcastFeedbackEvent(h.database, h.logger, h.feedbackBroadcaster, ctx, feedback)
}

func (h *PublicHandlers) recordSubscriptionTestEvent(site model.Site, subscriber model.Subscriber, eventType, status, message string) {
	if h == nil || h.subscriptionEvents == nil {
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
	h.subscriptionEvents.Broadcast(event)
}

func (h *PublicHandlers) applySubscriptionNotification(ctx context.Context, site model.Site, subscriber model.Subscriber) {
	if subscriber.Status != model.SubscriberStatusConfirmed {
		h.recordSubscriptionTestEvent(site, subscriber, subscriptionEventTypeNotification, subscriptionEventStatusSkipped, "subscriber not confirmed")
		return
	}
	if !h.subscriptionNotifications {
		h.recordSubscriptionTestEvent(site, subscriber, subscriptionEventTypeNotification, subscriptionEventStatusSkipped, "subscription notifications disabled")
		return
	}
	if h.subscriptionNotifier == nil {
		h.recordSubscriptionTestEvent(site, subscriber, subscriptionEventTypeNotification, subscriptionEventStatusSkipped, "subscription notifier unavailable")
		return
	}
	if notifyErr := h.subscriptionNotifier.NotifySubscription(ctx, site, subscriber); notifyErr != nil {
		h.logger.Warn("subscription_notification_failed", zap.Error(notifyErr), zap.String("site_id", site.ID), zap.String("subscriber_id", subscriber.ID))
		h.recordSubscriptionTestEvent(site, subscriber, subscriptionEventTypeNotification, subscriptionEventStatusError, notifyErr.Error())
		return
	}
	h.recordSubscriptionTestEvent(site, subscriber, subscriptionEventTypeNotification, subscriptionEventStatusSuccess, "")
}

func (h *PublicHandlers) sendSubscriptionConfirmation(ctx context.Context, site model.Site, subscriber model.Subscriber) {
	if h == nil {
		return
	}
	sendSubscriptionConfirmationEmail(ctx, h.logger, h.recordSubscriptionTestEvent, h.confirmationEmailSender, h.publicBaseURL, h.subscriptionTokenSecret, h.subscriptionTokenTTL, site, subscriber)
}

func (h *PublicHandlers) isRateLimited(key string) bool {
	now := time.Now()

	h.rateCountersMutex.Lock()
	defer h.rateCountersMutex.Unlock()

	h.pruneRateCounters(now)
	rateCounter, exists := h.rateCountersByKey[key]
	if !exists && len(h.rateCountersByKey) >= publicMaxRateCounterEntries {
		return true
	}
	if !exists || now.Sub(rateCounter.windowStartedAt) >= h.rateWindow {
		rateCounter = publicRateCounter{windowStartedAt: now}
	}
	rateCounter.count++
	h.rateCountersByKey[key] = rateCounter
	return rateCounter.count > h.maxRequestsPerKeyPerWindow
}

func (h *PublicHandlers) pruneRateCounters(now time.Time) {
	for key, rateCounter := range h.rateCountersByKey {
		if now.Sub(rateCounter.windowStartedAt) >= h.rateWindow {
			delete(h.rateCountersByKey, key)
		}
	}
}

func (h *PublicHandlers) isVisitRateLimited(key string) bool {
	now := time.Now()

	h.visitRateCountersMutex.Lock()
	defer h.visitRateCountersMutex.Unlock()

	h.pruneVisitRateCounters(now)
	rateCounter, exists := h.visitRateCountersByKey[key]
	if !exists && len(h.visitRateCountersByKey) >= publicMaxRateCounterEntries {
		return true
	}
	if !exists || now.Sub(rateCounter.windowStartedAt) >= h.visitRateWindow {
		rateCounter = publicRateCounter{windowStartedAt: now}
	}
	rateCounter.count++
	h.visitRateCountersByKey[key] = rateCounter
	return rateCounter.count > h.visitMaxRequestsPerWindow
}

func (h *PublicHandlers) pruneVisitRateCounters(now time.Time) {
	for key, rateCounter := range h.visitRateCountersByKey {
		if now.Sub(rateCounter.windowStartedAt) >= h.visitRateWindow {
			delete(h.visitRateCountersByKey, key)
		}
	}
}

func publicRateKey(scope string, siteID string, clientIP string) string {
	return scope + "\x00" + siteID + "\x00" + clientIP
}

// WidgetConfig returns the widget configuration for a site.
func (h *PublicHandlers) WidgetConfig(context *gin.Context) {
	siteID := strings.TrimSpace(context.Query("site_id"))
	if siteID == "" {
		siteID = strings.TrimSpace(context.GetHeader("X-Site-Id"))
	}
	if siteID == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "missing_site_id"})
		return
	}

	var site model.Site
	if siteID == demoWidgetSiteID {
		site = model.Site{
			ID:                         demoWidgetSiteID,
			Name:                       demoWidgetSiteName,
			WidgetBubbleSide:           widgetBubbleSideLeft,
			WidgetBubbleBottomOffsetPx: defaultWidgetBubbleBottomOffset,
			WidgetAccentColor:          defaultWidgetAccentColor,
			WidgetShowMessageInput:     defaultWidgetShowMessageInput,
			WidgetShowSentimentButtons: defaultWidgetShowSentiment,
		}
	} else {
		if h.database == nil || h.database.First(&site, "id = ?", siteID).Error != nil {
			context.JSON(http.StatusNotFound, gin.H{"error": "unknown_site"})
			return
		}

		originHeader := strings.TrimSpace(context.GetHeader("Origin"))
		refererHeader := strings.TrimSpace(context.GetHeader("Referer"))
		allowedOrigins := mergedAllowedOrigins(site.AllowedOrigin, site.WidgetAllowedOrigins)
		if !isOriginAllowed(allowedOrigins, originHeader, refererHeader, "") {
			context.JSON(http.StatusForbidden, gin.H{"error": "origin_forbidden"})
			return
		}
	}

	ensureWidgetBubblePlacementDefaults(&site)
	ensureWidgetAccentColorDefault(&site)
	ensureWidgetFeedbackVisibilityDefaults(&site)
	context.JSON(http.StatusOK, widgetConfigResponse{
		SiteID:                   site.ID,
		WidgetBubbleSide:         site.WidgetBubbleSide,
		WidgetBubbleBottomOffset: site.WidgetBubbleBottomOffsetPx,
		WidgetAccentColor:        site.WidgetAccentColor,
		WidgetShowMessageInput:   site.WidgetShowMessageInput,
		WidgetShowSentiment:      site.WidgetShowSentimentButtons,
	})
}

// CreateSubscription registers a new subscriber.
func (h *PublicHandlers) CreateSubscription(context *gin.Context) {
	var payload createSubscriptionRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	payload.SiteID = strings.TrimSpace(payload.SiteID)
	payload.Email = strings.TrimSpace(payload.Email)
	payload.Name = strings.TrimSpace(payload.Name)
	payload.SourceURL = strings.TrimSpace(payload.SourceURL)
	payload.AudienceKey = strings.TrimSpace(payload.AudienceKey)

	if payload.SiteID == "" || payload.Email == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "missing_fields"})
		return
	}
	audienceKey, audienceKeyErr := model.NormalizeSubscriberAudienceKey(payload.AudienceKey)
	if audienceKeyErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": errorValueInvalidAudience})
		return
	}

	var site model.Site
	if err := h.database.First(&site, "id = ?", payload.SiteID).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": errorValueInvalidSite})
		return
	}

	originHeader := strings.TrimSpace(context.GetHeader("Origin"))
	refererHeader := strings.TrimSpace(context.GetHeader("Referer"))
	allowedOrigins := mergedAllowedOrigins(site.AllowedOrigin, site.SubscribeAllowedOrigins)
	if !isOriginAllowed(allowedOrigins, originHeader, refererHeader, "") {
		context.JSON(http.StatusForbidden, gin.H{"error": "origin_forbidden"})
		return
	}
	clientIP := context.ClientIP()
	if h.isRateLimited(publicRateKey(publicRateScopeSubscription, site.ID, clientIP)) {
		context.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
		return
	}

	existingSubscriber, err := findSubscriber(context.Request.Context(), h.database, site.ID, audienceKey, payload.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		context.JSON(http.StatusInternalServerError, gin.H{"error": errorValueSaveSubscriberFailed})
		return
	}
	if err == nil {
		if existingSubscriber.Status == model.SubscriberStatusUnsubscribed {
			now := time.Now().UTC()
			updateErr := h.database.Model(&existingSubscriber).Updates(map[string]any{
				"status":          model.SubscriberStatusPending,
				"unsubscribed_at": time.Time{},
				"confirmed_at":    time.Time{},
				"consent_at":      now,
				"name":            payload.Name,
				"source_url":      payload.SourceURL,
				"audience_key":    audienceKey,
				"ip":              truncate(clientIP, subscriptionIPMaxLength),
				"user_agent":      truncate(context.Request.UserAgent(), subscriptionUserAgentMaxLength),
			}).Error
			if updateErr != nil {
				context.JSON(http.StatusInternalServerError, gin.H{"error": errorValueSaveSubscriberFailed})
				return
			}
			existingSubscriber.Status = model.SubscriberStatusPending
			existingSubscriber.UnsubscribedAt = time.Time{}
			existingSubscriber.ConfirmedAt = time.Time{}
			existingSubscriber.ConsentAt = now
			existingSubscriber.Name = payload.Name
			existingSubscriber.SourceURL = payload.SourceURL
			existingSubscriber.AudienceKey = audienceKey
			existingSubscriber.IP = truncate(clientIP, subscriptionIPMaxLength)
			existingSubscriber.UserAgent = truncate(context.Request.UserAgent(), subscriptionUserAgentMaxLength)
			h.recordSubscriptionTestEvent(site, existingSubscriber, subscriptionEventTypeSubmission, subscriptionEventStatusSuccess, "")
			h.sendSubscriptionConfirmation(context.Request.Context(), site, existingSubscriber)
			context.JSON(http.StatusOK, gin.H{"status": "ok", "subscriber_id": existingSubscriber.ID})
			return
		}
		h.recordSubscriptionTestEvent(site, existingSubscriber, subscriptionEventTypeSubmission, subscriptionEventStatusError, errorValueDuplicateSubscriber)
		context.JSON(http.StatusConflict, gin.H{"error": errorValueDuplicateSubscriber})
		return
	}

	input := model.SubscriberInput{
		SiteID:      site.ID,
		Email:       payload.Email,
		Name:        payload.Name,
		SourceURL:   payload.SourceURL,
		AudienceKey: audienceKey,
		IP:          truncate(clientIP, subscriptionIPMaxLength),
		UserAgent:   truncate(context.Request.UserAgent(), subscriptionUserAgentMaxLength),
		Status:      model.SubscriberStatusPending,
		ConsentAt:   time.Now().UTC(),
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

	if err := h.database.Create(&subscriber).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": errorValueSaveSubscriberFailed})
		return
	}

	h.recordSubscriptionTestEvent(site, subscriber, subscriptionEventTypeSubmission, subscriptionEventStatusSuccess, "")
	h.sendSubscriptionConfirmation(context.Request.Context(), site, subscriber)
	context.JSON(http.StatusOK, gin.H{"status": "ok", "subscriber_id": subscriber.ID})
}

// ConfirmSubscriptionLinkJSON returns confirmation link metadata.
func (h *PublicHandlers) ConfirmSubscriptionLinkJSON(context *gin.Context) {
	token := strings.TrimSpace(context.Query("token"))
	if token == "" {
		context.JSON(http.StatusBadRequest, buildSubscriptionLinkResponse("Subscription confirmation", "Missing confirmation token.", model.Site{}, model.Subscriber{}, ""))
		return
	}
	if strings.TrimSpace(h.subscriptionTokenSecret) == "" {
		context.JSON(http.StatusInternalServerError, buildSubscriptionLinkResponse("Subscription confirmation", "Subscription confirmation is unavailable.", model.Site{}, model.Subscriber{}, ""))
		return
	}

	parsed, tokenErr := parseSubscriptionConfirmationToken(context.Request.Context(), h.subscriptionTokenSecret, token, time.Now().UTC())
	if tokenErr != nil {
		context.JSON(http.StatusBadRequest, buildSubscriptionLinkResponse("Subscription confirmation", "Invalid or expired token.", model.Site{}, model.Subscriber{}, ""))
		return
	}

	var subscriber model.Subscriber
	findErr := h.database.First(&subscriber, "id = ? AND site_id = ?", parsed.SubscriberID, parsed.SiteID).Error
	if findErr != nil {
		context.JSON(http.StatusBadRequest, buildSubscriptionLinkResponse("Subscription confirmation", "Invalid or expired token.", model.Site{}, model.Subscriber{}, ""))
		return
	}
	if strings.TrimSpace(strings.ToLower(subscriber.Email)) != strings.TrimSpace(strings.ToLower(parsed.Email)) {
		context.JSON(http.StatusBadRequest, buildSubscriptionLinkResponse("Subscription confirmation", "Invalid or expired token.", model.Site{}, model.Subscriber{}, ""))
		return
	}

	var site model.Site
	if siteErr := h.database.First(&site, "id = ?", subscriber.SiteID).Error; siteErr != nil {
		site = model.Site{}
	}

	if subscriber.Status == model.SubscriberStatusUnsubscribed {
		context.JSON(http.StatusConflict, buildSubscriptionLinkResponse("Subscription confirmation", "Subscription already unsubscribed.", site, subscriber, ""))
		return
	}
	if subscriber.Status == model.SubscriberStatusConfirmed {
		context.JSON(http.StatusOK, buildSubscriptionLinkResponse("Subscription confirmed", "Your subscription is already confirmed.", site, subscriber, token))
		return
	}

	now := time.Now().UTC()
	updateErr := h.database.Model(&subscriber).Updates(map[string]any{
		"status":          model.SubscriberStatusConfirmed,
		"confirmed_at":    now,
		"unsubscribed_at": time.Time{},
	}).Error
	if updateErr != nil {
		context.JSON(http.StatusInternalServerError, buildSubscriptionLinkResponse("Subscription confirmation", "Failed to confirm subscription.", site, subscriber, ""))
		return
	}

	subscriber.Status = model.SubscriberStatusConfirmed
	subscriber.ConfirmedAt = now
	subscriber.UnsubscribedAt = time.Time{}

	if strings.TrimSpace(site.ID) != "" {
		h.applySubscriptionNotification(context.Request.Context(), site, subscriber)
	}

	context.JSON(http.StatusOK, buildSubscriptionLinkResponse("Subscription confirmed", "Subscription confirmed.", site, subscriber, token))
}

// UnsubscribeSubscriptionLinkJSON returns unsubscribe link metadata.
func (h *PublicHandlers) UnsubscribeSubscriptionLinkJSON(context *gin.Context) {
	token := strings.TrimSpace(context.Query("token"))
	if token == "" {
		context.JSON(http.StatusBadRequest, buildSubscriptionLinkResponse("Unsubscribe", "Missing unsubscribe token.", model.Site{}, model.Subscriber{}, ""))
		return
	}
	if strings.TrimSpace(h.subscriptionTokenSecret) == "" {
		context.JSON(http.StatusInternalServerError, buildSubscriptionLinkResponse("Unsubscribe", "Subscription unsubscribe is unavailable.", model.Site{}, model.Subscriber{}, ""))
		return
	}

	parsed, tokenErr := parseSubscriptionConfirmationToken(context.Request.Context(), h.subscriptionTokenSecret, token, time.Now().UTC())
	if tokenErr != nil {
		context.JSON(http.StatusBadRequest, buildSubscriptionLinkResponse("Unsubscribe", "Invalid or expired token.", model.Site{}, model.Subscriber{}, ""))
		return
	}

	var subscriber model.Subscriber
	findErr := h.database.First(&subscriber, "id = ? AND site_id = ?", parsed.SubscriberID, parsed.SiteID).Error
	if findErr != nil {
		context.JSON(http.StatusBadRequest, buildSubscriptionLinkResponse("Unsubscribe", "Invalid or expired token.", model.Site{}, model.Subscriber{}, ""))
		return
	}
	if strings.TrimSpace(strings.ToLower(subscriber.Email)) != strings.TrimSpace(strings.ToLower(parsed.Email)) {
		context.JSON(http.StatusBadRequest, buildSubscriptionLinkResponse("Unsubscribe", "Invalid or expired token.", model.Site{}, model.Subscriber{}, ""))
		return
	}

	var site model.Site
	if siteErr := h.database.First(&site, "id = ?", subscriber.SiteID).Error; siteErr != nil {
		site = model.Site{}
	}

	if subscriber.Status == model.SubscriberStatusUnsubscribed {
		context.JSON(http.StatusOK, buildSubscriptionLinkResponse("Unsubscribed", "Subscription already unsubscribed.", site, subscriber, ""))
		return
	}

	now := time.Now().UTC()
	updateErr := h.database.Model(&subscriber).Updates(map[string]any{
		"status":          model.SubscriberStatusUnsubscribed,
		"unsubscribed_at": now,
	}).Error
	if updateErr != nil {
		context.JSON(http.StatusInternalServerError, buildSubscriptionLinkResponse("Unsubscribe", "Failed to unsubscribe.", site, subscriber, ""))
		return
	}

	subscriber.Status = model.SubscriberStatusUnsubscribed
	subscriber.UnsubscribedAt = now

	context.JSON(http.StatusOK, buildSubscriptionLinkResponse("Unsubscribed", "You have been unsubscribed.", site, subscriber, ""))
}

func subscriptionConfirmationOpenURL(site model.Site, subscriber model.Subscriber) string {
	trimmedSourceURL := strings.TrimSpace(subscriber.SourceURL)
	if trimmedSourceURL != "" {
		parsed, parseErr := url.Parse(trimmedSourceURL)
		if parseErr == nil && parsed != nil {
			scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
			if (scheme == "http" || scheme == "https") && strings.TrimSpace(parsed.Host) != "" {
				allowedOrigins := mergedAllowedOrigins(site.AllowedOrigin, site.SubscribeAllowedOrigins)
				if isOriginAllowed(allowedOrigins, "", "", trimmedSourceURL) {
					return trimmedSourceURL
				}
			}
		}
	}

	originCandidate := strings.TrimSpace(primaryAllowedOrigin(site.AllowedOrigin))
	if originCandidate == "" {
		return ""
	}
	parsedOrigin, originErr := url.Parse(originCandidate)
	if originErr != nil || parsedOrigin == nil {
		return ""
	}
	scheme := strings.ToLower(strings.TrimSpace(parsedOrigin.Scheme))
	if scheme != "http" && scheme != "https" {
		return ""
	}
	if strings.TrimSpace(parsedOrigin.Host) == "" {
		return ""
	}
	return originCandidate
}

func parseAllowedOrigins(rawAllowedOrigin string) []string {
	trimmedValue := strings.TrimSpace(rawAllowedOrigin)
	if trimmedValue == "" {
		return nil
	}
	normalizedSeparators := strings.NewReplacer(",", " ", ";", " ").Replace(trimmedValue)
	parts := strings.Fields(normalizedSeparators)
	if len(parts) == 0 {
		return nil
	}
	uniqueOrigins := make([]string, 0, len(parts))
	seenOrigins := make(map[string]struct{}, len(parts))
	for _, partValue := range parts {
		trimmedPart := strings.TrimSpace(partValue)
		if trimmedPart == "" {
			continue
		}
		lowerPart := strings.ToLower(trimmedPart)
		if _, alreadySeen := seenOrigins[lowerPart]; alreadySeen {
			continue
		}
		seenOrigins[lowerPart] = struct{}{}
		uniqueOrigins = append(uniqueOrigins, trimmedPart)
	}
	if len(uniqueOrigins) == 0 {
		return nil
	}
	return uniqueOrigins
}

func primaryAllowedOrigin(rawAllowedOrigin string) string {
	origins := parseAllowedOrigins(rawAllowedOrigin)
	if len(origins) == 0 {
		return ""
	}
	return origins[0]
}

func mergedAllowedOrigins(primaryAllowedOrigin string, extraAllowedOrigin string) string {
	primaryList := parseAllowedOrigins(primaryAllowedOrigin)
	extraList := parseAllowedOrigins(extraAllowedOrigin)
	if len(primaryList) == 0 && len(extraList) == 0 {
		return ""
	}
	if len(primaryList) == 0 {
		return strings.Join(extraList, " ")
	}
	if len(extraList) == 0 {
		return strings.Join(primaryList, " ")
	}
	merged := make([]string, 0, len(primaryList)+len(extraList))
	seenOrigins := make(map[string]struct{}, len(primaryList)+len(extraList))
	for _, origin := range append(primaryList, extraList...) {
		trimmedOrigin := strings.TrimSpace(origin)
		if trimmedOrigin == "" {
			continue
		}
		key := strings.ToLower(trimmedOrigin)
		if _, ok := seenOrigins[key]; ok {
			continue
		}
		seenOrigins[key] = struct{}{}
		merged = append(merged, trimmedOrigin)
	}
	if len(merged) == 0 {
		return ""
	}
	return strings.Join(merged, " ")
}

func isOriginAllowed(allowedOrigin string, originHeader string, refererHeader string, urlValue string) bool {
	allowedOrigins := parseAllowedOrigins(allowedOrigin)
	if len(allowedOrigins) == 0 {
		return true
	}

	normalizedOriginHeader := normalizeOriginValue(originHeader)
	normalizedRefererHeader := normalizeOriginValue(refererHeader)
	normalizedURLValue := normalizeOriginValue(urlValue)

	for _, configuredOrigin := range allowedOrigins {
		normalizedAllowedOrigin := normalizeOriginValue(configuredOrigin)
		if normalizedAllowedOrigin == "" {
			continue
		}
		if normalizedOriginHeader == normalizedAllowedOrigin {
			return true
		}
		if normalizedURLValue != "" && normalizedURLValue == normalizedAllowedOrigin {
			return true
		}
		if normalizedRefererHeader != "" && normalizedRefererHeader == normalizedAllowedOrigin {
			return true
		}
	}
	return false
}

func normalizeOriginValue(rawValue string) string {
	trimmedValue := strings.TrimSpace(rawValue)
	if trimmedValue == "" {
		return ""
	}
	parsedURL, parseErr := url.Parse(trimmedValue)
	if parseErr != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return ""
	}
	return strings.ToLower(parsedURL.Scheme) + "://" + strings.ToLower(parsedURL.Host)
}

func findSubscriber(ctx context.Context, database *gorm.DB, siteID string, audienceKey string, email string) (model.Subscriber, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	normalizedAudienceKey, audienceErr := model.NormalizeSubscriberAudienceKey(audienceKey)
	if audienceErr != nil {
		return model.Subscriber{}, audienceErr
	}
	var subscriber model.Subscriber
	err := database.WithContext(ctx).First(&subscriber, "site_id = ? AND audience_key = ? AND email = ?", siteID, normalizedAudienceKey, normalizedEmail).Error
	return subscriber, err
}

func truncate(input string, max int) string {
	if len(input) <= max {
		return input
	}
	return input[:max]
}
