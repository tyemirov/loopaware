package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
)

const (
	jsonKeyError              = "error"
	jsonKeyEmail              = "email"
	jsonKeyName               = "name"
	jsonKeyRole               = "role"
	jsonKeyAvatar             = "avatar"
	jsonKeyAvatarURL          = "url"
	jsonKeyWidgetBubbleSide   = "widget_bubble_side"
	jsonKeyWidgetBubbleOffset = "widget_bubble_bottom_offset"

	errorValueInvalidJSON             = "invalid_json"
	errorValueMissingFields           = "missing_fields"
	errorValueSaveFailed              = "save_failed"
	errorValueMissingSite             = "missing_site"
	errorValueUnknownSite             = "unknown_site"
	errorValueQueryFailed             = "query_failed"
	errorValueNotAuthorized           = "not_authorized"
	errorValueInvalidOwner            = "invalid_owner"
	errorValueInvalidWidgetSide       = "invalid_widget_side"
	errorValueInvalidWidgetOffset     = "invalid_widget_offset"
	errorValueInvalidSubscriberStatus = "invalid_subscriber_status"
	errorValueNothingToUpdate         = "nothing_to_update"
	errorValueDeleteFailed            = "delete_failed"
	errorValueSiteExists              = "site_exists"
	errorValueStreamUnavailable       = "stream_unavailable"

	widgetScriptTemplate            = "<script defer src=\"%s/widget.js?site_id=%s\"></script>"
	siteFaviconURLTemplate          = "/api/sites/%s/favicon"
	widgetBubbleSideRight           = "right"
	widgetBubbleSideLeft            = "left"
	defaultWidgetBubbleSide         = widgetBubbleSideRight
	defaultWidgetBubbleBottomOffset = 16
	minWidgetBubbleBottomOffset     = 0
	maxWidgetBubbleBottomOffset     = 240
	feedbackCreatedEventName        = "feedback_created"
)

type SiteHandlers struct {
	database            *gorm.DB
	logger              *zap.Logger
	widgetBaseURL       string
	faviconManager      *SiteFaviconManager
	statsProvider       SiteStatisticsProvider
	feedbackBroadcaster *FeedbackEventBroadcaster
}

func NewSiteHandlers(database *gorm.DB, logger *zap.Logger, widgetBaseURL string, faviconManager *SiteFaviconManager, statsProvider SiteStatisticsProvider, feedbackBroadcaster *FeedbackEventBroadcaster) *SiteHandlers {
	if statsProvider == nil {
		statsProvider = NewDatabaseSiteStatisticsProvider(database)
	}
	return &SiteHandlers{
		database:            database,
		logger:              logger,
		widgetBaseURL:       normalizeWidgetBaseURL(widgetBaseURL),
		faviconManager:      faviconManager,
		statsProvider:       statsProvider,
		feedbackBroadcaster: feedbackBroadcaster,
	}
}

type createSiteRequest struct {
	Name                     string `json:"name"`
	AllowedOrigin            string `json:"allowed_origin"`
	SubscribeAllowedOrigins  string `json:"subscribe_allowed_origins"`
	WidgetAllowedOrigins     string `json:"widget_allowed_origins"`
	TrafficAllowedOrigins    string `json:"traffic_allowed_origins"`
	OwnerEmail               string `json:"owner_email"`
	WidgetBubbleSide         string `json:"widget_bubble_side"`
	WidgetBubbleBottomOffset *int   `json:"widget_bubble_bottom_offset"`
}

type updateSiteRequest struct {
	Name                     *string `json:"name"`
	AllowedOrigin            *string `json:"allowed_origin"`
	SubscribeAllowedOrigins  *string `json:"subscribe_allowed_origins"`
	WidgetAllowedOrigins     *string `json:"widget_allowed_origins"`
	TrafficAllowedOrigins    *string `json:"traffic_allowed_origins"`
	OwnerEmail               *string `json:"owner_email"`
	WidgetBubbleSide         *string `json:"widget_bubble_side"`
	WidgetBubbleBottomOffset *int    `json:"widget_bubble_bottom_offset"`
}

type siteResponse struct {
	ID                       string `json:"id"`
	Name                     string `json:"name"`
	AllowedOrigin            string `json:"allowed_origin"`
	SubscribeAllowedOrigins  string `json:"subscribe_allowed_origins"`
	WidgetAllowedOrigins     string `json:"widget_allowed_origins"`
	TrafficAllowedOrigins    string `json:"traffic_allowed_origins"`
	OwnerEmail               string `json:"owner_email"`
	FaviconURL               string `json:"favicon_url"`
	Widget                   string `json:"widget"`
	CreatedAt                int64  `json:"created_at"`
	FeedbackCount            int64  `json:"feedback_count"`
	SubscriberCount          int64  `json:"subscriber_count"`
	VisitCount               int64  `json:"visit_count"`
	UniqueVisitorCount       int64  `json:"unique_visitor_count"`
	WidgetBubbleSide         string `json:"widget_bubble_side"`
	WidgetBubbleBottomOffset int    `json:"widget_bubble_bottom_offset"`
}

