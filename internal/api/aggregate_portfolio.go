package api

import (
	"context"
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
	detailedSites := make([]model.Site, 0, len(sites))
	aggregateIDs := make([]string, 0, len(sites))
	for _, site := range sites {
		if site.TrafficProfile == model.TrafficProfileAggregate {
			aggregateIDs = append(aggregateIDs, site.ID)
		} else {
			detailedSites = append(detailedSites, site)
		}
	}
	detailedRows, err := detailedPortfolioSiteRows(ctx, database, detailedSites, start)
	if err != nil {
		return portfolioTrafficReportData{}, err
	}
	var rows []struct {
		SiteID, Day    string
		Count          int64
		UniqueVisitors int64
	}
	err = database.WithContext(ctx).Raw(`SELECT site_id, DATE(occurred_at) AS day, COUNT(*) AS count,
		COUNT(DISTINCT CASE WHEN visitor_id <> '' THEN visitor_id END) AS unique_visitors
		FROM site_visits WHERE site_id IN ? AND occurred_at >= ? AND is_bot = false GROUP BY site_id, DATE(occurred_at)
		UNION ALL SELECT site_id, date AS day, count, 0 AS unique_visitors FROM site_visit_counts WHERE site_id IN ? AND date >= ?`,
		portfolioSiteIDs(detailedSites), start, aggregateIDs, start.Format(visitTrendDateFormat)).Scan(&rows).Error
	if err != nil {
		return portfolioTrafficReportData{}, err
	}
	report := portfolioTrafficReportData{CountsOnly: true, TimeResolution: portfolioTrafficTimeResolutionUTCDay, WindowDays: days, SiteCount: len(sites)}
	for _, row := range detailedRows {
		report.UniqueVisitors += row.UniqueVisitorCount
	}
	bySite := make(map[string]int64)
	byDate := make(map[string]int64)
	uniqueVisitorsByDate := make(map[string]int64)
	for _, row := range rows {
		bySite[row.SiteID] += row.Count
		byDate[row.Day] += row.Count
		uniqueVisitorsByDate[row.Day] += row.UniqueVisitors
		report.PageViews += row.Count
	}
	for index := 0; index < days; index++ {
		date := start.AddDate(0, 0, index).Format(visitTrendDateFormat)
		report.Trend = append(report.Trend, VisitTrendPoint{Date: date, PageViews: byDate[date], UniqueVisitors: uniqueVisitorsByDate[date]})
	}
	report.Sites = append(report.Sites, detailedRows...)
	for _, site := range sites {
		if site.TrafficProfile == model.TrafficProfileAggregate {
			report.Sites = append(report.Sites, PortfolioTrafficSiteRecord{SiteID: site.ID, SiteName: site.Name, VisitCount: bySite[site.ID], CountsOnly: true})
		}
	}
	sort.Slice(report.Sites, func(left, right int) bool {
		if report.Sites[left].VisitCount == report.Sites[right].VisitCount {
			return report.Sites[left].SiteName < report.Sites[right].SiteName
		}
		return report.Sites[left].VisitCount > report.Sites[right].VisitCount
	})
	return report, nil
}
