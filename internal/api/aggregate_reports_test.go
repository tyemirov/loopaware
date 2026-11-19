package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAggregateReportsUseCountsAndDeclareUnavailableMetrics(testingT *testing.T) {
	harness := buildTrafficReportHarness(testingT, true)
	insertTrafficReportSite(testingT, harness.database)
	require.NoError(testingT, harness.database.Model(&model.Site{}).Where("id = ?", testTrafficReportSiteID).Update("traffic_profile", model.TrafficProfileAggregate).Error)
	require.NoError(testingT, harness.database.Create(&model.SiteVisitCount{SiteID: testTrafficReportSiteID, Date: time.Now().UTC().Format(visitTrendDateFormat), Count: 3}).Error)
	other := model.Site{ID: storage.NewID(), Name: "Detailed", AllowedOrigin: "https://other.example", OwnerEmail: testTrafficReportOwnerEmail}
	require.NoError(testingT, harness.database.Create(&other).Error)
	visit, err := model.NewSiteVisit(model.SiteVisitInput{SiteID: other.ID, URL: other.AllowedOrigin, Occurred: time.Now().UTC()})
	require.NoError(testingT, err)
	require.NoError(testingT, harness.database.Create(&visit).Error)
	harness.handlers.statsProvider = NewDatabaseSiteStatisticsProvider(harness.database)
	router := gin.New()
	router.Use(func(context *gin.Context) { setTrafficReportUser(context, testTrafficReportOwnerEmail) })
	router.GET("/api/reports/traffic/portfolio", harness.handlers.GetPortfolioReport)
	router.POST("/api/sites/:id/traffic-report-schedule/test", harness.handlers.SendTestReport)
	server := httptest.NewServer(router)
	defer server.Close()
	response, err := server.Client().Get(server.URL + "/api/reports/traffic/portfolio?days=30")
	require.NoError(testingT, err)
	defer response.Body.Close()
	require.Equal(testingT, http.StatusOK, response.StatusCode)
	var report map[string]any
	require.NoError(testingT, json.NewDecoder(response.Body).Decode(&report))
	require.Equal(testingT, float64(4), report["visit_count"])
	require.Nil(testingT, report["unique_visitor_count"])
	response, err = server.Client().Post(server.URL+testTrafficReportTestPath, "application/json", nil)
	require.NoError(testingT, err)
	response.Body.Close()
	require.Equal(testingT, http.StatusOK, response.StatusCode)
	require.Len(testingT, harness.emailSender.calls, 1)
	require.Contains(testingT, harness.emailSender.calls[0].message, "Accepted requests: 3")
	require.Contains(testingT, harness.emailSender.calls[0].message, "Unique visitors: unavailable")
	require.NotContains(testingT, harness.emailSender.calls[0].message, "Unique visitors: 0")
}

func TestMixedPortfolioPreservesDetailedVisitorCounts(testingT *testing.T) {
	harness := buildTrafficReportHarness(testingT, true)
	insertTrafficReportSite(testingT, harness.database)
	require.NoError(testingT, harness.database.Model(&model.Site{}).Where("id = ?", testTrafficReportSiteID).Update("traffic_profile", model.TrafficProfileAggregate).Error)
	detailed := model.Site{ID: storage.NewID(), Name: "Detailed", AllowedOrigin: "https://detailed.example", OwnerEmail: testTrafficReportOwnerEmail}
	empty := model.Site{ID: storage.NewID(), Name: "Empty", AllowedOrigin: "https://empty.example", OwnerEmail: testTrafficReportOwnerEmail}
	require.NoError(testingT, harness.database.Create(&detailed).Error)
	require.NoError(testingT, harness.database.Create(&empty).Error)
	start := time.Now().UTC().Truncate(24 * time.Hour)
	require.NoError(testingT, harness.database.Create(&model.SiteVisitCount{SiteID: testTrafficReportSiteID, Date: start.Format(visitTrendDateFormat), Count: 9}).Error)
	for _, entry := range []struct {
		visitor string
		at      time.Time
		bot     bool
	}{
		{"11111111-1111-1111-1111-111111111111", start, false}, {"11111111-1111-1111-1111-111111111111", start.Add(time.Second), false}, {"", start, false},
		{"22222222-2222-2222-2222-222222222222", start, true}, {"33333333-3333-3333-3333-333333333333", start.Add(-time.Second), false},
		{"11111111-1111-1111-1111-111111111111", start.Add(-time.Second), false},
	} {
		visit, err := model.NewSiteVisit(model.SiteVisitInput{SiteID: detailed.ID, URL: detailed.AllowedOrigin, VisitorID: entry.visitor, Occurred: entry.at})
		require.NoError(testingT, err)
		visit.IsBot = entry.bot
		require.NoError(testingT, harness.database.Create(&visit).Error)
	}
	server := visiblePortfolioServer(testingT, harness)
	month := readVisiblePortfolioResponse(testingT, server, RoleUser, testVisiblePortfolioPath+"?scope=visible&days=30", http.StatusOK)
	var detailedRow map[string]any
	for _, item := range month["sites"].([]any) {
		row := item.(map[string]any)
		if row["site_id"] == detailed.ID {
			detailedRow = row
			require.Equal(testingT, float64(5), row["visit_count"])
			require.Equal(testingT, float64(2), row["unique_visitor_count"])
		}
	}
	require.NotNil(testingT, detailedRow)
	report := readVisiblePortfolioResponse(testingT, server, RoleUser, testVisiblePortfolioPath+"?scope=visible&days=1", http.StatusOK)
	require.Equal(testingT, float64(12), report["visit_count"])
	require.Nil(testingT, report["unique_visitor_count"])
	require.True(testingT, report["counts_only"].(bool))
	rows := report["sites"].([]any)
	require.Len(testingT, rows, 3)
	for _, item := range rows {
		row := item.(map[string]any)
		switch row["site_id"] {
		case detailed.ID:
			require.Equal(testingT, float64(3), row["visit_count"])
			require.Equal(testingT, float64(1), row["unique_visitor_count"])
			require.Equal(testingT, false, row["counts_only"])
		case empty.ID:
			require.Equal(testingT, float64(0), row["visit_count"])
			require.Equal(testingT, float64(0), row["unique_visitor_count"])
			require.Equal(testingT, false, row["counts_only"])
		default:
			require.Equal(testingT, testTrafficReportSiteID, row["site_id"])
			require.Equal(testingT, float64(9), row["visit_count"])
			require.Nil(testingT, row["unique_visitor_count"])
			require.Equal(testingT, true, row["counts_only"])
		}
	}
	points := report["trend"].([]any)
	require.Len(testingT, points, 1)
	require.Equal(testingT, float64(12), points[0].(map[string]any)["page_views"])
	require.Nil(testingT, points[0].(map[string]any)["unique_visitors"])
}
