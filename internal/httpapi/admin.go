package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
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

func (adminHandlers *AdminHandlers) CreateSite(context *gin.Context) {
	var payload createSiteRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	payload.Name = strings.TrimSpace(payload.Name)
	payload.AllowedOrigin = strings.TrimSpace(payload.AllowedOrigin)
	if payload.Name == "" || payload.AllowedOrigin == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "missing_fields"})
		return
	}
	site := model.Site{
		ID:            storage.NewID(),
		Name:          payload.Name,
		AllowedOrigin: payload.AllowedOrigin,
	}
	if err := adminHandlers.database.Create(&site).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "save_failed"})
		return
	}
	context.JSON(http.StatusOK, gin.H{
		"id":             site.ID,
		"name":           site.Name,
		"allowed_origin": site.AllowedOrigin,
		"widget":         "<script src=\"" + site.AllowedOrigin + "/widget.js?site_id=" + site.ID + "\"></script>",
	})
}

func (adminHandlers *AdminHandlers) ListMessagesBySite(context *gin.Context) {
	siteID := strings.TrimSpace(context.Param("id"))
	if siteID == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "missing_site"})
		return
	}
	var site model.Site
	if err := adminHandlers.database.First(&site, "id = ?", siteID).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "unknown_site"})
		return
	}
	type row struct {
		ID        string `json:"id"`
		Contact   string `json:"contact"`
		Message   string `json:"message"`
		IP        string `json:"ip"`
		UserAgent string `json:"user_agent"`
		CreatedAt int64  `json:"created_at"`
	}
	var out []row
	if err := adminHandlers.database.
		Table("feedbacks").
		Select("id, contact, message, ip, user_agent, (EXTRACT(EPOCH FROM created_at)::bigint) AS created_at").
		Where("site_id = ?", site.ID).
		Order("created_at desc").
		Scan(&out).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "query_failed"})
		return
	}
	context.JSON(http.StatusOK, gin.H{"site_id": site.ID, "messages": out})
}
