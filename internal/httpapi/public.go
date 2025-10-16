package httpapi

import (
	"bytes"
	"fmt"
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
}

func NewPublicHandlers(database *gorm.DB, logger *zap.Logger) *PublicHandlers {
	return &PublicHandlers{
		database:                  database,
		logger:                    logger,
		rateWindow:                30 * time.Second,
		maxRequestsPerIPPerWindow: 6,
		rateCountersByIP:          make(map[string]int),
	}
}

type createFeedbackRequest struct {
	SiteID      string `json:"site_id"`
	ContactInfo string `json:"contact"`
	MessageBody string `json:"message"`
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
	if site.AllowedOrigin != "" {
		if originHeader != "" && originHeader != site.AllowedOrigin {
			context.JSON(403, gin.H{"error": "origin_forbidden"})
			return
		}
		if originHeader == "" && refererHeader != "" && !strings.HasPrefix(refererHeader, site.AllowedOrigin) {
			context.JSON(403, gin.H{"error": "origin_forbidden"})
			return
		}
	}

	feedback := model.Feedback{
		ID:        storage.NewID(),
		SiteID:    site.ID,
		Contact:   truncate(payload.ContactInfo, 320),
		Message:   truncate(payload.MessageBody, 4000),
		IP:        clientIP,
		UserAgent: truncate(context.Request.UserAgent(), 400),
	}

	if err := h.database.Create(&feedback).Error; err != nil {
		h.logger.Warn("save_feedback", zap.Error(err))
		context.JSON(500, gin.H{"error": "save_failed"})
		return
	}

	context.JSON(200, gin.H{"status": "ok"})
}

func (h *PublicHandlers) isRateLimited(ip string) bool {
	nowBucket := time.Now().Unix() / int64(h.rateWindow.Seconds())
	key := fmt.Sprintf("%s:%d", ip, nowBucket)

	h.rateCountersMutex.Lock()
	defer h.rateCountersMutex.Unlock()

	h.rateCountersByIP[key]++
	if h.rateCountersByIP[key] > h.maxRequestsPerIPPerWindow {
		return true
	}
	return false
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
	if err := h.database.First(&site, "id = ?", siteID).Error; err != nil {
		context.String(404, "/* unknown site */")
		return
	}

	script, tplErr := renderWidgetTemplate(site.ID)
	if tplErr != nil {
		context.String(500, "/* render error */")
		return
	}

	context.Data(200, "application/javascript; charset=utf-8", []byte(script))
}

func truncate(input string, max int) string {
	if len(input) <= max {
		return input
	}
	return input[:max]
}

func renderWidgetTemplate(siteID string) (string, error) {
	var buffer bytes.Buffer
	executeErr := widgetJavaScriptTemplate.Execute(&buffer, map[string]string{
		"SiteID": siteID,
	})
	if executeErr != nil {
		return "", fmt.Errorf("render widget template: %w", executeErr)
	}
	return buffer.String(), nil
}
