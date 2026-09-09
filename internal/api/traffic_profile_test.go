package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/api"
	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAggregateSiteCreationAndReports(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	router := gin.New()
	router.Use(func(context *gin.Context) {
		context.Set(testSessionContextKey, &api.CurrentUser{Email: testUserEmailAddress, Role: api.RoleUser})
	})
	router.POST("/api/sites", harness.handlers.CreateSite)
	router.DELETE("/api/sites/:id", harness.handlers.DeleteSite)
	router.GET("/api/sites/:id/visits/stats", harness.handlers.VisitStats)
	router.GET("/api/sites/:id/visits/export", harness.handlers.ExportTraffic)
	router.POST(api.VisitCountsPath, api.NewVisitCountHandler(harness.database, time.Now))
	server := httptest.NewServer(router)
	defer server.Close()
	response, err := server.Client().Post(server.URL+"/api/sites", "application/json", strings.NewReader(`{"name":"Native game","allowed_origin":"https://game.example","traffic_profile":"aggregate"}`))
	require.NoError(testingT, err)
	defer response.Body.Close()
	require.Equal(testingT, http.StatusOK, response.StatusCode)
	var profile map[string]any
	require.NoError(testingT, json.NewDecoder(response.Body).Decode(&profile))
	require.Equal(testingT, "aggregate", profile["traffic_profile"])
	require.Nil(testingT, profile["unique_visitor_count"])
	siteID := profile["id"].(string)
	for index := 0; index < 2; index++ {
		request, err := http.NewRequest(http.MethodPost, server.URL+"/public/sites/"+siteID+"/visit-counts", strings.NewReader("{}"))
		require.NoError(testingT, err)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "https://game.example")
		response, err := server.Client().Do(request)
		require.NoError(testingT, err)
		response.Body.Close()
		require.Equal(testingT, http.StatusNoContent, response.StatusCode)
	}
	response, err = server.Client().Get(server.URL + "/api/sites/" + siteID + "/visits/stats")
	require.NoError(testingT, err)
	defer response.Body.Close()
	var stats map[string]any
	require.NoError(testingT, json.NewDecoder(response.Body).Decode(&stats))
	require.Equal(testingT, float64(2), stats["visit_count"])
	require.Nil(testingT, stats["unique_visitor_count"])
	require.Equal(testingT, "accepted_requests", stats["metric"])
	request, err := http.NewRequest(http.MethodDelete, server.URL+"/api/sites/"+siteID, nil)
	require.NoError(testingT, err)
	response, err = server.Client().Do(request)
	require.NoError(testingT, err)
	response.Body.Close()
	require.Equal(testingT, http.StatusNoContent, response.StatusCode)
	var count int64
	require.NoError(testingT, harness.database.Model(&model.SiteVisitCount{}).Where("site_id = ?", siteID).Count(&count).Error)
	require.Zero(testingT, count)
}

func TestAggregateProfileCannotReturnToDetailed(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := insertSite(testingT, harness.database, "Restricted", "https://game.example", testUserEmailAddress)
	require.NoError(testingT, harness.database.Model(&site).Update("traffic_profile", model.TrafficProfileAggregate).Error)
	router := gin.New()
	router.Use(func(context *gin.Context) {
		context.Set(testSessionContextKey, &api.CurrentUser{Email: testUserEmailAddress, Role: api.RoleUser})
	})
	router.PATCH("/api/sites/:id", harness.handlers.UpdateSite)
	server := httptest.NewServer(router)
	defer server.Close()
	request, err := http.NewRequest(http.MethodPatch, server.URL+"/api/sites/"+site.ID, strings.NewReader(`{"traffic_profile":"detailed"}`))
	require.NoError(testingT, err)
	request.Header.Set("Content-Type", "application/json")
	response, err := server.Client().Do(request)
	require.NoError(testingT, err)
	response.Body.Close()
	require.Equal(testingT, http.StatusConflict, response.StatusCode)
	require.NoError(testingT, harness.database.First(&site, "id = ?", site.ID).Error)
	require.Equal(testingT, model.TrafficProfileAggregate, site.TrafficProfile)
}
