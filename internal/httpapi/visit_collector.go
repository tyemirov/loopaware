package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	visitCollectionPath   = "/api/visits"
	visitPixelContentType = "image/gif"
	visitPixelBody        = "\x47\x49\x46\x38\x39\x61\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xff\xff\xff\x21\xf9\x04\x01\x00\x00\x00\x00\x2c\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02\x4c\x01\x00\x3b"
	visitHeaderVisitorID  = "X-Visitor-Id"
	visitQuerySiteID      = "site_id"
	visitQueryURL         = "url"
	visitQueryVisitorID   = "visitor_id"
	visitQueryReferrer    = "referrer"
)

// CollectVisit handles pixel-style visit recording.
func (h *PublicHandlers) CollectVisit(context *gin.Context) {
	siteID := strings.TrimSpace(context.Query(visitQuerySiteID))
	if siteID == "" {
		context.String(http.StatusBadRequest, "missing site_id")
		return
	}

	var site model.Site
	if err := h.database.First(&site, "id = ?", siteID).Error; err != nil {
		context.String(http.StatusNotFound, "/* unknown site */")
		return
	}

	originHeader := strings.TrimSpace(context.GetHeader("Origin"))
	refererHeader := strings.TrimSpace(context.GetHeader("Referer"))
	rawURL := strings.TrimSpace(context.Query(visitQueryURL))
	if !isOriginAllowed(site.AllowedOrigin, originHeader, refererHeader, rawURL) {
		context.String(http.StatusForbidden, "/* origin_forbidden */")
		return
	}
	if rawURL == "" && refererHeader != "" {
		rawURL = refererHeader
	}

	visitorID := strings.TrimSpace(context.Query(visitQueryVisitorID))
	if visitorID == "" {
		visitorID = strings.TrimSpace(context.GetHeader(visitHeaderVisitorID))
	}

	input := model.SiteVisitInput{
		SiteID:    site.ID,
		URL:       rawURL,
		VisitorID: visitorID,
		IP:        context.ClientIP(),
		UserAgent: context.Request.UserAgent(),
		Referrer:  refererHeader,
		Occurred:  time.Now().UTC(),
	}

	visit, err := model.NewSiteVisit(input)
	if err != nil {
		if h.logger != nil {
			h.logger.Debug("visit_validation_failed", zap.Error(err))
		}
		if strings.Contains(err.Error(), "invalid_visit_id") {
			context.String(http.StatusBadRequest, "/* "+errorValueInvalidVisitorID+" */")
			return
		}
		context.String(http.StatusBadRequest, "/* "+errorValueInvalidURL+" */")
		return
	}

	if err := h.database.Create(&visit).Error; err != nil {
		if h.logger != nil {
			h.logger.Warn("visit_save_failed", zap.Error(err))
		}
		context.String(http.StatusInternalServerError, "/* save_failed */")
		return
	}

	context.Header("Content-Type", visitPixelContentType)
	context.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	context.Header("Pragma", "no-cache")
	context.Data(http.StatusOK, visitPixelContentType, []byte(visitPixelBody))
}