type listSitesResponse struct {
	Sites []siteResponse `json:"sites"`
}

type siteMessagesResponse struct {
	SiteID   string                    `json:"site_id"`
	Messages []feedbackMessageResponse `json:"messages"`
}

type SiteSubscribersResponse struct {
	SiteID      string             `json:"site_id"`
	Subscribers []SubscriberRecord `json:"subscribers"`
}

type SubscriberRecord struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	CreatedAt      int64  `json:"created_at"`
	ConfirmedAt    int64  `json:"confirmed_at"`
	UnsubscribedAt int64  `json:"unsubscribed_at"`
}

type feedbackMessageResponse struct {
	ID        string `json:"id"`
	Contact   string `json:"contact"`
	Message   string `json:"message"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	CreatedAt int64  `json:"created_at"`
	Delivery  string `json:"delivery"`
}

type VisitStatsResponse struct {
	SiteID             string          `json:"site_id"`
	VisitCount         int64           `json:"visit_count"`
	UniqueVisitorCount int64           `json:"unique_visitor_count"`
	TopPages           []TopPageEntry  `json:"top_pages"`
	RecentVisits       []VisitLogEntry `json:"recent_visits"`
}

type TopPageEntry struct {
	Path       string `json:"path"`
	VisitCount int64  `json:"visit_count"`
}

type VisitLogEntry struct {
	URL        string `json:"url"`
	Path       string `json:"path"`
	IP         string `json:"ip"`
	Country    string `json:"country"`
	Browser    string `json:"browser"`
	UserAgent  string `json:"user_agent"`
	Referrer   string `json:"referrer"`
	VisitorID  string `json:"visitor_id"`
	OccurredAt int64  `json:"occurred_at"`
}

func (handlers *SiteHandlers) CurrentUser(context *gin.Context) {
	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return
	}

	responsePayload := gin.H{
		jsonKeyEmail: currentUser.Email,
		jsonKeyName:  currentUser.Name,
		jsonKeyRole:  currentUser.Role,
	}

	responsePayload[jsonKeyAvatar] = gin.H{jsonKeyAvatarURL: currentUser.PictureURL}

	context.JSON(http.StatusOK, responsePayload)
}

func (handlers *SiteHandlers) CreateSite(context *gin.Context) {
	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return
	}

	var payload createSiteRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}

	payload.Name = strings.TrimSpace(payload.Name)
	payload.AllowedOrigin = strings.TrimSpace(payload.AllowedOrigin)
	payload.SubscribeAllowedOrigins = normalizeAllowedOrigins(payload.SubscribeAllowedOrigins)
	payload.WidgetAllowedOrigins = normalizeAllowedOrigins(payload.WidgetAllowedOrigins)
	payload.TrafficAllowedOrigins = normalizeAllowedOrigins(payload.TrafficAllowedOrigins)
	creatorEmail := currentUser.normalizedEmail()
	desiredOwnerEmail := strings.ToLower(strings.TrimSpace(payload.OwnerEmail))
	if desiredOwnerEmail == "" {
		desiredOwnerEmail = creatorEmail
	}

	if payload.Name == "" || payload.AllowedOrigin == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingFields})
		return
	}

	if desiredOwnerEmail == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidOwner})
		return
	}

	widgetBubbleSide, widgetBubbleSideErr := sanitizeWidgetBubbleSide(payload.WidgetBubbleSide)
	if widgetBubbleSideErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidWidgetSide})
		return
	}
	widgetBubbleBottomOffset, widgetBubbleBottomOffsetErr := sanitizeWidgetBubbleBottomOffset(payload.WidgetBubbleBottomOffset)
	if widgetBubbleBottomOffsetErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidWidgetOffset})
		return
	}

	conflictExists, conflictCheckErr := handlers.allowedOriginConflictExists(payload.AllowedOrigin, "")
	if conflictCheckErr != nil {
		handlers.logger.Warn("check_allowed_origin_conflict", zap.Error(conflictCheckErr))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	if conflictExists {
		context.JSON(http.StatusConflict, gin.H{jsonKeyError: errorValueSiteExists})
		return
	}

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       payload.Name,
		AllowedOrigin:              payload.AllowedOrigin,
		SubscribeAllowedOrigins:    payload.SubscribeAllowedOrigins,
		WidgetAllowedOrigins:       payload.WidgetAllowedOrigins,
		TrafficAllowedOrigins:      payload.TrafficAllowedOrigins,
		OwnerEmail:                 desiredOwnerEmail,
		CreatorEmail:               creatorEmail,
		FaviconOrigin:              primaryAllowedOrigin(payload.AllowedOrigin),
		WidgetBubbleSide:           widgetBubbleSide,
		WidgetBubbleBottomOffsetPx: widgetBubbleBottomOffset,
	}

	if err := handlers.database.Create(&site).Error; err != nil {
		handlers.logger.Warn("create_site", zap.Error(err))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}

	handlers.scheduleFaviconFetch(site)

	context.JSON(http.StatusOK, handlers.toSiteResponse(handlers.ginRequestContext(context), site, 0))
}

func (handlers *SiteHandlers) ListSites(context *gin.Context) {
	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return
	}

	var sites []model.Site

	query := handlers.database.Model(&model.Site{})
	if !currentUser.hasRole(RoleAdmin) {
		normalizedEmail := currentUser.normalizedEmail()
		query = query.Where("(LOWER(owner_email) = ? OR LOWER(creator_email) = ?)", normalizedEmail, normalizedEmail)
	}

	if err := query.Order("created_at desc").Find(&sites).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}

	responses := make([]siteResponse, 0, len(sites))
	requestContext := handlers.ginRequestContext(context)
	for _, site := range sites {
		feedbackCount := handlers.feedbackCount(requestContext, site.ID)
		handlers.scheduleFaviconFetch(site)
		responses = append(responses, handlers.toSiteResponse(requestContext, site, feedbackCount))
	}

	context.JSON(http.StatusOK, listSitesResponse{Sites: responses})
}

func (handlers *SiteHandlers) UserAvatar(context *gin.Context) {
	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return
	}

	trimmedEmail := strings.ToLower(strings.TrimSpace(currentUser.Email))
	if trimmedEmail == "" {
		context.AbortWithStatus(http.StatusNotFound)
		return
	}

	var user model.User
	if err := handlers.database.First(&user, "email = ?", trimmedEmail).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			context.AbortWithStatus(http.StatusNotFound)
			return
		}
		handlers.logger.Warn("load_user_avatar", zap.Error(err))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}

	if len(user.AvatarData) == 0 {
		context.AbortWithStatus(http.StatusNotFound)
		return
	}

	contentType := user.AvatarContentType
	if contentType == "" {
		contentType = defaultAvatarMimeType
	}
	context.Header("Cache-Control", "no-cache")
	context.Data(http.StatusOK, contentType, user.AvatarData)
}

func (handlers *SiteHandlers) SiteFavicon(context *gin.Context) {
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
	if err := handlers.database.First(&site, "id = ?", siteIdentifier).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSite})
		return
	}

	if !currentUser.canManageSite(site) {
		context.JSON(http.StatusForbidden, gin.H{jsonKeyError: errorValueNotAuthorized})
		return
	}

	if len(site.FaviconData) == 0 {
		context.AbortWithStatus(http.StatusNotFound)
		return
	}

	contentType := strings.TrimSpace(site.FaviconContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	context.Header("Cache-Control", "public, max-age=300")
	context.Data(http.StatusOK, contentType, site.FaviconData)
}

func (handlers *SiteHandlers) StreamFaviconUpdates(ginContext *gin.Context) {
	currentUser, ok := CurrentUserFromContext(ginContext)
	if !ok {
		ginContext.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return
	}
	if handlers.faviconManager == nil {
		ginContext.JSON(http.StatusServiceUnavailable, gin.H{jsonKeyError: errorValueStreamUnavailable})
		return
	}
	subscription := handlers.faviconManager.Subscribe()
	if subscription == nil {
		ginContext.JSON(http.StatusServiceUnavailable, gin.H{jsonKeyError: errorValueStreamUnavailable})
		return
	}
	defer subscription.Close()

	ginContext.Header("Content-Type", "text/event-stream")
	ginContext.Header("Cache-Control", "no-cache")
	ginContext.Header("Connection", "keep-alive")

	flusher, flushable := ginContext.Writer.(http.Flusher)
	if !flushable {
		ginContext.JSON(http.StatusServiceUnavailable, gin.H{jsonKeyError: errorValueStreamUnavailable})
		return
	}

	ginContext.Writer.WriteHeaderNow()
	flusher.Flush()

	requestContext := ginContext.Request.Context()

	for {
		select {
		case <-requestContext.Done():
			return
		case event, ok := <-subscription.Events():
			if !ok {
				return
			}
			if !handlers.userCanAccessSite(context.Background(), currentUser, event.SiteID) {
				continue
			}
			payload := struct {
				SiteID     string `json:"site_id"`
				FaviconURL string `json:"favicon_url"`
				UpdatedAt  int64  `json:"updated_at"`
			}{
				SiteID:     event.SiteID,
				FaviconURL: event.FaviconURL,
				UpdatedAt:  event.UpdatedAt.UTC().Unix(),
			}
			serializedPayload, marshalErr := json.Marshal(payload)
			if marshalErr != nil {
				if handlers.logger != nil {
					handlers.logger.Debug("marshal_favicon_event_failed", zap.Error(marshalErr))
				}
				continue
			}
			var buffer bytes.Buffer
			buffer.WriteString("event: favicon_updated\n")
			buffer.WriteString("data: ")
			buffer.Write(serializedPayload)
			buffer.WriteString("\n\n")
			if _, writeErr := ginContext.Writer.Write(buffer.Bytes()); writeErr != nil {
				return
			}
			flusher.Flush()
			if handlers.logger != nil {
				handlers.logger.Debug(
					"stream_favicon_event",
					zap.String("site_id", event.SiteID),
					zap.String("favicon_url", event.FaviconURL),
				)
			}
		}
	}
}

func (handlers *SiteHandlers) StreamFeedbackUpdates(ginContext *gin.Context) {
	currentUser, ok := CurrentUserFromContext(ginContext)
	if !ok {
		ginContext.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return
	}
	if handlers.feedbackBroadcaster == nil {
		ginContext.JSON(http.StatusServiceUnavailable, gin.H{jsonKeyError: errorValueStreamUnavailable})
		return
	}
	subscription := handlers.feedbackBroadcaster.Subscribe()
	if subscription == nil {
		ginContext.JSON(http.StatusServiceUnavailable, gin.H{jsonKeyError: errorValueStreamUnavailable})
		return
	}
	defer subscription.Close()

	ginContext.Header("Content-Type", "text/event-stream")
	ginContext.Header("Cache-Control", "no-cache")
	ginContext.Header("Connection", "keep-alive")

	flusher, flushable := ginContext.Writer.(http.Flusher)
	if !flushable {
		ginContext.JSON(http.StatusServiceUnavailable, gin.H{jsonKeyError: errorValueStreamUnavailable})
		return
	}

	ginContext.Writer.WriteHeaderNow()
	flusher.Flush()

	requestContext := ginContext.Request.Context()

	for {
		select {
		case <-requestContext.Done():
			return
		case event, ok := <-subscription.Events():
			if !ok {
				return
			}
			if event.SiteID == "" {
				continue
			}
			if !handlers.userCanAccessSite(context.Background(), currentUser, event.SiteID) {
				continue
			}
			createdAt := event.CreatedAt.UTC().Unix()
			if createdAt <= 0 {
				createdAt = time.Now().UTC().Unix()
			}
			payload := struct {
				SiteID        string `json:"site_id"`
				FeedbackID    string `json:"feedback_id,omitempty"`
				CreatedAt     int64  `json:"created_at"`
				FeedbackCount int64  `json:"feedback_count"`
			}{
				SiteID:        event.SiteID,
				FeedbackID:    event.FeedbackID,
				CreatedAt:     createdAt,
				FeedbackCount: event.FeedbackCount,
			}
			serializedPayload, marshalErr := json.Marshal(payload)
			if marshalErr != nil {
				if handlers.logger != nil {
					handlers.logger.Debug("marshal_feedback_event_failed", zap.Error(marshalErr))
				}
				continue
			}
			var buffer bytes.Buffer
			buffer.WriteString("event: ")
			buffer.WriteString(feedbackCreatedEventName)
			buffer.WriteString("\n")
			buffer.WriteString("data: ")
			buffer.Write(serializedPayload)
			buffer.WriteString("\n\n")
			if _, writeErr := ginContext.Writer.Write(buffer.Bytes()); writeErr != nil {
				return
			}
			flusher.Flush()
			if handlers.logger != nil {
				handlers.logger.Debug(
					"stream_feedback_event",
					zap.String("site_id", event.SiteID),
					zap.String("feedback_id", event.FeedbackID),
				)
			}
		}
	}
}

func (handlers *SiteHandlers) UpdateSite(context *gin.Context) {
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

	var payload updateSiteRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}

	if payload.Name == nil && payload.AllowedOrigin == nil && payload.SubscribeAllowedOrigins == nil && payload.WidgetAllowedOrigins == nil && payload.TrafficAllowedOrigins == nil && payload.OwnerEmail == nil && payload.WidgetBubbleSide == nil && payload.WidgetBubbleBottomOffset == nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueNothingToUpdate})
		return
	}

	var site model.Site
	if err := handlers.database.First(&site, "id = ?", siteIdentifier).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSite})
		return
	}

	if !currentUser.canManageSite(site) {
		context.JSON(http.StatusForbidden, gin.H{jsonKeyError: errorValueNotAuthorized})
		return
	}

	if payload.Name != nil {
		trimmed := strings.TrimSpace(*payload.Name)
		if trimmed == "" {
			context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingFields})
			return
		}
		site.Name = trimmed
	}

	originChanged := false
	if payload.AllowedOrigin != nil {
		trimmed := strings.TrimSpace(*payload.AllowedOrigin)
		if trimmed == "" {
			context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingFields})
			return
		}
		if !strings.EqualFold(strings.TrimSpace(site.AllowedOrigin), trimmed) {
			conflictExists, conflictCheckErr := handlers.allowedOriginConflictExists(trimmed, site.ID)
			if conflictCheckErr != nil {
				handlers.logger.Warn("check_allowed_origin_conflict", zap.Error(conflictCheckErr))
				context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
				return
			}
			if conflictExists {
				context.JSON(http.StatusConflict, gin.H{jsonKeyError: errorValueSiteExists})
				return
			}
			originChanged = true
		}
		site.AllowedOrigin = trimmed
	}

	if payload.OwnerEmail != nil {
		trimmed := strings.ToLower(strings.TrimSpace(*payload.OwnerEmail))
		if trimmed == "" {
			context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidOwner})
			return
		}
		site.OwnerEmail = trimmed
	}

	if payload.SubscribeAllowedOrigins != nil {
		site.SubscribeAllowedOrigins = normalizeAllowedOrigins(*payload.SubscribeAllowedOrigins)
	}
	if payload.WidgetAllowedOrigins != nil {
		site.WidgetAllowedOrigins = normalizeAllowedOrigins(*payload.WidgetAllowedOrigins)
	}
	if payload.TrafficAllowedOrigins != nil {
		site.TrafficAllowedOrigins = normalizeAllowedOrigins(*payload.TrafficAllowedOrigins)
	}

	if payload.WidgetBubbleSide != nil {
		normalizedSide, sideErr := sanitizeWidgetBubbleSide(*payload.WidgetBubbleSide)
		if sideErr != nil {
			context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidWidgetSide})
			return
		}
		site.WidgetBubbleSide = normalizedSide
	}

	if payload.WidgetBubbleBottomOffset != nil {
		offset, offsetErr := sanitizeWidgetBubbleBottomOffset(payload.WidgetBubbleBottomOffset)
		if offsetErr != nil {
			context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidWidgetOffset})
			return
		}
		site.WidgetBubbleBottomOffsetPx = offset
	}

	primaryOriginValue := primaryAllowedOrigin(site.AllowedOrigin)
	normalizedPrimaryOrigin := strings.TrimSpace(primaryOriginValue)

	if originChanged {
		site.FaviconData = nil
		site.FaviconContentType = ""
		site.FaviconFetchedAt = time.Time{}
		site.FaviconLastAttemptAt = time.Time{}
		site.FaviconOrigin = normalizedPrimaryOrigin
	} else if strings.TrimSpace(site.FaviconOrigin) == "" {
		site.FaviconOrigin = normalizedPrimaryOrigin
	}

	if err := handlers.database.Save(&site).Error; err != nil {
		handlers.logger.Warn("update_site", zap.Error(err))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}

	ctx := handlers.ginRequestContext(context)
	feedbackCount := handlers.feedbackCount(ctx, site.ID)
	handlers.scheduleFaviconFetch(site)
	context.JSON(http.StatusOK, handlers.toSiteResponse(ctx, site, feedbackCount))
}

func (handlers *SiteHandlers) DeleteSite(context *gin.Context) {
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
	if err := handlers.database.First(&site, "id = ?", siteIdentifier).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSite})
		return
	}

	if !currentUser.canManageSite(site) {
		context.JSON(http.StatusForbidden, gin.H{jsonKeyError: errorValueNotAuthorized})
		return
	}

	deleteErr := handlers.database.Transaction(func(transaction *gorm.DB) error {
		if err := transaction.Where("site_id = ?", site.ID).Delete(&model.Feedback{}).Error; err != nil {
			return err
		}
		if err := transaction.Delete(&model.Site{ID: site.ID}).Error; err != nil {
			return err
		}
		return nil
	})
	if deleteErr != nil {
		handlers.logger.Warn("delete_site", zap.Error(deleteErr))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueDeleteFailed})
		return
	}

	context.Status(http.StatusNoContent)
	context.Writer.WriteHeaderNow()
}

func (handlers *SiteHandlers) ListMessagesBySite(context *gin.Context) {
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
	if err := handlers.database.First(&site, "id = ?", siteIdentifier).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSite})
		return
	}

	if !currentUser.canManageSite(site) {
		context.JSON(http.StatusForbidden, gin.H{jsonKeyError: errorValueNotAuthorized})
		return
	}

	var feedbacks []model.Feedback
	if err := handlers.database.
		Where("site_id = ?", site.ID).
		Order("created_at desc").
		Find(&feedbacks).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}

	messageResponses := make([]feedbackMessageResponse, 0, len(feedbacks))
	for _, feedback := range feedbacks {
		messageResponses = append(messageResponses, feedbackMessageResponse{
			ID:        feedback.ID,
			Contact:   feedback.Contact,
			Message:   feedback.Message,
			IP:        feedback.IP,
			UserAgent: feedback.UserAgent,
			CreatedAt: feedback.CreatedAt.Unix(),
			Delivery:  feedback.Delivery,
		})
	}

	context.JSON(http.StatusOK, siteMessagesResponse{SiteID: site.ID, Messages: messageResponses})
}

func (handlers *SiteHandlers) VisitStats(context *gin.Context) {
	site, _, ok := handlers.resolveAuthorizedSite(context)
	if !ok {
		return
	}

	total, err := handlers.statsProvider.VisitCount(context.Request.Context(), site.ID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	unique, err := handlers.statsProvider.UniqueVisitorCount(context.Request.Context(), site.ID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	topPages, err := handlers.statsProvider.TopPages(context.Request.Context(), site.ID, 10)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	entries := make([]TopPageEntry, 0, len(topPages))
	for _, page := range topPages {
		entry := TopPageEntry(page)
		entries = append(entries, entry)
	}
	recentVisits, err := handlers.recentVisits(context.Request.Context(), site.ID, 6)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	context.JSON(http.StatusOK, VisitStatsResponse{
		SiteID:             site.ID,
		VisitCount:         total,
		UniqueVisitorCount: unique,
		TopPages:           entries,
		RecentVisits:       recentVisits,
	})
}

func (handlers *SiteHandlers) recentVisits(ctx context.Context, siteID string, limit int) ([]VisitLogEntry, error) {
	if strings.TrimSpace(siteID) == "" || handlers.database == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}
	var visits []model.SiteVisit
	if err := handlers.database.
		WithContext(ctx).
		Where("site_id = ?", siteID).
		Order("occurred_at desc").
		Limit(limit).
		Find(&visits).Error; err != nil {
		return nil, err
	}
	entries := make([]VisitLogEntry, 0, len(visits))
	for _, visit := range visits {
		entries = append(entries, VisitLogEntry{
			URL:        visit.URL,
			Path:       visit.Path,
			IP:         visit.IP,
			Country:    classifyVisitCountry(visit.IP),
			Browser:    classifyVisitBrowser(visit.UserAgent),
			UserAgent:  visit.UserAgent,
			Referrer:   visit.Referrer,
			VisitorID:  visit.VisitorID,
			OccurredAt: visit.OccurredAt.Unix(),
		})
	}
	return entries, nil
}

func classifyVisitBrowser(userAgent string) string {
	normalized := strings.ToLower(strings.TrimSpace(userAgent))
	if normalized == "" {
		return "Unknown"
	}
	switch {
	case strings.Contains(normalized, "edg/"):
		return "Microsoft Edge"
	case strings.Contains(normalized, "opr/") || strings.Contains(normalized, "opera"):
		return "Opera"
	case strings.Contains(normalized, "chrome") && strings.Contains(normalized, "safari"):
		return "Google Chrome"
	case strings.Contains(normalized, "safari"):
		return "Safari"
	case strings.Contains(normalized, "firefox"):
		return "Firefox"
	case strings.Contains(normalized, "msie") || strings.Contains(normalized, "trident/"):
		return "Internet Explorer"
	case strings.Contains(normalized, "curl"):
		return "curl"
	default:
		return "Other"
	}
}

func classifyVisitCountry(ipAddress string) string {
	trimmed := strings.TrimSpace(ipAddress)
	if trimmed == "" {
		return "Unknown"
	}
	parsed := net.ParseIP(trimmed)
	if parsed == nil {
		return "Unknown"
	}
	if parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast() || parsed.IsLinkLocalMulticast() {
		return "Local network"
	}
	return "Unknown"
}

func (handlers *SiteHandlers) ListSubscribers(context *gin.Context) {
	site, currentUser, ok := handlers.resolveAuthorizedSite(context)
	if !ok {
		return
	}

	searchQuery := strings.TrimSpace(context.Query("q"))

	var subscribers []model.Subscriber
	query := handlers.database.Where("site_id = ?", site.ID)
	if searchQuery != "" {
		like := "%" + strings.ToLower(searchQuery) + "%"
		query = query.Where("(LOWER(email) LIKE ? OR LOWER(name) LIKE ?)", like, like)
	}
	if err := query.Order("created_at desc").Find(&subscribers).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}

	response := SiteSubscribersResponse{SiteID: site.ID}
	for _, subscriber := range subscribers {
		response.Subscribers = append(response.Subscribers, SubscriberRecord{
			ID:             subscriber.ID,
			Email:          subscriber.Email,
			Name:           subscriber.Name,
			Status:         subscriber.Status,
			CreatedAt:      subscriber.CreatedAt.Unix(),
			ConfirmedAt:    subscriber.ConfirmedAt.Unix(),
			UnsubscribedAt: subscriber.UnsubscribedAt.Unix(),
		})
	}

	_ = currentUser // retained for symmetry; auth already enforced
	context.JSON(http.StatusOK, response)
}

func (handlers *SiteHandlers) ExportSubscribers(context *gin.Context) {
	site, _, ok := handlers.resolveAuthorizedSite(context)
	if !ok {
		return
	}

	var subscribers []model.Subscriber
	if err := handlers.database.Where("site_id = ?", site.ID).Order("created_at desc").Find(&subscribers).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}

	context.Header("Content-Type", "text/csv")
	context.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="subscribers-%s.csv"`, site.ID))

	csvWriter := csv.NewWriter(context.Writer)
	_ = csvWriter.Write([]string{"email", "name", "status", "created_at", "confirmed_at", "unsubscribed_at"})
	for _, subscriber := range subscribers {
		record := []string{
			subscriber.Email,
			subscriber.Name,
			subscriber.Status,
			fmt.Sprintf("%d", subscriber.CreatedAt.Unix()),
			fmt.Sprintf("%d", subscriber.ConfirmedAt.Unix()),
			fmt.Sprintf("%d", subscriber.UnsubscribedAt.Unix()),
		}
		_ = csvWriter.Write(record)
	}
	csvWriter.Flush()
}

