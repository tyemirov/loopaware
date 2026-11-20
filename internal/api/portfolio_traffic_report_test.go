package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderPortfolioTrafficReportEmailOmitsAggregateTopPages(testingT *testing.T) {
	body, bodyErr := renderPortfolioTrafficReportEmailTemplate("body", portfolioTrafficReportEmailTemplateData{
		FrequencyLabel: "Weekly",
		ReportName:     "All sites traffic",
		WindowDays:     7,
		SiteCount:      2,
		PageViews:      52,
		UniqueVisitors: 14,
		Sites: []PortfolioTrafficSiteRecord{
			{SiteName: "Tyemirov.net", VisitCount: 40, UniqueVisitorCount: 12},
			{SiteName: "LoopAware", VisitCount: 12, UniqueVisitorCount: 2},
		},
	})

	require.NoError(testingT, bodyErr)
	require.Contains(testingT, body, "Sites: 2")
	require.Contains(testingT, body, "- Tyemirov.net: 40 views, 12 unique")
	require.NotContains(testingT, body, "Top pages:")
	require.NotContains(testingT, body, "/login")
}

func TestRenderPortfolioTrafficReportEmailIncludesRecordedUniqueCounts(testingT *testing.T) {
	body, err := renderPortfolioTrafficReportEmailTemplate("body", portfolioTrafficReportEmailTemplateData{
		CountsOnly:     true,
		FrequencyLabel: "Weekly",
		ReportName:     "Mixed report",
		WindowDays:     7,
		SiteCount:      2,
		PageViews:      15,
		UniqueVisitors: 2,
		Sites: []PortfolioTrafficSiteRecord{
			{SiteName: "Daily counts", VisitCount: 9, CountsOnly: true},
			{SiteName: "Detailed", VisitCount: 6, UniqueVisitorCount: 2},
		},
	})
	require.NoError(testingT, err)
	require.Contains(testingT, body, "Recorded page views and accepted requests: 15")
	require.Contains(testingT, body, "Time resolution: UTC calendar days")
	require.Contains(testingT, body, "Unique site visitors: 2")
	require.Contains(testingT, body, "- Daily counts: 9 recorded requests, 0 unique")
	require.Contains(testingT, body, "- Detailed: 6 views, 2 unique")
	require.NotContains(testingT, body, "unavailable")
}
