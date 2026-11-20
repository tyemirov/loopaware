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

func TestAggregateReportsUseNumericPortfolioAndPreserveSingleSiteUnavailableMetrics(testingT *testing.T) {
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
	require.Equal(testingT, float64(0), report["unique_visitor_count"])
	require.Equal(testingT, "utc_day", report["time_resolution"])
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
	secondDetailed := model.Site{ID: storage.NewID(), Name: "Second detailed", AllowedOrigin: "https://second-detailed.example", OwnerEmail: testTrafficReportOwnerEmail}
	empty := model.Site{ID: storage.NewID(), Name: "Empty", AllowedOrigin: "https://empty.example", OwnerEmail: testTrafficReportOwnerEmail}
	require.NoError(testingT, harness.database.Create(&detailed).Error)
	require.NoError(testingT, harness.database.Create(&secondDetailed).Error)
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
	for _, occurred := range []time.Time{start, start.Add(-time.Second)} {
		visit, err := model.NewSiteVisit(model.SiteVisitInput{SiteID: secondDetailed.ID, URL: secondDetailed.AllowedOrigin, VisitorID: "11111111-1111-1111-1111-111111111111", Occurred: occurred})
		require.NoError(testingT, err)
		require.NoError(testingT, harness.database.Create(&visit).Error)
	}
	server := visiblePortfolioServer(testingT, harness)
	month := readVisiblePortfolioResponse(testingT, server, RoleUser, testVisiblePortfolioPath+"?scope=visible&days=30", http.StatusOK)
	require.Equal(testingT, float64(16), month["visit_count"])
	require.Equal(testingT, float64(3), month["unique_visitor_count"])
	require.Equal(testingT, "utc_day", month["time_resolution"])
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
	dailyUniqueTotal := float64(0)
	for _, item := range month["trend"].([]any) {
		point := item.(map[string]any)
		dailyUniqueTotal += point["unique_visitors"].(float64)
		switch point["date"] {
		case start.Format(visitTrendDateFormat):
			require.Equal(testingT, float64(13), point["page_views"])
			require.Equal(testingT, float64(2), point["unique_visitors"])
		case start.AddDate(0, 0, -1).Format(visitTrendDateFormat):
			require.Equal(testingT, float64(3), point["page_views"])
			require.Equal(testingT, float64(3), point["unique_visitors"])
		default:
			require.Equal(testingT, float64(0), point["page_views"])
			require.Equal(testingT, float64(0), point["unique_visitors"])
		}
	}
	require.Equal(testingT, float64(5), dailyUniqueTotal)
	require.NotEqual(testingT, month["unique_visitor_count"], dailyUniqueTotal)
	report := readVisiblePortfolioResponse(testingT, server, RoleUser, testVisiblePortfolioPath+"?scope=visible&days=1", http.StatusOK)
	require.Equal(testingT, float64(13), report["visit_count"])
	require.Equal(testingT, float64(2), report["unique_visitor_count"])
	require.True(testingT, report["counts_only"].(bool))
	require.Equal(testingT, "utc_day", report["time_resolution"])
	rows := report["sites"].([]any)
	require.Len(testingT, rows, 4)
	for _, item := range rows {
		row := item.(map[string]any)
		switch row["site_id"] {
		case detailed.ID:
			require.Equal(testingT, float64(3), row["visit_count"])
			require.Equal(testingT, float64(1), row["unique_visitor_count"])
			require.Equal(testingT, false, row["counts_only"])
		case secondDetailed.ID:
			require.Equal(testingT, float64(1), row["visit_count"])
			require.Equal(testingT, float64(1), row["unique_visitor_count"])
			require.Equal(testingT, false, row["counts_only"])
		case empty.ID:
			require.Equal(testingT, float64(0), row["visit_count"])
			require.Equal(testingT, float64(0), row["unique_visitor_count"])
			require.Equal(testingT, false, row["counts_only"])
		default:
			require.Equal(testingT, testTrafficReportSiteID, row["site_id"])
			require.Equal(testingT, float64(9), row["visit_count"])
			require.Equal(testingT, float64(0), row["unique_visitor_count"])
			require.Equal(testingT, true, row["counts_only"])
		}
	}
	points := report["trend"].([]any)
	require.Len(testingT, points, 1)
	require.Equal(testingT, float64(13), points[0].(map[string]any)["page_views"])
	require.Equal(testingT, float64(2), points[0].(map[string]any)["unique_visitors"])
	library := readVisiblePortfolioResponse(testingT, server, RoleUser, testVisiblePortfolioPath+"/reports", http.StatusOK)
	for _, item := range library["available_sites"].([]any) {
		row := item.(map[string]any)
		switch row["site_id"] {
		case detailed.ID:
			require.Equal(testingT, float64(2), row["unique_visitor_count"])
		case secondDetailed.ID:
			require.Equal(testingT, float64(1), row["unique_visitor_count"])
		default:
			require.Equal(testingT, float64(0), row["unique_visitor_count"])
		}
	}
	response, err := server.Client().Post(server.URL+testVisiblePortfolioPath+"/schedule/test", "application/json", nil)
	require.NoError(testingT, err)
	response.Body.Close()
	require.Equal(testingT, http.StatusOK, response.StatusCode)
	require.Len(testingT, harness.emailSender.calls, 1)
	message := harness.emailSender.calls[0].message
	require.Contains(testingT, message, "Recorded page views and accepted requests: 16")
	require.Contains(testingT, message, "Unique site visitors: 3")
	require.Contains(testingT, message, "Time resolution: UTC calendar days")
	require.Contains(testingT, message, "- Detailed: 5 views, 2 unique")
	require.Contains(testingT, message, "- Second detailed: 2 views, 1 unique")
	require.Contains(testingT, message, "9 recorded requests, 0 unique")
	require.NotContains(testingT, message, "unavailable")
}

