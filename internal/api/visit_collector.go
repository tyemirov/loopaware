package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	visitCollectionPath        = "/public/visits"
	visitPixelContentType      = "image/gif"
	visitPixelBody             = "\x47\x49\x46\x38\x39\x61\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xff\xff\xff\x21\xf9\x04\x01\x00\x00\x00\x00\x2c\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02\x4c\x01\x00\x3b"
	visitHeaderVisitorID       = "X-Visitor-Id"
	visitQuerySiteID           = "site_id"
	visitQueryURL              = "url"
	visitQueryVisitorID        = "visitor_id"
	visitQueryReferrer         = "referrer"
	visitQueryScreenResolution = "screen_resolution"
	visitQueryViewport         = "viewport"
	visitQueryTimezone         = "timezone"
	visitQueryLocale           = "locale"
	visitQueryPageTitle        = "page_title"
	visitGeoSourceCloudflare   = "cloudflare"
	visitGeoSourceVercel       = "vercel"
	visitGeoSourceCloudfront   = "cloudfront"
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
	localeValue := strings.TrimSpace(context.Query(visitQueryLocale))
	if localeValue == "" {
		localeValue = primaryAcceptedLanguage(context.GetHeader("Accept-Language"))
	}
	edgeGeo := readVisitEdgeGeo(context)
	input := model.SiteVisitInput{
		SiteID:           site.ID,
		URL:              rawURL,
		VisitorID:        visitorID,
		IP:               context.ClientIP(),
		UserAgent:        userAgentValue,
		Referrer:         referrerValue,
		IsBot:            isLikelyBotUserAgent(userAgentValue),
		ScreenResolution: strings.TrimSpace(context.Query(visitQueryScreenResolution)),
		Viewport:         strings.TrimSpace(context.Query(visitQueryViewport)),
		Timezone:         strings.TrimSpace(context.Query(visitQueryTimezone)),
		Locale:           localeValue,
		GeoSource:        edgeGeo.Source,
		GeoCountry:       edgeGeo.Country,
		GeoRegion:        edgeGeo.Region,
		GeoCity:          edgeGeo.City,
		GeoLatitude:      edgeGeo.Latitude,
		GeoLongitude:     edgeGeo.Longitude,
		PageTitle:        strings.TrimSpace(context.Query(visitQueryPageTitle)),
		Occurred:         time.Now().UTC(),
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

type visitEdgeGeo struct {
	Source    string
	Country   string
	Region    string
	City      string
	Latitude  float64
	Longitude float64
}

func readVisitEdgeGeo(context *gin.Context) visitEdgeGeo {
	if edgeGeo, ok := readCloudflareVisitEdgeGeo(context); ok {
		return edgeGeo
	}
	if edgeGeo, ok := readVercelVisitEdgeGeo(context); ok {
		return edgeGeo
	}
	if edgeGeo, ok := readCloudfrontVisitEdgeGeo(context); ok {
		return edgeGeo
	}
	return visitEdgeGeo{}
}

func readCloudflareVisitEdgeGeo(context *gin.Context) (visitEdgeGeo, bool) {
	return buildVisitEdgeGeo(
		visitGeoSourceCloudflare,
		context.GetHeader("CF-IPCountry"),
		firstHeaderValue(context, "CF-Region-Code", "CF-Region"),
		context.GetHeader("CF-IPCity"),
		context.GetHeader("CF-IPLatitude"),
		context.GetHeader("CF-IPLongitude"),
	)
}

func readVercelVisitEdgeGeo(context *gin.Context) (visitEdgeGeo, bool) {
	return buildVisitEdgeGeo(
		visitGeoSourceVercel,
		context.GetHeader("X-Vercel-IP-Country"),
		context.GetHeader("X-Vercel-IP-Country-Region"),
		context.GetHeader("X-Vercel-IP-City"),
		context.GetHeader("X-Vercel-IP-Latitude"),
		context.GetHeader("X-Vercel-IP-Longitude"),
	)
}

func readCloudfrontVisitEdgeGeo(context *gin.Context) (visitEdgeGeo, bool) {
	return buildVisitEdgeGeo(
		visitGeoSourceCloudfront,
		context.GetHeader("CloudFront-Viewer-Country"),
		"",
		"",
		"",
		"",
	)
}

func buildVisitEdgeGeo(source string, rawCountry string, rawRegion string, rawCity string, rawLatitude string, rawLongitude string) (visitEdgeGeo, bool) {
	country := normalizeVisitGeoCountry(rawCountry)
	region := normalizeVisitGeoText(rawRegion)
	city := normalizeVisitGeoText(rawCity)
	latitude, latitudeOK := parseVisitGeoCoordinate(rawLatitude, -90, 90)
	longitude, longitudeOK := parseVisitGeoCoordinate(rawLongitude, -180, 180)
	if !latitudeOK || !longitudeOK {
		latitude = 0
		longitude = 0
	}
	if country == "" && region == "" && city == "" && (latitude == 0 && longitude == 0) {
		return visitEdgeGeo{}, false
	}
	return visitEdgeGeo{
		Source:    source,
		Country:   country,
		Region:    region,
		City:      city,
		Latitude:  latitude,
		Longitude: longitude,
	}, true
}

func firstHeaderValue(context *gin.Context, headerNames ...string) string {
	for _, headerName := range headerNames {
		value := strings.TrimSpace(context.GetHeader(headerName))
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeVisitGeoCountry(rawCountry string) string {
	country := strings.ToUpper(strings.TrimSpace(rawCountry))
	if country == "" || country == "XX" || country == "T1" || country == "A1" || country == "A2" {
		return ""
	}
	if len(country) != 2 && len(country) != 3 {
		return ""
	}
	for _, character := range country {
		if character < 'A' || character > 'Z' {
			return ""
		}
	}
	return country
}

func normalizeVisitGeoText(rawValue string) string {
	trimmedValue := strings.TrimSpace(rawValue)
	if trimmedValue == "" {
		return ""
	}
	return strings.Map(func(character rune) rune {
		if character < 0x20 || character == 0x7f {
			return -1
		}
		return character
	}, trimmedValue)
}

func parseVisitGeoCoordinate(rawValue string, minimum float64, maximum float64) (float64, bool) {
	trimmedValue := strings.TrimSpace(rawValue)
	if trimmedValue == "" {
		return 0, false
	}
	value, parseErr := strconv.ParseFloat(trimmedValue, 64)
	if parseErr != nil || value < minimum || value > maximum {
		return 0, false
	}
	return value, true
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

func primaryAcceptedLanguage(rawHeader string) string {
	parts := strings.Split(rawHeader, ",")
	if len(parts) == 0 {
		return ""
	}
	primary := strings.TrimSpace(parts[0])
	if primary == "" {
		return ""
	}
	languageTag := strings.TrimSpace(strings.Split(primary, ";")[0])
	return languageTag
}
