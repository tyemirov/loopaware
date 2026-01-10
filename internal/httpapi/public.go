package httpapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
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

type PublicHandlers struct {
	database                  *gorm.DB
	logger                    *zap.Logger
	rateWindow                time.Duration
	maxRequestsPerIPPerWindow int
	rateCountersByIP          map[string]int
	rateCountersMutex         sync.Mutex
	feedbackBroadcaster       *FeedbackEventBroadcaster
	subscriptionEvents        *SubscriptionTestEventBroadcaster
	feedbackNotifier          FeedbackNotifier
	subscriptionNotifier      SubscriptionNotifier
	subscriptionNotifications bool
	publicBaseURL             string
	subscriptionTokenSecret   string
	subscriptionTokenTTL      time.Duration
	confirmationEmailSender   EmailSender
	authConfig                AuthClientConfig
}

const (
	demoWidgetSiteID   = "__loopaware_widget_demo__"
	demoWidgetSiteName = "LoopAware Widget Demo"

	errorValueInvalidEmail         = "invalid_email"
	errorValueUnknownSubscription  = "unknown_subscription"
	errorValueDuplicateSubscriber  = "duplicate_subscription"
	errorValueUnsubscribedAccount  = "unsubscribed"
	errorValueSaveSubscriberFailed = "save_failed"
	errorValueInvalidSite          = "unknown_site"
	errorValueInvalidVisitorID     = "invalid_visitor"
	errorValueInvalidURL           = "invalid_url"

	subscriptionIPMaxLength        = 64
	subscriptionUserAgentMaxLength = 400

	subscriptionEventTypeSubmission   = "subscription"
	subscriptionEventTypeNotification = "notification"
	subscriptionEventTypeConfirmation = "confirmation"
	subscriptionEventStatusSuccess    = "ok"
	subscriptionEventStatusError      = "error"
	subscriptionEventStatusSkipped    = "skipped"

	defaultSubscriptionConfirmationTokenTTL = 48 * time.Hour
)

func NewPublicHandlers(database *gorm.DB, logger *zap.Logger, feedbackBroadcaster *FeedbackEventBroadcaster, subscriptionEvents *SubscriptionTestEventBroadcaster, notifier FeedbackNotifier, subscriptionNotifier SubscriptionNotifier, subscriptionNotificationsEnabled bool, publicBaseURL string, subscriptionTokenSecret string, confirmationEmailSender EmailSender, authConfig AuthClientConfig) *PublicHandlers {
	normalizedPublicBaseURL := strings.TrimSpace(publicBaseURL)
	normalizedTokenSecret := strings.TrimSpace(subscriptionTokenSecret)
	return &PublicHandlers{
		database:                  database,
		logger:                    logger,
		rateWindow:                30 * time.Second,
		maxRequestsPerIPPerWindow: 6,
		rateCountersByIP:          make(map[string]int),
		feedbackBroadcaster:       feedbackBroadcaster,
		subscriptionEvents:        subscriptionEvents,
		feedbackNotifier:          resolveFeedbackNotifier(notifier),
		subscriptionNotifier:      resolveSubscriptionNotifier(subscriptionNotifier),
		subscriptionNotifications: subscriptionNotificationsEnabled,
		publicBaseURL:             normalizedPublicBaseURL,
		subscriptionTokenSecret:   normalizedTokenSecret,
		subscriptionTokenTTL:      defaultSubscriptionConfirmationTokenTTL,
		confirmationEmailSender:   confirmationEmailSender,
		authConfig:                authConfig,
	}
}

type createFeedbackRequest struct {
	SiteID      string `json:"site_id"`
	ContactInfo string `json:"contact"`
	MessageBody string `json:"message"`
}