func (handlers *SiteHandlers) UpdateSubscriberStatus(context *gin.Context) {
	site, _, ok := handlers.resolveAuthorizedSite(context)
	if !ok {
		return
	}

	subscriberID := strings.TrimSpace(context.Param("subscriber_id"))
	if subscriberID == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingFields})
		return
	}

	var payload struct {
		Status string `json:"status"`
	}
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}
	desiredStatus := strings.TrimSpace(payload.Status)
	if desiredStatus != model.SubscriberStatusConfirmed && desiredStatus != model.SubscriberStatusUnsubscribed {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidSubscriberStatus})
		return
	}

	var subscriber model.Subscriber
	if err := handlers.database.Where("id = ? AND site_id = ?", subscriberID, site.ID).First(&subscriber).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSubscription})
		return
	}

	updateFields := map[string]any{
		"status": desiredStatus,
	}
	now := time.Now().UTC()
	if desiredStatus == model.SubscriberStatusUnsubscribed {
		updateFields["unsubscribed_at"] = now
	}
	if desiredStatus == model.SubscriberStatusConfirmed {
		updateFields["confirmed_at"] = now
		updateFields["unsubscribed_at"] = time.Time{}
	}

	if err := handlers.database.Model(&subscriber).Updates(updateFields).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}

	context.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (handlers *SiteHandlers) DeleteSubscriber(context *gin.Context) {
	site, _, ok := handlers.resolveAuthorizedSite(context)
	if !ok {
		return
	}

	subscriberID := strings.TrimSpace(context.Param("subscriber_id"))
	if subscriberID == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingFields})
		return
	}

	deleteResult := handlers.database.Where("id = ? AND site_id = ?", subscriberID, site.ID).Delete(&model.Subscriber{})
	if deleteResult.Error != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}
	if deleteResult.RowsAffected == 0 {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSubscription})
		return
	}

	context.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (handlers *SiteHandlers) resolveAuthorizedSite(context *gin.Context) (model.Site, *CurrentUser, bool) {
	siteIdentifier := strings.TrimSpace(context.Param("id"))
	if siteIdentifier == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingSite})
		return model.Site{}, nil, false
	}

	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return model.Site{}, nil, false
	}

	var site model.Site
	if err := handlers.database.First(&site, "id = ?", siteIdentifier).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSite})
		return model.Site{}, nil, false
	}

	if !currentUser.canManageSite(site) {
		context.JSON(http.StatusForbidden, gin.H{jsonKeyError: errorValueNotAuthorized})
		return model.Site{}, nil, false
	}

	return site, currentUser, true
}
func (handlers *SiteHandlers) toSiteResponse(ctx context.Context, site model.Site, feedbackCount int64) siteResponse {
	widgetBase := handlers.widgetBaseURL
	if widgetBase == "" {
		widgetBaseOrigin := primaryAllowedOrigin(site.AllowedOrigin)
		widgetBase = normalizeWidgetBaseURL(widgetBaseOrigin)
	}
	ensureWidgetBubblePlacementDefaults(&site)

	faviconURL := ""
	if len(site.FaviconData) > 0 {
		faviconURL = versionedSiteFaviconURL(site.ID, site.FaviconFetchedAt)
	}

	return siteResponse{
		ID:                       site.ID,
		Name:                     site.Name,
		AllowedOrigin:            site.AllowedOrigin,
		SubscribeAllowedOrigins:  site.SubscribeAllowedOrigins,
		WidgetAllowedOrigins:     site.WidgetAllowedOrigins,
		TrafficAllowedOrigins:    site.TrafficAllowedOrigins,
		OwnerEmail:               site.OwnerEmail,
		FaviconURL:               faviconURL,
		Widget:                   fmt.Sprintf(widgetScriptTemplate, widgetBase, site.ID),
		CreatedAt:                site.CreatedAt.UTC().Unix(),
		FeedbackCount:            feedbackCount,
		SubscriberCount:          handlers.subscriberCount(ctx, site.ID),
		VisitCount:               handlers.visitCount(ctx, site.ID),
		UniqueVisitorCount:       handlers.uniqueVisitorCount(ctx, site.ID),
		WidgetBubbleSide:         site.WidgetBubbleSide,
		WidgetBubbleBottomOffset: site.WidgetBubbleBottomOffsetPx,
	}
}