func TestAggregateReportsUseZeroUniqueContributions(testingT *testing.T) {
	for _, scenario := range []struct {
		name  string
		count int64
	}{
		{name: "empty", count: 0},
		{name: "accepted requests", count: 3},
	} {
		testingT.Run(scenario.name, func(testingT *testing.T) {
			count := scenario.count
			harness := buildTrafficReportHarness(testingT, true)
			insertTrafficReportSite(testingT, harness.database)
			require.NoError(testingT, harness.database.Model(&model.Site{}).Where("id = ?", testTrafficReportSiteID).Update("traffic_profile", model.TrafficProfileAggregate).Error)
			if count > 0 {
				require.NoError(testingT, harness.database.Create(&model.SiteVisitCount{SiteID: testTrafficReportSiteID, Date: time.Now().UTC().Format(visitTrendDateFormat), Count: count}).Error)
			}
			server := visiblePortfolioServer(testingT, harness)
			report := readVisiblePortfolioResponse(testingT, server, RoleUser, testVisiblePortfolioPath, http.StatusOK)
			require.Equal(testingT, float64(count), report["visit_count"])
			require.Equal(testingT, float64(0), report["unique_visitor_count"])
			require.Equal(testingT, true, report["counts_only"])
			require.Equal(testingT, "utc_day", report["time_resolution"])
			require.Len(testingT, report["trend"].([]any), portfolioTrafficReportDefaultDays)
			for _, point := range report["trend"].([]any) {
				require.Equal(testingT, float64(0), point.(map[string]any)["unique_visitors"])
			}
			require.Equal(testingT, float64(0), report["sites"].([]any)[0].(map[string]any)["unique_visitor_count"])
			sites := readVisiblePortfolioResponse(testingT, server, RoleUser, "/api/sites", http.StatusOK)
			require.Nil(testingT, sites["sites"].([]any)[0].(map[string]any)["unique_visitor_count"])
		})
	}
}
