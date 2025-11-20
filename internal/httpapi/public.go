package httpapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
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
	feedbackNotifier          FeedbackNotifier
	subscriptionNotifier      SubscriptionNotifier
	subscriptionNotifications bool
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
)

func NewPublicHandlers(database *gorm.DB, logger *zap.Logger, feedbackBroadcaster *FeedbackEventBroadcaster, notifier FeedbackNotifier, subscriptionNotifier SubscriptionNotifier, subscriptionNotificationsEnabled bool) *PublicHandlers {
	return &PublicHandlers{
		database:                  database,
		logger:                    logger,
		rateWindow:                30 * time.Second,
		maxRequestsPerIPPerWindow: 6,
		rateCountersByIP:          make(map[string]int),
		feedbackBroadcaster:       feedbackBroadcaster,
		feedbackNotifier:          resolveFeedbackNotifier(notifier),
		subscriptionNotifier:      resolveSubscriptionNotifier(subscriptionNotifier),
		subscriptionNotifications: subscriptionNotificationsEnabled,
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
	if !isOriginAllowed(site.AllowedOrigin, originHeader, refererHeader) {
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

func (h *PublicHandlers) applySubscriptionNotification(ctx context.Context, site model.Site, subscriber model.Subscriber) {
	if !h.subscriptionNotifications {
		return
	}
	if h.subscriptionNotifier == nil {
		return
	}
	if notifyErr := h.subscriptionNotifier.NotifySubscription(ctx, site, subscriber); notifyErr != nil {
		h.logger.Warn("subscription_notification_failed", zap.Error(notifyErr), zap.String("site_id", site.ID), zap.String("subscriber_id", subscriber.ID))
	}
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

	var buffer bytes.Buffer
	if err := subscribeDemoTemplate.Execute(&buffer, map[string]any{"SiteID": site.ID}); err != nil {
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
	if !isOriginAllowed(site.AllowedOrigin, originHeader, refererHeader) {
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
			h.applySubscriptionNotification(context.Request.Context(), site, existingSubscriber)
			context.JSON(http.StatusOK, gin.H{"status": "ok", "subscriber_id": existingSubscriber.ID})
			return
		}
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

	h.applySubscriptionNotification(context.Request.Context(), site, subscriber)
	context.JSON(http.StatusOK, gin.H{"status": "ok", "subscriber_id": subscriber.ID})
}

func (h *PublicHandlers) ConfirmSubscription(context *gin.Context) {
	h.updateSubscriptionStatus(context, model.SubscriberStatusConfirmed)
}

func (h *PublicHandlers) Unsubscribe(context *gin.Context) {
	h.updateSubscriptionStatus(context, model.SubscriberStatusUnsubscribed)
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
	if !isOriginAllowed(site.AllowedOrigin, originHeader, refererHeader) {
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

	context.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func isOriginAllowed(allowedOrigin string, originHeader string, refererHeader string) bool {
	normalizedAllowedOrigin := strings.TrimSpace(allowedOrigin)
	if normalizedAllowedOrigin == "" {
		return true
	}

	if originHeader != "" {
		return originHeader == normalizedAllowedOrigin
	}
	if refererHeader != "" {
		return strings.HasPrefix(refererHeader, normalizedAllowedOrigin)
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
