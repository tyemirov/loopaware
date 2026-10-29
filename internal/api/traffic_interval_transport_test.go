package api_test

import (
	"encoding/csv"
	"encoding/json"
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

func TestSevenDayTrafficPublicWindow(testingT *testing.T) {
	for _, profile := range []model.TrafficProfile{model.TrafficProfileDetailed, model.TrafficProfileAggregate} {
		testingT.Run(string(profile), func(testingT *testing.T) {
			harness := newSiteTestHarness(testingT)
			site := insertSite(testingT, harness.database, "Seven days", "https://seven.example", testUserEmailAddress)
			require.NoError(testingT, harness.database.Model(&site).Update("traffic_profile", profile).Error)
			now := time.Now().UTC()
			start := now.Truncate(24*time.Hour).AddDate(0, 0, -6)
			otherSite := insertSite(testingT, harness.database, "Unrelated", "https://other.example", testUserEmailAddress)
			if profile == model.TrafficProfileAggregate {
				for _, row := range []model.SiteVisitCount{
					{SiteID: site.ID, Date: start.AddDate(0, 0, -1).Format(time.DateOnly), Count: 5},
					{SiteID: site.ID, Date: start.Format(time.DateOnly), Count: 3},
					{SiteID: site.ID, Date: now.Format(time.DateOnly), Count: 4},
					{SiteID: otherSite.ID, Date: now.Format(time.DateOnly), Count: 99},
				} {
					require.NoError(testingT, harness.database.Create(&row).Error)
				}
			} else {
				for _, input := range []model.SiteVisitInput{
					{SiteID: site.ID, URL: site.AllowedOrigin + "/excluded?utm_source=excluded", VisitorID: storage.NewID(), Occurred: start.Add(-time.Second), Viewport: "1920x1080", Timezone: "Europe/London"},
					{SiteID: site.ID, URL: site.AllowedOrigin + "/boundary?utm_source=included", VisitorID: storage.NewID(), Occurred: start, Viewport: "390x844", Timezone: "America/Los_Angeles"},
					{SiteID: site.ID, URL: site.AllowedOrigin + "/current?utm_source=included", VisitorID: storage.NewID(), Occurred: now, Viewport: "390x844", Timezone: "America/Los_Angeles"},
					{SiteID: site.ID, URL: site.AllowedOrigin + "/bot", Occurred: now, IsBot: true},
					{SiteID: otherSite.ID, URL: otherSite.AllowedOrigin, VisitorID: storage.NewID(), Occurred: now},
				} {
					visit, err := model.NewSiteVisit(input)
					require.NoError(testingT, err)
					require.NoError(testingT, harness.database.Create(&visit).Error)
				}
			}
			router := gin.New()
			router.Use(func(context *gin.Context) {
				context.Set(testSessionContextKey, &api.CurrentUser{Email: testUserEmailAddress, Role: api.RoleUser})
			})
			basePath := "/api/sites/:id/visits/"
			router.GET(basePath+"stats", harness.handlers.VisitStats)
			router.GET(basePath+"trend", harness.handlers.VisitTrend)
			router.GET(basePath+"attribution", harness.handlers.VisitAttribution)
			router.GET(basePath+"engagement", harness.handlers.VisitEngagement)
			router.GET(basePath+"devices", harness.handlers.DeviceBreakdown)
			router.GET(basePath+"locations", harness.handlers.LocationDistribution)
			router.GET(basePath+"export", harness.handlers.ExportTraffic)
			server := httptest.NewServer(router)
			defer server.Close()
			readJSON := func(endpoint string, expectedStatus int) map[string]any {
				testingT.Helper()
				response, err := server.Client().Get(server.URL + "/api/sites/" + site.ID + "/visits/" + endpoint + "?interval=7days")
				require.NoError(testingT, err)
				defer response.Body.Close()
				require.Equal(testingT, expectedStatus, response.StatusCode, endpoint)
				var payload map[string]any
				require.NoError(testingT, json.NewDecoder(response.Body).Decode(&payload))
				return payload
			}
			stats := readJSON("stats", http.StatusOK)
			require.Equal(testingT, "7days", stats["interval"])
			trend := readJSON("trend", http.StatusOK)
			require.Equal(testingT, "7days", trend["interval"])
			points := trend["trend"].([]any)
			if profile == model.TrafficProfileAggregate {
				require.Equal(testingT, float64(7), stats["visit_count"])
				require.Nil(testingT, stats["unique_visitor_count"])
				require.Equal(testingT, "accepted_requests", stats["metric"])
				require.Len(testingT, points, 2)
				require.Equal(testingT, float64(2), trend["days"])
				require.Nil(testingT, points[0].(map[string]any)["unique_visitors"])
				for _, endpoint := range []string{"attribution", "engagement", "devices", "locations"} {
					require.Equal(testingT, "metric_unavailable", readJSON(endpoint, http.StatusConflict)["error"])
				}
			} else {
				require.Equal(testingT, float64(2), stats["visit_count"])
				require.Equal(testingT, float64(2), stats["unique_visitor_count"])
				require.Len(testingT, stats["top_pages"], 2)
				require.Len(testingT, stats["recent_visits"], 2)
				require.Len(testingT, points, 7)
				require.Equal(testingT, float64(7), trend["days"])
				require.Equal(testingT, float64(1), points[0].(map[string]any)["page_views"])
				attribution := readJSON("attribution", http.StatusOK)
				require.Equal(testingT, "7days", attribution["interval"])
				require.Equal(testingT, []any{map[string]any{"value": "included", "visit_count": float64(2)}}, attribution["sources"])
				engagement := readJSON("engagement", http.StatusOK)
				require.Equal(testingT, "7days", engagement["interval"])
				require.Equal(testingT, float64(7), engagement["days"])
				require.Equal(testingT, float64(2), engagement["tracked_visitor_count"])
				require.Equal(testingT, float64(1), engagement["average_pages_per_visitor"])
				devices := readJSON("devices", http.StatusOK)
				require.Equal(testingT, "7days", devices["interval"])
				require.Equal(testingT, []any{map[string]any{"device_type": "mobile", "visit_count": float64(2)}}, devices["device_types"])
				locations := readJSON("locations", http.StatusOK)
				require.Equal(testingT, "7days", locations["interval"])
				require.Len(testingT, locations["locations"], 1)
				require.Equal(testingT, float64(2), locations["locations"].([]any)[0].(map[string]any)["visit_count"])
			}
			require.Equal(testingT, start.Format(time.DateOnly), points[0].(map[string]any)["date"])
			require.Equal(testingT, now.Format(time.DateOnly), points[len(points)-1].(map[string]any)["date"])
			response, err := server.Client().Get(server.URL + "/api/sites/" + site.ID + "/visits/export?interval=7days")
			require.NoError(testingT, err)
			defer response.Body.Close()
			require.Equal(testingT, http.StatusOK, response.StatusCode)
			rows, err := csv.NewReader(response.Body).ReadAll()
			require.NoError(testingT, err)
			require.Len(testingT, rows, 3)
			if profile == model.TrafficProfileAggregate {
				require.Equal(testingT, []string{"site_id", "utc_date", "accepted_requests"}, rows[0])
				require.Equal(testingT, []string{site.ID, start.Format(time.DateOnly), "3"}, rows[1])
				require.Equal(testingT, []string{site.ID, now.Format(time.DateOnly), "4"}, rows[2])
			} else {
				require.Equal(testingT, "/current", rows[1][2])
				require.Equal(testingT, "/boundary", rows[2][2])
				require.Contains(testingT, response.Header.Get("Content-Disposition"), "-7days.csv")
			}
		})
	}
}
