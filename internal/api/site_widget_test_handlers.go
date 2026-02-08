package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
)

// SiteWidgetTestHandlers serves widget test feedback API endpoints.
type SiteWidgetTestHandlers struct {
	database            *gorm.DB
	logger              *zap.Logger
	feedbackBroadcaster *FeedbackEventBroadcaster
	notifier            FeedbackNotifier
}

// NewSiteWidgetTestHandlers constructs handlers for widget test feedback APIs.
func NewSiteWidgetTestHandlers(database *gorm.DB, logger *zap.Logger, feedbackBroadcaster *FeedbackEventBroadcaster, notifier FeedbackNotifier) *SiteWidgetTestHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SiteWidgetTestHandlers{
		database:            database,
		logger:              logger,
		feedbackBroadcaster: feedbackBroadcaster,
		notifier:            resolveFeedbackNotifier(notifier),
	}
}

type widgetTestFeedbackRequest struct {
	Contact string `json:"contact"`
	Message string `json:"message"`
}

// SubmitWidgetTestFeedback records feedback submitted from the widget test UI.
func (handlers *SiteWidgetTestHandlers) SubmitWidgetTestFeedback(context *gin.Context) {
	siteIdentifier := strings.TrimSpace(context.Param("id"))
	if siteIdentifier == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingSite})
		return
	}

	currentUser, userOK := CurrentUserFromContext(context)
	if !userOK {
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
	if bindErr := context.BindJSON(&payload); bindErr != nil {
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

	applyFeedbackNotification(context.Request.Context(), handlers.database, handlers.logger, handlers.notifier, site, &feedback)
	broadcastFeedbackEvent(handlers.database, handlers.logger, handlers.feedbackBroadcaster, context.Request.Context(), feedback)

	context.JSON(http.StatusOK, gin.H{"status": "ok"})
}
