package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/api"
	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUniqueTrafficPublicCountsAndRanking(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := insertSite(testingT, harness.database, "Unique traffic", "https://unique.example", testUserEmailAddress)
	other := insertSite(testingT, harness.database, "Other traffic", "https://other.example", testUserEmailAddress)
	now := time.Now().UTC()
	visitorA, visitorB, visitorC, visitorD := storage.NewID(), storage.NewID(), storage.NewID(), storage.NewID()
	add := func(siteID, path, visitor, source, viewport, timezone, locale string, when time.Time, bot bool) {
		visit, err := model.NewSiteVisit(model.SiteVisitInput{SiteID: siteID, URL: site.AllowedOrigin + path + "?utm_source=" + source, VisitorID: visitor, Viewport: viewport, ScreenResolution: viewport, Timezone: timezone, Locale: locale, Occurred: when, IsBot: bot})
		require.NoError(testingT, err)
		require.NoError(testingT, harness.database.Create(&visit).Error)
	}
	for index := 0; index < 6; index++ {
		add(site.ID, "/busy/", visitorA, "busy", "1200x800", "Europe/London", "en-GB", now, false)
	}
	for page := 0; page < 11; page++ {
		for visit := 0; visit < 4; visit++ {
			add(site.ID, fmt.Sprintf("/busy-%02d", page), visitorA, "busy", "1200x800", "Europe/London", "en-GB", now, false)
		}
	}
	for index, visitor := range []string{visitorB, visitorB, visitorC, visitorD, ""} {
		viewport, locale := "390x844", "en-US"
		if index == 1 {
			viewport, locale = "400x844", "en-GB"
		}
		path := "/reach"
		if index%2 == 0 {
			path += "/"
		}
		source := "reach"
		if index == 1 {
			source = "Reach"
		}
		add(site.ID, path, visitor, source, viewport, "America/Los_Angeles", locale, now, false)
	}
	add(site.ID, "/reach", storage.NewID(), "reach", "390x844", "America/Los_Angeles", "en-US", now, true)
	add(site.ID, "/reach", storage.NewID(), "reach", "390x844", "America/Los_Angeles", "en-US", now.Add(-8*24*time.Hour), false)
	add(other.ID, "/reach", storage.NewID(), "reach", "390x844", "America/Los_Angeles", "en-US", now, false)
	router := gin.New()
	router.Use(func(context *gin.Context) {
		context.Set(testSessionContextKey, &api.CurrentUser{Email: testUserEmailAddress, Role: api.RoleUser})
	})
	router.GET("/api/sites/:id/visits/stats", harness.handlers.VisitStats)
	router.GET("/api/sites/:id/visits/attribution", harness.handlers.VisitAttribution)
	router.GET("/api/sites/:id/visits/devices", harness.handlers.DeviceBreakdown)
	router.GET("/api/sites/:id/visits/locations", harness.handlers.LocationDistribution)
	server := httptest.NewServer(router)
	defer server.Close()
	read := func(endpoint, sort string, status int) map[string]any {
		testingT.Helper()
		response, err := server.Client().Get(server.URL + "/api/sites/" + site.ID + "/visits/" + endpoint + "?interval=7days&limit=1" + sort)
		require.NoError(testingT, err)
		defer response.Body.Close()
		require.Equal(testingT, status, response.StatusCode)
		var payload map[string]any
		require.NoError(testingT, json.NewDecoder(response.Body).Decode(&payload))
		return payload
	}
	first := func(payload map[string]any, key string) map[string]any {
		testingT.Helper()
		rows := payload[key].([]any)
		require.NotEmpty(testingT, rows)
		return rows[0].(map[string]any)
	}
	stats := read("stats", "&sort=unique_visitors", http.StatusOK)
	page := first(stats, "top_pages")
	require.Equal(testingT, "/reach", page["path"])
	require.Equal(testingT, float64(3), page["unique_visitor_count"])
	require.Equal(testingT, float64(5), page["visit_count"])
	attribution := read("attribution", "&sort=unique_visitors", http.StatusOK)
	source := first(attribution, "sources")
	require.Equal(testingT, "reach", source["value"])
	require.Equal(testingT, float64(3), source["unique_visitor_count"])
	devicePayload := read("devices", "&sort=unique_visitors", http.StatusOK)
	device := first(devicePayload, "device_types")
	require.Equal(testingT, "mobile", device["device_type"])
	require.Equal(testingT, float64(3), device["unique_visitor_count"])
	for _, field := range []string{"top_resolutions", "top_viewports"} {
		row := first(devicePayload, field)
		require.Equal(testingT, "390x844", row["value"])
		require.Equal(testingT, float64(3), row["unique_visitor_count"])
	}
	require.Equal(testingT, float64(4), first(attribution, "mediums")["unique_visitor_count"])
	require.Equal(testingT, float64(4), first(attribution, "campaigns")["unique_visitor_count"])
	location := first(read("locations", "&sort=unique_visitors", http.StatusOK), "locations")
	require.Equal(testingT, "US", location["country"])
	require.Equal(testingT, float64(3), location["unique_visitor_count"])
	require.Equal(testingT, "/busy", first(read("stats", "", http.StatusOK), "top_pages")["path"])
	require.Equal(testingT, "busy", first(read("attribution", "&sort=page_views", http.StatusOK), "sources")["value"])
	for _, endpoint := range []string{"stats", "attribution", "devices", "locations"} {
		require.Equal(testingT, "invalid_sort", read(endpoint, "&sort=invalid", http.StatusBadRequest)["error"])
	}
	require.NoError(testingT, harness.database.Model(&site).Update("traffic_profile", model.TrafficProfileAggregate).Error)
	aggregate := read("stats", "&sort=unique_visitors", http.StatusOK)
	require.Nil(testingT, aggregate["unique_visitor_count"])
	require.Nil(testingT, aggregate["top_pages"])
	for _, endpoint := range []string{"attribution", "devices", "locations"} {
		require.Equal(testingT, "metric_unavailable", read(endpoint, "&sort=unique_visitors", http.StatusConflict)["error"])
	}
}

