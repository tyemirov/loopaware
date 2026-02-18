package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	visitCollectionPath   = "/public/visits"
	visitPixelContentType = "image/gif"
	visitPixelBody        = "\x47\x49\x46\x38\x39\x61\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xff\xff\xff\x21\xf9\x04\x01\x00\x00\x00\x00\x2c\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02\x4c\x01\x00\x3b"
	visitHeaderVisitorID  = "X-Visitor-Id"
	visitQuerySiteID      = "site_id"
	visitQueryURL         = "url"
	visitQueryVisitorID   = "visitor_id"
	visitQueryReferrer    = "referrer"
)

var visitBotUserAgentTokens = [...]string{
	"bot",
	"crawler",
	"crawl",
	"spider",
	"slurp",
	"bingpreview",
	"duckduckbot",
	"baiduspider",
	"yandexbot",
	"semrushbot",
	"ahrefsbot",
	"mj12bot",
	"facebookexternalhit",
	"telegrambot",
	"petalbot",
}

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
	queryReferrer := strings.TrimSpace(context.Query(visitQueryReferrer))
	referrerValue := refererHeader
	if referrerValue == "" {
		referrerValue = queryReferrer
	}
	rawURL := strings.TrimSpace(context.Query(visitQueryURL))
	allowedOrigins := mergedAllowedOrigins(site.AllowedOrigin, site.TrafficAllowedOrigins)
	if !isOriginAllowed(allowedOrigins, originHeader, refererHeader, rawURL) {
		context.String(http.StatusForbidden, "/* origin_forbidden */")
		return
	}
	if rawURL == "" && referrerValue != "" {
		rawURL = referrerValue
	}

	visitorID := strings.TrimSpace(context.Query(visitQueryVisitorID))
	if visitorID == "" {
		visitorID = strings.TrimSpace(context.GetHeader(visitHeaderVisitorID))
	}

	userAgentValue := context.Request.UserAgent()
	input := model.SiteVisitInput{
		SiteID:    site.ID,
		URL:       rawURL,
		VisitorID: visitorID,
		IP:        context.ClientIP(),
		UserAgent: userAgentValue,
		Referrer:  referrerValue,
		IsBot:     isLikelyBotUserAgent(userAgentValue),
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

func isLikelyBotUserAgent(userAgentValue string) bool {
	normalizedUserAgent := strings.ToLower(strings.TrimSpace(userAgentValue))
	if normalizedUserAgent == "" {
		return false
	}
	for _, userAgentToken := range visitBotUserAgentTokens {
		if strings.Contains(normalizedUserAgent, userAgentToken) {
			return true
		}
	}
	return false
}
