package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

// SiteSubscribeTestHandlers serves subscription test API endpoints.
type SiteSubscribeTestHandlers struct {
	database                  *gorm.DB
	logger                    *zap.Logger
	eventBroadcaster          *SubscriptionTestEventBroadcaster
	subscriptionNotifier      SubscriptionNotifier
	subscriptionNotifications bool
	publicBaseURL             string
	subscriptionTokenSecret   string
	subscriptionTokenTTL      time.Duration
	confirmationEmailSender   EmailSender
}

// NewSiteSubscribeTestHandlers constructs handlers for subscription test APIs.
func NewSiteSubscribeTestHandlers(database *gorm.DB, logger *zap.Logger, broadcaster *SubscriptionTestEventBroadcaster, subscriptionNotifier SubscriptionNotifier, subscriptionNotificationsEnabled bool, publicBaseURL string, subscriptionTokenSecret string, confirmationEmailSender EmailSender) *SiteSubscribeTestHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	normalizedPublicBaseURL := strings.TrimSpace(publicBaseURL)
	normalizedTokenSecret := strings.TrimSpace(subscriptionTokenSecret)
	return &SiteSubscribeTestHandlers{
		database:                  database,
		logger:                    logger,
		eventBroadcaster:          broadcaster,
		subscriptionNotifier:      resolveSubscriptionNotifier(subscriptionNotifier),
		subscriptionNotifications: subscriptionNotificationsEnabled,
		publicBaseURL:             normalizedPublicBaseURL,
		subscriptionTokenSecret:   normalizedTokenSecret,
		subscriptionTokenTTL:      defaultSubscriptionConfirmationTokenTTL,
		confirmationEmailSender:   confirmationEmailSender,
	}
}

// StreamSubscriptionTestEvents streams subscription test events as SSE.
func (handlers *SiteSubscribeTestHandlers) StreamSubscriptionTestEvents(context *gin.Context) {
	siteIdentifier := strings.TrimSpace(context.Param("id"))
	if siteIdentifier == "" {
		context.AbortWithStatus(http.StatusBadRequest)
		return
	}
	currentUser, userOK := CurrentUserFromContext(context)
	if !userOK {
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
	flusher, flushOK := writer.(http.Flusher)
	if !flushOK {
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
			payload, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				if handlers.logger != nil {
					handlers.logger.Debug("subscribe_test_event_encode_failed", zap.Error(marshalErr))
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

// CreateSubscription creates a test subscription for the site.
func (handlers *SiteSubscribeTestHandlers) CreateSubscription(context *gin.Context) {
	siteIdentifier := strings.TrimSpace(context.Param("id"))
	if siteIdentifier == "" {
		context.AbortWithStatus(http.StatusBadRequest)
		return
	}
	currentUser, userOK := CurrentUserFromContext(context)
	if !userOK {
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
	existingSubscriber, findErr := findSubscriber(context.Request.Context(), handlers.database, site.ID, payload.Email)
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		context.JSON(http.StatusInternalServerError, gin.H{"error": errorValueSaveSubscriberFailed})
		return
	}
	if findErr == nil {
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
			existingSubscriber.Status = model.SubscriberStatusPending
			existingSubscriber.UnsubscribedAt = time.Time{}
			existingSubscriber.ConfirmedAt = time.Time{}
			existingSubscriber.ConsentAt = now
			existingSubscriber.Name = payload.Name
			existingSubscriber.SourceURL = payload.SourceURL
			existingSubscriber.IP = truncate(clientIP, subscriptionIPMaxLength)
			existingSubscriber.UserAgent = truncate(context.Request.UserAgent(), subscriptionUserAgentMaxLength)
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
