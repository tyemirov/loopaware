package api

import (
	"context"
	"errors"
	"fmt"
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

// NewPublicHandlers constructs a PublicHandlers instance with the provided dependencies.
func NewPublicHandlers(database *gorm.DB, logger *zap.Logger, feedbackBroadcaster *FeedbackEventBroadcaster, subscriptionEvents *SubscriptionTestEventBroadcaster, notifier FeedbackNotifier, subscriptionNotifier SubscriptionNotifier, subscriptionNotificationsEnabled bool, publicBaseURL string, subscriptionTokenSecret string, confirmationEmailSender EmailSender) *PublicHandlers {
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

type widgetConfigResponse struct {
	SiteID                   string `json:"site_id"`
	WidgetBubbleSide         string `json:"widget_bubble_side"`
	WidgetBubbleBottomOffset int    `json:"widget_bubble_bottom_offset"`
}

type subscriptionLinkResponse struct {
	Heading        string `json:"heading"`
	Message        string `json:"message"`
	OpenURL        string `json:"open_url"`
	OpenLabel      string `json:"open_label"`
	UnsubscribeURL string `json:"unsubscribe_url"`
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
	context.JSON(http.StatusOK, widgetConfigResponse{
		SiteID:                   site.ID,
		WidgetBubbleSide:         site.WidgetBubbleSide,
		WidgetBubbleBottomOffset: site.WidgetBubbleBottomOffsetPx,
	})
}

// CreateSubscription registers a new subscriber.
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

// ConfirmSubscription confirms a pending subscription.
func (h *PublicHandlers) ConfirmSubscription(context *gin.Context) {
	h.updateSubscriptionStatus(context, model.SubscriberStatusConfirmed)
}

// Unsubscribe marks a subscriber as unsubscribed.
func (h *PublicHandlers) Unsubscribe(context *gin.Context) {
	h.updateSubscriptionStatus(context, model.SubscriberStatusUnsubscribed)
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
