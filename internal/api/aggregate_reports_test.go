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