func TestUniqueTrafficTieOrderAndUnidentifiedViews(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := insertSite(testingT, harness.database, "Tied traffic", "https://ties.example", testUserEmailAddress)
	visitor := storage.NewID()
	for _, bucket := range []struct {
		path, source, viewport, country, visitor string
		views                                    int
	}{
		{"/zulu", "Zulu", "390x844", "US", visitor, 7},
		{"/alpha", "Alpha", "1200x800", "GB", visitor, 2},
		{"/anonymous", "Anonymous", "2000x1000", "ZA", "", 9},
	} {
		for index := 0; index < bucket.views; index++ {
			visit, err := model.NewSiteVisit(model.SiteVisitInput{SiteID: site.ID, URL: site.AllowedOrigin + bucket.path + "?utm_source=" + bucket.source, VisitorID: bucket.visitor, Viewport: bucket.viewport, ScreenResolution: bucket.viewport, GeoCountry: bucket.country, GeoSource: "edge", Occurred: time.Now().UTC()})
			require.NoError(testingT, err)
			require.NoError(testingT, harness.database.Create(&visit).Error)
		}
	}
	router := gin.New()
	router.Use(func(context *gin.Context) {
		context.Set(testSessionContextKey, &api.CurrentUser{Email: testUserEmailAddress, Role: api.RoleUser})
	})
	router.GET("/api/sites/:id/visits/stats", harness.handlers.VisitStats)
	router.GET("/api/sites/:id/visits/attribution", harness.handlers.VisitAttribution)
	router.GET("/api/sites/:id/visits/devices", harness.handlers.DeviceBreakdown)
	router.GET("/api/sites/:id/visits/locations", harness.handlers.LocationDistribution)
	server := httptest.NewServer(router)
	defer server.Close()
	for _, scenario := range []struct{ endpoint, field, key, first string }{
		{"stats", "top_pages", "path", "/alpha"},
		{"attribution", "sources", "value", "alpha"},
		{"devices", "device_types", "device_type", "desktop"},
		{"devices", "top_resolutions", "value", "1200x800"},
		{"devices", "top_viewports", "value", "1200x800"},
		{"locations", "locations", "country", "GB"},
	} {
		response, err := server.Client().Get(server.URL + "/api/sites/" + site.ID + "/visits/" + scenario.endpoint + "?interval=all&sort=unique_visitors")
		require.NoError(testingT, err)
		require.Equal(testingT, http.StatusOK, response.StatusCode)
		var payload map[string]any
		require.NoError(testingT, json.NewDecoder(response.Body).Decode(&payload))
		require.NoError(testingT, response.Body.Close())
		rows := payload[scenario.field].([]any)
		first := rows[0].(map[string]any)
		require.Equal(testingT, scenario.first, first[scenario.key], scenario.field)
		require.Equal(testingT, float64(1), first["unique_visitor_count"])
		if scenario.field != "device_types" {
			last := rows[len(rows)-1].(map[string]any)
			require.Equal(testingT, float64(0), last["unique_visitor_count"])
			require.Equal(testingT, float64(9), last["visit_count"])
		}
	}
}
