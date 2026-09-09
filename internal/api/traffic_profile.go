package api

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const errorValueInvalidTrafficProfile = "invalid_traffic_profile"

func validTrafficProfile(profile model.TrafficProfile) bool {
	return profile == model.TrafficProfileDetailed || profile == model.TrafficProfileAggregate
}

func (response siteResponse) MarshalJSON() ([]byte, error) {
	type siteRepresentation siteResponse
	var uniqueVisitors *int64
	if response.TrafficProfile == model.TrafficProfileDetailed {
		uniqueVisitors = &response.UniqueVisitorCount
	}
	return json.Marshal(struct {
		siteRepresentation
		UniqueVisitorCount *int64 `json:"unique_visitor_count"`
	}{siteRepresentation(response), uniqueVisitors})
}

func aggregateVisitRows(database *gorm.DB, siteID string, start time.Time) ([]model.SiteVisitCount, error) {
	query := database.Where("site_id = ?", siteID)
	if !start.IsZero() {
		query = query.Where("date >= ?", start.UTC().Format(visitTrendDateFormat))
	}
	rows := make([]model.SiteVisitCount, 0)
	err := query.Order("date ASC").Find(&rows).Error
	return rows, err
}

func (handlers *SiteHandlers) aggregateStats(context *gin.Context, site model.Site, interval trafficInterval) {
	rows, err := aggregateVisitRows(handlers.database.WithContext(context.Request.Context()), site.ID, aggregateIntervalStart(interval))
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	var count int64
	for _, row := range rows {
		count += row.Count
	}
	context.JSON(http.StatusOK, gin.H{
		"site_id": site.ID, "interval": interval.Value(), "traffic_profile": site.TrafficProfile,
		"metric": "accepted_requests", "time_resolution": "utc_day", "visit_count": count,
		"unique_visitor_count": nil, "top_pages": nil, "recent_visits": nil,
	})
}

func (handlers *SiteHandlers) aggregateTrend(context *gin.Context, site model.Site) {
	interval, err := parseTrafficInterval(context.Query("interval"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidInterval})
		return
	}
	start := aggregateIntervalStart(interval)
	if days := context.Query("days"); days != "" && context.Query("interval") == "" {
		parsedDays, parseErr := parseVisitTrendDays(days)
		if parseErr != nil {
			context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidDays})
			return
		}
		start = aggregateCalendarStart(parsedDays)
	}
	rows, err := aggregateVisitRows(handlers.database.WithContext(context.Request.Context()), site.ID, start)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	points := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		points = append(points, gin.H{"date": row.Date, "page_views": row.Count, "unique_visitors": nil})
	}
	context.JSON(http.StatusOK, gin.H{"site_id": site.ID, "interval": interval.Value(), "days": len(points), "traffic_profile": site.TrafficProfile, "metric": "accepted_requests", "time_resolution": "utc_day", "trend": points})
}

func (handlers *SiteHandlers) aggregateExport(context *gin.Context, site model.Site) {
	interval, parseErr := parseTrafficInterval(context.Query("interval"))
	if parseErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidInterval})
		return
	}
	rows, err := aggregateVisitRows(handlers.database.WithContext(context.Request.Context()), site.ID, aggregateIntervalStart(interval))
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	context.Header("Content-Type", "text/csv; charset=utf-8")
	context.Header("Content-Disposition", `attachment; filename="daily-counts.csv"`)
	context.Header("Cache-Control", "no-store")
	writer := csv.NewWriter(context.Writer)
	if err := writer.Write([]string{"site_id", "utc_date", "accepted_requests"}); err != nil {
		context.Abort()
		return
	}
	for _, row := range rows {
		if err := writer.Write([]string{row.SiteID, row.Date, strconv.FormatInt(row.Count, 10)}); err != nil {
			context.Abort()
			return
		}
	}
	writer.Flush()
	if writer.Error() != nil {
		context.Abort()
	}
}

func requireDetailedTraffic(context *gin.Context, site model.Site) bool {
	if site.TrafficProfile == model.TrafficProfileDetailed {
		return true
	}
	context.JSON(http.StatusConflict, gin.H{jsonKeyError: "metric_unavailable", "traffic_profile": site.TrafficProfile})
	return false
}

func aggregateCalendarStart(days int) time.Time {
	return time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -(days - 1))
}

func aggregateIntervalStart(interval trafficInterval) time.Time {
	if interval.IsAll() {
		return time.Time{}
	}
	return aggregateCalendarStart(interval.Days())
}