func normalizeAllowedOrigins(rawValue string) string {
	origins := parseAllowedOrigins(rawValue)
	return strings.Join(origins, " ")
}

func (handlers *SiteHandlers) feedbackCount(ctx context.Context, siteID string) int64 {
	if handlers.statsProvider == nil {
		return 0
	}
	count, err := handlers.statsProvider.FeedbackCount(ctx, siteID)
	if err != nil && handlers.logger != nil {
		handlers.logger.Debug("feedback_count_failed", zap.String("site_id", siteID), zap.Error(err))
		return 0
	}
	return count
}

func (handlers *SiteHandlers) subscriberCount(ctx context.Context, siteID string) int64 {
	if handlers.statsProvider == nil {
		return 0
	}
	count, err := handlers.statsProvider.SubscriberCount(ctx, siteID)
	if err != nil && handlers.logger != nil {
		handlers.logger.Debug("subscriber_count_failed", zap.String("site_id", siteID), zap.Error(err))
		return 0
	}
	return count
}

func (handlers *SiteHandlers) visitCount(ctx context.Context, siteID string) int64 {
	if handlers.statsProvider == nil {
		return 0
	}
	count, err := handlers.statsProvider.VisitCount(ctx, siteID)
	if err != nil && handlers.logger != nil {
		handlers.logger.Debug("visit_count_failed", zap.String("site_id", siteID), zap.Error(err))
		return 0
	}
	return count
}

