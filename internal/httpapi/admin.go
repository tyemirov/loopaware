package httpapi

import (
	"errors"
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
	jsonKeyError      = "error"
	jsonKeyEmail      = "email"
	jsonKeyName       = "name"
	jsonKeyIsAdmin    = "is_admin"
	jsonKeyPictureURL = "picture_url"

	errorValueInvalidJSON      = "invalid_json"
	errorValueMissingFields    = "missing_fields"
	errorValueSaveFailed       = "save_failed"
	errorValueMissingSite      = "missing_site"
	errorValueUnknownSite      = "unknown_site"
	errorValueQueryFailed      = "query_failed"
	errorValueNotAuthorized    = "not_authorized"
	errorValueInvalidOwner     = "invalid_owner"
	errorValueNothingToUpdate  = "nothing_to_update"
	errorValueInvalidOperation = "invalid_operation"

	widgetScriptTemplate = "<script src=\"%s/widget.js?site_id=%s\"></script>"
)

type SiteHandlers struct {
	database      *gorm.DB
	logger        *zap.Logger
	widgetBaseURL string
}

func NewSiteHandlers(database *gorm.DB, logger *zap.Logger, widgetBaseURL string) *SiteHandlers {
	return &SiteHandlers{
		database:      database,
		logger:        logger,
		widgetBaseURL: normalizeWidgetBaseURL(widgetBaseURL),
	}
}

type createSiteRequest struct {
	Name          string `json:"name"`
	AllowedOrigin string `json:"allowed_origin"`
	OwnerEmail    string `json:"owner_email"`
}

type updateSiteRequest struct {
	Name          *string `json:"name"`
	AllowedOrigin *string `json:"allowed_origin"`
	OwnerEmail    *string `json:"owner_email"`
}

type siteResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AllowedOrigin string `json:"allowed_origin"`
	OwnerEmail    string `json:"owner_email"`
	Widget        string `json:"widget"`
	CreatedAt     int64  `json:"created_at"`
}

type listSitesResponse struct {
	Sites []siteResponse `json:"sites"`
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

func (handlers *SiteHandlers) CurrentUser(context *gin.Context) {
	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return
	}

	context.JSON(http.StatusOK, gin.H{
		jsonKeyEmail:      currentUser.Email,
		jsonKeyName:       currentUser.Name,
		jsonKeyIsAdmin:    currentUser.IsAdmin,
		jsonKeyPictureURL: currentUser.PictureURL,
	})
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
	desiredOwnerEmail := strings.ToLower(strings.TrimSpace(payload.OwnerEmail))
	currentUserEmail := strings.ToLower(strings.TrimSpace(currentUser.Email))

	if !currentUser.IsAdmin {
		if desiredOwnerEmail != "" && !strings.EqualFold(desiredOwnerEmail, currentUserEmail) {
			context.JSON(http.StatusForbidden, gin.H{jsonKeyError: errorValueInvalidOperation})
			return
		}
		desiredOwnerEmail = currentUserEmail
	}

	if payload.Name == "" || payload.AllowedOrigin == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingFields})
		return
	}

	if desiredOwnerEmail == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidOwner})
		return
	}

	site := model.Site{
		ID:            storage.NewID(),
		Name:          payload.Name,
		AllowedOrigin: payload.AllowedOrigin,
		OwnerEmail:    desiredOwnerEmail,
	}

	if err := handlers.database.Create(&site).Error; err != nil {
		handlers.logger.Warn("create_site", zap.Error(err))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}

	context.JSON(http.StatusOK, handlers.toSiteResponse(site))
}

func (handlers *SiteHandlers) ListSites(context *gin.Context) {
	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return
	}

	var sites []model.Site

	query := handlers.database.Model(&model.Site{})
	if !currentUser.IsAdmin {
		query = query.Where("owner_email = ?", strings.ToLower(strings.TrimSpace(currentUser.Email)))
	}

	if err := query.Order("created_at desc").Find(&sites).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}

	responses := make([]siteResponse, 0, len(sites))
	for _, site := range sites {
		responses = append(responses, handlers.toSiteResponse(site))
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

	if payload.Name == nil && payload.AllowedOrigin == nil && payload.OwnerEmail == nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueNothingToUpdate})
		return
	}

	var site model.Site
	if err := handlers.database.First(&site, "id = ?", siteIdentifier).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSite})
		return
	}

	if !canManageSite(currentUser, site) {
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

	if payload.AllowedOrigin != nil {
		trimmed := strings.TrimSpace(*payload.AllowedOrigin)
		if trimmed == "" {
			context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingFields})
			return
		}
		site.AllowedOrigin = trimmed
	}

	if payload.OwnerEmail != nil {
		if !currentUser.IsAdmin {
			context.JSON(http.StatusForbidden, gin.H{jsonKeyError: errorValueInvalidOperation})
			return
		}
		trimmed := strings.ToLower(strings.TrimSpace(*payload.OwnerEmail))
		if trimmed == "" {
			context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidOwner})
			return
		}
		site.OwnerEmail = trimmed
	}

	if err := handlers.database.Save(&site).Error; err != nil {
		handlers.logger.Warn("update_site", zap.Error(err))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}

	context.JSON(http.StatusOK, handlers.toSiteResponse(site))
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

	if !canManageSite(currentUser, site) {
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
		})
	}

	context.JSON(http.StatusOK, siteMessagesResponse{SiteID: site.ID, Messages: messageResponses})
}

func (handlers *SiteHandlers) toSiteResponse(site model.Site) siteResponse {
	widgetBase := handlers.widgetBaseURL
	if widgetBase == "" {
		widgetBase = normalizeWidgetBaseURL(site.AllowedOrigin)
	}

	return siteResponse{
		ID:            site.ID,
		Name:          site.Name,
		AllowedOrigin: site.AllowedOrigin,
		OwnerEmail:    site.OwnerEmail,
		Widget:        fmt.Sprintf(widgetScriptTemplate, widgetBase, site.ID),
		CreatedAt:     site.CreatedAt.UTC().Unix(),
	}
}

func normalizeWidgetBaseURL(value string) string {
	trimmed := strings.TrimSpace(value)
	return strings.TrimRight(trimmed, "/")
}

func canManageSite(currentUser *CurrentUser, site model.Site) bool {
	if currentUser.IsAdmin {
		return true
	}
	return strings.EqualFold(site.OwnerEmail, strings.TrimSpace(currentUser.Email))
}