type createSubscriptionRequest struct {
	SiteID    string `json:"site_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	SourceURL string `json:"source_url"`
}

type subscriptionMutationRequest struct {
	SiteID string `json:"site_id"`
	Email  string `json:"email"`
}

func (h *PublicHandlers) CreateFeedback(context *gin.Context) {
	clientIP := context.ClientIP()
	if h.isRateLimited(clientIP) {
		context.JSON(429, gin.H{"error": "rate_limited"})
		return
	}

	var payload createFeedbackRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(400, gin.H{"error": "invalid_json"})
		return
	}

	payload.SiteID = strings.TrimSpace(payload.SiteID)
	payload.ContactInfo = strings.TrimSpace(payload.ContactInfo)
	payload.MessageBody = strings.TrimSpace(payload.MessageBody)

	if payload.SiteID == "" || payload.ContactInfo == "" || payload.MessageBody == "" {
		context.JSON(400, gin.H{"error": "missing_fields"})
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

	feedback := model.Feedback{
		ID:        storage.NewID(),
		SiteID:    site.ID,
		Contact:   truncate(payload.ContactInfo, 320),
		Message:   truncate(payload.MessageBody, 4000),
		IP:        clientIP,
		UserAgent: truncate(context.Request.UserAgent(), 400),
		Delivery:  model.FeedbackDeliveryNone,
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

func (h *PublicHandlers) isRateLimited(ip string) bool {
	nowBucket := time.Now().Unix() / int64(h.rateWindow.Seconds())
	key := fmt.Sprintf("%s:%d", ip, nowBucket)

	h.rateCountersMutex.Lock()
	defer h.rateCountersMutex.Unlock()

	h.rateCountersByIP[key]++
	return h.rateCountersByIP[key] > h.maxRequestsPerIPPerWindow
}

func (h *PublicHandlers) WidgetJS(context *gin.Context) {
	siteID := strings.TrimSpace(context.Query("site_id"))
	if siteID == "" {
		siteID = strings.TrimSpace(context.GetHeader("X-Site-Id"))
	}
	if siteID == "" {
		context.String(400, "/* missing site_id */")
		return
	}

	var site model.Site
	if siteID == demoWidgetSiteID {
		site = model.Site{
			ID:                         demoWidgetSiteID,
			Name:                       demoWidgetSiteName,
			WidgetBubbleSide:           widgetBubbleSideLeft,
			WidgetBubbleBottomOffsetPx: defaultWidgetBubbleBottomOffset,
		}
	} else {
		if err := h.database.First(&site, "id = ?", siteID).Error; err != nil {
			context.String(404, "/* unknown site */")
			return
		}
	}
	ensureWidgetBubblePlacementDefaults(&site)

	script, tplErr := renderWidgetTemplate(site)
	if tplErr != nil {
		context.String(500, "/* render error */")
		return
	}

	context.Data(200, "application/javascript; charset=utf-8", []byte(script))
}

func (h *PublicHandlers) SubscribeJS(context *gin.Context) {
	siteID := strings.TrimSpace(context.Query("site_id"))
	if siteID == "" {
		siteID = strings.TrimSpace(context.GetHeader("X-Site-Id"))
	}
	if siteID == "" {
		context.String(http.StatusBadRequest, "/* missing site_id */")
		return
	}

	var site model.Site
	if err := h.database.First(&site, "id = ?", siteID).Error; err != nil {
		context.String(http.StatusNotFound, "/* unknown site */")
		return
	}

	script, tplErr := renderSubscribeTemplate(site)
	if tplErr != nil {
		context.String(http.StatusInternalServerError, "/* render error */")
		return
	}

	context.Data(http.StatusOK, "application/javascript; charset=utf-8", []byte(script))
}

func (h *PublicHandlers) SubscribeDemo(context *gin.Context) {
	siteID := strings.TrimSpace(context.Query("site_id"))
	if siteID == "" {
		context.String(http.StatusBadRequest, "missing site_id")
		return
	}

	var site model.Site
	if err := h.database.First(&site, "id = ?", siteID).Error; err != nil {
		context.String(http.StatusNotFound, "unknown site")
		return
	}

	extraParams := url.Values{}
	for _, key := range []string{"mode", "accent", "cta", "success", "error", "name_field"} {
		value := strings.TrimSpace(context.Query(key))
		if value != "" {
			extraParams.Set(key, value)
		}
	}

	scriptURL := "/subscribe.js?site_id=" + url.QueryEscape(site.ID)
	if encoded := extraParams.Encode(); encoded != "" {
		scriptURL += "&" + encoded
	}

	var buffer bytes.Buffer
	if err := subscribeDemoTemplate.Execute(&buffer, map[string]any{
		"SiteID":    site.ID,
		"ScriptURL": scriptURL,
	}); err != nil {
		context.String(http.StatusInternalServerError, "render error")
		return
	}

	context.Data(http.StatusOK, "text/html; charset=utf-8", buffer.Bytes())
}

func (h *PublicHandlers) CreateSubscription(context *gin.Context) {
	clientIP := context.ClientIP()
	if h.isRateLimited(clientIP) {
		context.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
		return
	}

	var payload createSubscriptionRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	payload.SiteID = strings.TrimSpace(payload.SiteID)
	payload.Email = strings.TrimSpace(payload.Email)
	payload.Name = strings.TrimSpace(payload.Name)
	payload.SourceURL = strings.TrimSpace(payload.SourceURL)

	if payload.SiteID == "" || payload.Email == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "missing_fields"})
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

	existingSubscriber, err := findSubscriber(context.Request.Context(), h.database, site.ID, payload.Email)
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

	if err := h.database.Create(&subscriber).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": errorValueSaveSubscriberFailed})
		return
	}

	h.recordSubscriptionTestEvent(site, subscriber, subscriptionEventTypeSubmission, subscriptionEventStatusSuccess, "")
	h.sendSubscriptionConfirmation(context.Request.Context(), site, subscriber)
	context.JSON(http.StatusOK, gin.H{"status": "ok", "subscriber_id": subscriber.ID})
}

func (h *PublicHandlers) ConfirmSubscription(context *gin.Context) {
	h.updateSubscriptionStatus(context, model.SubscriberStatusConfirmed)
}

func (h *PublicHandlers) Unsubscribe(context *gin.Context) {
	h.updateSubscriptionStatus(context, model.SubscriberStatusUnsubscribed)
}

func (h *PublicHandlers) ConfirmSubscriptionLink(context *gin.Context) {
	token := strings.TrimSpace(context.Query("token"))
	if token == "" {
		h.renderSubscriptionConfirmationPage(context, http.StatusBadRequest, "Subscription confirmation", "Missing confirmation token.", model.Site{}, model.Subscriber{}, "")
		return
	}
	if strings.TrimSpace(h.subscriptionTokenSecret) == "" {
		h.renderSubscriptionConfirmationPage(context, http.StatusInternalServerError, "Subscription confirmation", "Subscription confirmation is unavailable.", model.Site{}, model.Subscriber{}, "")
		return
	}

	parsed, tokenErr := parseSubscriptionConfirmationToken(context.Request.Context(), h.subscriptionTokenSecret, token, time.Now().UTC())
	if tokenErr != nil {
		h.renderSubscriptionConfirmationPage(context, http.StatusBadRequest, "Subscription confirmation", "Invalid or expired token.", model.Site{}, model.Subscriber{}, "")
		return
	}

	var subscriber model.Subscriber
	findErr := h.database.First(&subscriber, "id = ? AND site_id = ?", parsed.SubscriberID, parsed.SiteID).Error
	if findErr != nil {
		h.renderSubscriptionConfirmationPage(context, http.StatusBadRequest, "Subscription confirmation", "Invalid or expired token.", model.Site{}, model.Subscriber{}, "")
		return
	}
	if strings.TrimSpace(strings.ToLower(subscriber.Email)) != strings.TrimSpace(strings.ToLower(parsed.Email)) {
		h.renderSubscriptionConfirmationPage(context, http.StatusBadRequest, "Subscription confirmation", "Invalid or expired token.", model.Site{}, model.Subscriber{}, "")
		return
	}

	var site model.Site
	if siteErr := h.database.First(&site, "id = ?", subscriber.SiteID).Error; siteErr != nil {
		site = model.Site{}
	}

	if subscriber.Status == model.SubscriberStatusUnsubscribed {
		h.renderSubscriptionConfirmationPage(context, http.StatusConflict, "Subscription confirmation", "Subscription already unsubscribed.", site, subscriber, "")
		return
	}
	if subscriber.Status == model.SubscriberStatusConfirmed {
		h.renderSubscriptionConfirmationPage(context, http.StatusOK, "Subscription confirmed", "Your subscription is already confirmed.", site, subscriber, token)
		return
	}

	now := time.Now().UTC()
	updateErr := h.database.Model(&subscriber).Updates(map[string]any{
		"status":          model.SubscriberStatusConfirmed,
		"confirmed_at":    now,
		"unsubscribed_at": time.Time{},
	}).Error
	if updateErr != nil {
		h.renderSubscriptionConfirmationPage(context, http.StatusInternalServerError, "Subscription confirmation", "Failed to confirm subscription.", site, subscriber, "")
		return
	}

	subscriber.Status = model.SubscriberStatusConfirmed
	subscriber.ConfirmedAt = now
	subscriber.UnsubscribedAt = time.Time{}

	if strings.TrimSpace(site.ID) != "" {
		h.applySubscriptionNotification(context.Request.Context(), site, subscriber)
	}

	h.renderSubscriptionConfirmationPage(context, http.StatusOK, "Subscription confirmed", "Subscription confirmed.", site, subscriber, token)
}

func (h *PublicHandlers) UnsubscribeSubscriptionLink(context *gin.Context) {
	token := strings.TrimSpace(context.Query("token"))
	if token == "" {
		h.renderSubscriptionConfirmationPage(context, http.StatusBadRequest, "Unsubscribe", "Missing unsubscribe token.", model.Site{}, model.Subscriber{}, "")
		return
	}
	if strings.TrimSpace(h.subscriptionTokenSecret) == "" {
		h.renderSubscriptionConfirmationPage(context, http.StatusInternalServerError, "Unsubscribe", "Subscription unsubscribe is unavailable.", model.Site{}, model.Subscriber{}, "")
		return
	}

	parsed, tokenErr := parseSubscriptionConfirmationToken(context.Request.Context(), h.subscriptionTokenSecret, token, time.Now().UTC())
	if tokenErr != nil {
		h.renderSubscriptionConfirmationPage(context, http.StatusBadRequest, "Unsubscribe", "Invalid or expired token.", model.Site{}, model.Subscriber{}, "")
		return
	}

	var subscriber model.Subscriber
	findErr := h.database.First(&subscriber, "id = ? AND site_id = ?", parsed.SubscriberID, parsed.SiteID).Error
	if findErr != nil {
		h.renderSubscriptionConfirmationPage(context, http.StatusBadRequest, "Unsubscribe", "Invalid or expired token.", model.Site{}, model.Subscriber{}, "")
		return
	}
	if strings.TrimSpace(strings.ToLower(subscriber.Email)) != strings.TrimSpace(strings.ToLower(parsed.Email)) {
		h.renderSubscriptionConfirmationPage(context, http.StatusBadRequest, "Unsubscribe", "Invalid or expired token.", model.Site{}, model.Subscriber{}, "")
		return
	}

	var site model.Site
	if siteErr := h.database.First(&site, "id = ?", subscriber.SiteID).Error; siteErr != nil {
		site = model.Site{}
	}

	if subscriber.Status == model.SubscriberStatusUnsubscribed {
		h.renderSubscriptionConfirmationPage(context, http.StatusOK, "Unsubscribed", "Subscription already unsubscribed.", site, subscriber, "")
		return
	}

	now := time.Now().UTC()
	updateErr := h.database.Model(&subscriber).Updates(map[string]any{
		"status":          model.SubscriberStatusUnsubscribed,
		"unsubscribed_at": now,
	}).Error
	if updateErr != nil {
		h.renderSubscriptionConfirmationPage(context, http.StatusInternalServerError, "Unsubscribe", "Failed to unsubscribe.", site, subscriber, "")
		return
	}

	subscriber.Status = model.SubscriberStatusUnsubscribed
	subscriber.UnsubscribedAt = now

	h.renderSubscriptionConfirmationPage(context, http.StatusOK, "Unsubscribed", "You have been unsubscribed.", site, subscriber, "")
}

func (h *PublicHandlers) renderSubscriptionConfirmationPage(context *gin.Context, statusCode int, heading string, message string, site model.Site, subscriber model.Subscriber, confirmationToken string) {
	footerHTML, footerErr := renderFooterHTMLForVariant(footerVariantLanding)
	if footerErr != nil {
		footerHTML = ""
	}

	headerHTML, headerErr := renderPublicHeader(landingLogoDataURI, false, publicPageLanding, h.authConfig, false)
	if headerErr != nil {
		headerHTML = ""
	}

	themeScript, themeErr := renderPublicThemeScript()
	if themeErr != nil {
		themeScript = ""
	}
	authScript, authErr := renderPublicAuthScript()
	if authErr != nil {
		authScript = ""
	}

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

	payload := subscriptionConfirmedTemplateData{
		PageTitle:      heading + " — LoopAware",
		SharedStyles:   sharedPublicStyles(),
		ThemeScript:    themeScript,
		AuthScript:     authScript,
		FaviconDataURI: template.URL(dashboardFaviconDataURI),
		HeaderHTML:     headerHTML,
		FooterHTML:     footerHTML,
		TauthScriptURL: template.URL(h.authConfig.TauthScriptURL),
		LandingPath:    LandingPagePath,
		Heading:        heading,
		Message:        message,
		OpenURL:        "",
		OpenLabel:      openLabel,
		UnsubscribeURL: "",
	}
	if openURL != "" {
		payload.OpenURL = template.URL(openURL)
	}
	if unsubscribeURLValue != "" {
		payload.UnsubscribeURL = template.URL(unsubscribeURLValue)
	}

	var buffer bytes.Buffer
	if err := subscriptionConfirmedTemplate.Execute(&buffer, payload); err != nil {
		if h.logger != nil {
			h.logger.Warn("render_subscription_confirmed_page", zap.Error(err))
		}
		context.Data(statusCode, "text/html; charset=utf-8", []byte(message))
		return
	}
	context.Data(statusCode, "text/html; charset=utf-8", buffer.Bytes())
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

func (h *PublicHandlers) updateSubscriptionStatus(context *gin.Context, targetStatus string) {
	clientIP := context.ClientIP()
	if h.isRateLimited(clientIP) {
		context.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
		return
	}

	var payload subscriptionMutationRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	payload.SiteID = strings.TrimSpace(payload.SiteID)
	payload.Email = strings.TrimSpace(strings.ToLower(payload.Email))
	if payload.SiteID == "" || payload.Email == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "missing_fields"})
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

	subscriber, findErr := findSubscriber(context.Request.Context(), h.database, site.ID, payload.Email)
	if findErr != nil {
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			context.JSON(http.StatusNotFound, gin.H{"error": errorValueUnknownSubscription})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{"error": errorValueSaveSubscriberFailed})
		return
	}

	if targetStatus == model.SubscriberStatusConfirmed && subscriber.Status == model.SubscriberStatusUnsubscribed {
		context.JSON(http.StatusConflict, gin.H{"error": errorValueUnsubscribedAccount})
		return
	}
	if subscriber.Status == targetStatus {
		context.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	updateFields := map[string]any{
		"status": targetStatus,
	}
	now := time.Now().UTC()
	if targetStatus == model.SubscriberStatusConfirmed {
		updateFields["confirmed_at"] = now
		updateFields["unsubscribed_at"] = time.Time{}
	}
	if targetStatus == model.SubscriberStatusUnsubscribed {
		updateFields["unsubscribed_at"] = now
	}

	if err := h.database.Model(&subscriber).Updates(updateFields).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": errorValueSaveSubscriberFailed})
		return
	}

	if targetStatus == model.SubscriberStatusConfirmed {
		subscriber.Status = model.SubscriberStatusConfirmed
		subscriber.ConfirmedAt = now
		subscriber.UnsubscribedAt = time.Time{}
		h.applySubscriptionNotification(context.Request.Context(), site, subscriber)
	}

	context.JSON(http.StatusOK, gin.H{"status": "ok"})
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

	normalizedOriginHeader := strings.TrimSpace(originHeader)
	normalizedRefererHeader := strings.TrimSpace(refererHeader)
	normalizedURLValue := strings.TrimSpace(urlValue)

	for _, configuredOrigin := range allowedOrigins {
		normalizedAllowedOrigin := strings.TrimSpace(configuredOrigin)
		if normalizedAllowedOrigin == "" {
			continue
		}
		if normalizedOriginHeader == normalizedAllowedOrigin {
			return true
		}
		if normalizedURLValue != "" && strings.HasPrefix(normalizedURLValue, normalizedAllowedOrigin) {
			return true
		}
		if normalizedRefererHeader != "" && strings.HasPrefix(normalizedRefererHeader, normalizedAllowedOrigin) {
			return true
		}
	}
	return false
}

func findSubscriber(ctx context.Context, database *gorm.DB, siteID string, email string) (model.Subscriber, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	var subscriber model.Subscriber
	err := database.WithContext(ctx).First(&subscriber, "site_id = ? AND email = ?", siteID, normalizedEmail).Error
	return subscriber, err
}

func truncate(input string, max int) string {
	if len(input) <= max {
		return input
	}
	return input[:max]
}

func renderWidgetTemplate(site model.Site) (string, error) {
	var buffer bytes.Buffer
	executeErr := widgetJavaScriptTemplate.Execute(&buffer, map[string]any{
		"SiteID":             site.ID,
		"WidgetBubbleSide":   site.WidgetBubbleSide,
		"WidgetBottomOffset": site.WidgetBubbleBottomOffsetPx,
	})
	if executeErr != nil {
		return "", fmt.Errorf("render widget template: %w", executeErr)
	}
	return buffer.String(), nil
}

func renderSubscribeTemplate(site model.Site) (string, error) {
	var buffer bytes.Buffer
	executeErr := subscribeJavaScriptTemplate.Execute(&buffer, map[string]any{
		"SiteID": site.ID,
	})
	if executeErr != nil {
		return "", fmt.Errorf("render subscribe template: %w", executeErr)
	}
	return buffer.String(), nil
}

func renderPixelTemplate(site model.Site) (string, error) {
	var buffer bytes.Buffer
	executeErr := pixelJavaScriptTemplate.Execute(&buffer, map[string]any{
		"SiteID": site.ID,
	})
	if executeErr != nil {
		return "", fmt.Errorf("render pixel template: %w", executeErr)
	}
	return buffer.String(), nil
}

func (h *PublicHandlers) PixelJS(context *gin.Context) {
	siteID := strings.TrimSpace(context.Query("site_id"))
	if siteID == "" {
		context.String(http.StatusBadRequest, "/* missing site_id */")
		return
	}
	var site model.Site
	if err := h.database.First(&site, "id = ?", siteID).Error; err != nil {
		context.String(http.StatusNotFound, "/* unknown site */")
		return
	}
	script, tplErr := renderPixelTemplate(site)
	if tplErr != nil {
		context.String(http.StatusInternalServerError, "/* render error */")
		return
	}
	context.Data(http.StatusOK, "application/javascript; charset=utf-8", []byte(script))
}
