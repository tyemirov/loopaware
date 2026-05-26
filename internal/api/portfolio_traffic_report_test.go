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
