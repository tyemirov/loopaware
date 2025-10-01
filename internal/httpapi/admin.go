package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
)

const (
	jsonKeyError         = "error"
	jsonKeyIdentifier    = "id"
	jsonKeyName          = "name"
	jsonKeyAllowedOrigin = "allowed_origin"
	jsonKeyWidget        = "widget"

	errorValueInvalidJSON   = "invalid_json"
	errorValueMissingFields = "missing_fields"
	errorValueSaveFailed    = "save_failed"
	errorValueMissingSite   = "missing_site"
	errorValueUnknownSite   = "unknown_site"
	errorValueQueryFailed   = "query_failed"

	widgetScriptTemplate = "<script src=\"%s/widget.js?site_id=%s\"></script>"
)

type AdminHandlers struct {
	database         *gorm.DB
	logger           *zap.Logger
	adminBearerToken string
}

func NewAdminHandlers(database *gorm.DB, logger *zap.Logger, adminBearerToken string) *AdminHandlers {
	return &AdminHandlers{
		database:         database,
		logger:           logger,
		adminBearerToken: adminBearerToken,
	}
}

type createSiteRequest struct {
	Name          string `json:"name"`
	AllowedOrigin string `json:"allowed_origin"`
}

type siteMessagesResponse struct {
	SiteID   string                    `json:"site_id"`
	Messages []feedbackMessageResponse `json:"messages"`
}

type feedbackMessageResponse struct {
	ID        string `json:"id"`
	Contact   string `json:"contact"`
	Message   string `json:"message"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	CreatedAt int64  `json:"created_at"`
}

func (adminHandlers *AdminHandlers) CreateSite(context *gin.Context) {
	var payload createSiteRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}
	payload.Name = strings.TrimSpace(payload.Name)
	payload.AllowedOrigin = strings.TrimSpace(payload.AllowedOrigin)
	if payload.Name == "" || payload.AllowedOrigin == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingFields})
		return
	}
	site := model.Site{
		ID:            storage.NewID(),
		Name:          payload.Name,
		AllowedOrigin: payload.AllowedOrigin,
	}
	if err := adminHandlers.database.Create(&site).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}
	context.JSON(http.StatusOK, gin.H{
		jsonKeyIdentifier:    site.ID,
		jsonKeyName:          site.Name,
		jsonKeyAllowedOrigin: site.AllowedOrigin,
		jsonKeyWidget:        fmt.Sprintf(widgetScriptTemplate, site.AllowedOrigin, site.ID),
	})
}

func (adminHandlers *AdminHandlers) ListMessagesBySite(context *gin.Context) {
	siteID := strings.TrimSpace(context.Param("id"))
	if siteID == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingSite})
		return
	}
	var site model.Site
	if err := adminHandlers.database.First(&site, "id = ?", siteID).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSite})
		return
	}
	var feedbacks []model.Feedback
	if err := adminHandlers.database.
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
		})
	}

	context.JSON(http.StatusOK, siteMessagesResponse{SiteID: site.ID, Messages: messageResponses})
}