func (handlers *SiteHandlers) uniqueVisitorCount(ctx context.Context, siteID string) int64 {
	if handlers.statsProvider == nil {
		return 0
	}
	count, err := handlers.statsProvider.UniqueVisitorCount(ctx, siteID)
	if err != nil && handlers.logger != nil {
		handlers.logger.Debug("unique_visitor_count_failed", zap.String("site_id", siteID), zap.Error(err))
		return 0
	}
	return count
}

func (handlers *SiteHandlers) scheduleFaviconFetch(site model.Site) {
	if handlers.faviconManager == nil {
		return
	}
	handlers.faviconManager.ScheduleFetch(site)
}

func (handlers *SiteHandlers) userCanAccessSite(ctx context.Context, currentUser *CurrentUser, siteID string) bool {
	if handlers.database == nil || currentUser == nil {
		return false
	}
	var site model.Site
	if err := handlers.database.WithContext(ctx).Select("id", "owner_email", "creator_email").First(&site, "id = ?", siteID).Error; err != nil {
		return false
	}
	return currentUser.canManageSite(site)
}

func (handlers *SiteHandlers) allowedOriginConflictExists(allowedOrigin string, excludeSiteID string) (bool, error) {
	if handlers.database == nil {
		return false, nil
	}
	allowedOrigins := parseAllowedOrigins(allowedOrigin)
	if len(allowedOrigins) == 0 {
		return false, nil
	}

	normalizedOrigins := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		normalized := strings.ToLower(strings.TrimSpace(origin))
		if normalized == "" {
			continue
		}
		normalizedOrigins[normalized] = struct{}{}
	}
	if len(normalizedOrigins) == 0 {
		return false, nil
	}

	query := handlers.database.Model(&model.Site{}).Select("id", "allowed_origin")
	excludedIdentifier := strings.TrimSpace(excludeSiteID)
	if excludedIdentifier != "" {
		query = query.Where("id <> ?", excludedIdentifier)
	}
	var existingSites []model.Site
	if err := query.Find(&existingSites).Error; err != nil {
		return false, err
	}

	for _, site := range existingSites {
		for _, existingOrigin := range parseAllowedOrigins(site.AllowedOrigin) {
			normalized := strings.ToLower(strings.TrimSpace(existingOrigin))
			if normalized == "" {
				continue
			}
			if _, ok := normalizedOrigins[normalized]; ok {
				return true, nil
			}
		}
	}

	return false, nil
}

