package api

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"gorm.io/gorm"
)

func hasAggregateSite(sites []model.Site) bool {
	for _, site := range sites {
		if site.TrafficProfile == model.TrafficProfileAggregate {
			return true
		}
	}
	return false
}

func buildAggregatePortfolio(ctx context.Context, database *gorm.DB, sites []model.Site, days int) (portfolioTrafficReportData, error) {
	start := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -(days - 1))
	identifiers := portfolioSiteIDs(sites)
	var rows []struct {
		SiteID, Day string
		Count       int64
	}
	err := database.WithContext(ctx).Raw(`SELECT site_id, DATE(occurred_at) AS day, COUNT(*) AS count
		FROM site_visits WHERE site_id IN ? AND occurred_at >= ? AND is_bot = false GROUP BY site_id, DATE(occurred_at)
		UNION ALL SELECT site_id, date AS day, count FROM site_visit_counts WHERE site_id IN ? AND date >= ?`,
		identifiers, start, identifiers, start.Format(visitTrendDateFormat)).Scan(&rows).Error
	if err != nil {
		return portfolioTrafficReportData{}, err
	}
	report := portfolioTrafficReportData{CountsOnly: true, WindowDays: days, SiteCount: len(sites)}
	bySite := make(map[string]int64)
	byDate := make(map[string]int64)
	for _, row := range rows {
		bySite[row.SiteID] += row.Count
		byDate[row.Day] += row.Count
		report.PageViews += row.Count
	}
	for index := 0; index < days; index++ {
		date := start.AddDate(0, 0, index).Format(visitTrendDateFormat)
		report.Trend = append(report.Trend, VisitTrendPoint{Date: date, PageViews: byDate[date]})
	}
	for _, site := range sites {
		report.Sites = append(report.Sites, PortfolioTrafficSiteRecord{SiteID: site.ID, SiteName: site.Name, VisitCount: bySite[site.ID], CountsOnly: true})
	}
	sort.Slice(report.Sites, func(left, right int) bool {
		if report.Sites[left].VisitCount == report.Sites[right].VisitCount {
			return report.Sites[left].SiteName < report.Sites[right].SiteName
		}
		return report.Sites[left].VisitCount > report.Sites[right].VisitCount
	})
	return report, nil
}

// MarshalJSON makes unavailable portfolio visitor totals explicit.
func (response PortfolioTrafficReportResponse) MarshalJSON() ([]byte, error) {
	type representation PortfolioTrafficReportResponse
	if !response.CountsOnly {
		return json.Marshal(representation(response))
	}
	points := make([]map[string]any, 0, len(response.Trend))
	for _, point := range response.Trend {
		points = append(points, map[string]any{"date": point.Date, "page_views": point.PageViews, "unique_visitors": nil})
	}
	return json.Marshal(struct {
		representation
		UniqueVisitors *int64           `json:"unique_visitor_count"`
		Trend          []map[string]any `json:"trend"`
		TimeResolution string           `json:"time_resolution"`
	}{representation(response), nil, points, "utc_day"})
}

// MarshalJSON marks visitor metrics unavailable in a portfolio of daily totals.
func (response PortfolioTrafficSiteRecord) MarshalJSON() ([]byte, error) {
	type representation PortfolioTrafficSiteRecord
	if !response.CountsOnly {
		return json.Marshal(representation(response))
	}
	return json.Marshal(struct {
		representation
		UniqueVisitors *int64 `json:"unique_visitor_count"`
	}{representation(response), nil})
}