func normalizeWidgetBaseURL(value string) string {
	trimmed := strings.TrimSpace(value)
	return strings.TrimRight(trimmed, "/")
}

func sanitizeWidgetBubbleSide(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return defaultWidgetBubbleSide, nil
	}
	if normalized != widgetBubbleSideLeft && normalized != widgetBubbleSideRight {
		return "", errors.New("invalid widget bubble side")
	}
	return normalized, nil
}

func sanitizeWidgetBubbleBottomOffset(value *int) (int, error) {
	if value == nil {
		return defaultWidgetBubbleBottomOffset, nil
	}
	offset := *value
	if offset < minWidgetBubbleBottomOffset || offset > maxWidgetBubbleBottomOffset {
		return 0, errors.New("invalid widget bubble bottom offset")
	}
	return offset, nil
}

func ensureWidgetBubblePlacementDefaults(site *model.Site) {
	if site == nil {
		return
	}
	side := strings.ToLower(strings.TrimSpace(site.WidgetBubbleSide))
	if side != widgetBubbleSideLeft && side != widgetBubbleSideRight {
		side = defaultWidgetBubbleSide
	}
	site.WidgetBubbleSide = side
	if site.WidgetBubbleBottomOffsetPx < minWidgetBubbleBottomOffset || site.WidgetBubbleBottomOffsetPx > maxWidgetBubbleBottomOffset {
		site.WidgetBubbleBottomOffsetPx = defaultWidgetBubbleBottomOffset
	}
}

func (handlers *SiteHandlers) ginRequestContext(ginContext *gin.Context) context.Context {
	if ginContext != nil && ginContext.Request != nil {
		return ginContext.Request.Context()
	}
	return context.Background()
}
