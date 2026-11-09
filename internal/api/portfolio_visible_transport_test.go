package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testVisiblePortfolioPath = "/api/reports/traffic/portfolio"

func TestPortfolioVisibleScopeMatchesSiteAccess(testingT *testing.T) {
	harness := buildTrafficReportHarness(testingT, true)
	sites := []model.Site{
		{ID: storage.NewID(), Name: "Owned", AllowedOrigin: "https://owned.example", OwnerEmail: testTrafficReportOwnerEmail},
		{ID: storage.NewID(), Name: "Created", AllowedOrigin: "https://created.example", OwnerEmail: testTrafficReportOtherEmail, CreatorEmail: testTrafficReportOwnerEmail},
		{ID: storage.NewID(), Name: "Team", AllowedOrigin: "https://team.example", OwnerEmail: testTrafficReportOtherEmail},
		{ID: storage.NewID(), Name: "Excluded", AllowedOrigin: "https://excluded.example", OwnerEmail: testTrafficReportOtherEmail},
	}
	for index, site := range sites {
		require.NoError(testingT, harness.database.Create(&site).Error)
		for count := 0; count <= index; count++ {
			visit, err := model.NewSiteVisit(model.SiteVisitInput{SiteID: site.ID, URL: site.AllowedOrigin, VisitorID: site.ID, Occurred: time.Now().UTC()})
			require.NoError(testingT, err)
			require.NoError(testingT, harness.database.Create(&visit).Error)
		}
	}
	membership, err := model.NewSiteTeamMember(model.SiteTeamMemberInput{SiteID: sites[2].ID, Email: testTrafficReportOwnerEmail, AddedByEmail: testTrafficReportOtherEmail})
	require.NoError(testingT, err)
	require.NoError(testingT, harness.database.Create(&membership).Error)
	server := visiblePortfolioServer(testingT, harness)

	for _, scenario := range []struct {
		name  string
		role  UserRole
		ids   []string
		views float64
	}{
		{name: "owner creator and team", role: RoleUser, ids: []string{sites[0].ID, sites[1].ID, sites[2].ID}, views: 6},
		{name: "global admin", role: RoleAdmin, ids: []string{sites[0].ID, sites[1].ID, sites[2].ID, sites[3].ID}, views: 10},
	} {
		testingT.Run(scenario.name, func(testingT *testing.T) {
			report := readVisiblePortfolioResponse(testingT, server, scenario.role, testVisiblePortfolioPath+"?scope=visible&days=30", http.StatusOK)
			require.Equal(testingT, "visible", report["scope"])
			require.Equal(testingT, scenario.views, report["visit_count"])
			require.Equal(testingT, float64(len(scenario.ids)), report["unique_visitor_count"])
			require.Equal(testingT, float64(len(scenario.ids)), report["site_count"])
			reportIDs := visiblePortfolioResponseIDs(report)
			require.ElementsMatch(testingT, scenario.ids, reportIDs)
			siteList := readVisiblePortfolioResponse(testingT, server, scenario.role, "/api/sites", http.StatusOK)
			listedIDs := []string{}
			for _, row := range siteList["sites"].([]any) {
				listedIDs = append(listedIDs, row.(map[string]any)["id"].(string))
			}
			require.ElementsMatch(testingT, listedIDs, reportIDs)
		})
	}
	owned := readVisiblePortfolioResponse(testingT, server, RoleUser, testVisiblePortfolioPath, http.StatusOK)
	require.Equal(testingT, "owned", owned["scope"])
	require.ElementsMatch(testingT, []string{sites[0].ID, sites[1].ID}, visiblePortfolioResponseIDs(owned))
	adminOwned := readVisiblePortfolioResponse(testingT, server, RoleAdmin, testVisiblePortfolioPath+"?scope=owned", http.StatusOK)
	require.ElementsMatch(testingT, []string{sites[0].ID, sites[1].ID}, visiblePortfolioResponseIDs(adminOwned))

	require.NoError(testingT, harness.database.Delete(&membership).Error)
	revoked := readVisiblePortfolioResponse(testingT, server, RoleUser, testVisiblePortfolioPath+"?scope=visible", http.StatusOK)
	require.ElementsMatch(testingT, []string{sites[0].ID, sites[1].ID}, visiblePortfolioResponseIDs(revoked))
}

func TestPortfolioVisibleScopeValidatesScopeAndPreservesCustomReports(testingT *testing.T) {
	harness := buildTrafficReportHarness(testingT, true)
	insertTrafficReportSite(testingT, harness.database)
	server := visiblePortfolioServer(testingT, harness)
	payload := []byte(`{"name":"Owned selection","site_ids":["` + testTrafficReportSiteID + `"]}`)
	response, err := server.Client().Post(server.URL+testVisiblePortfolioPath+"/reports", "application/json", bytes.NewReader(payload))
	require.NoError(testingT, err)
	defer response.Body.Close()
	require.Equal(testingT, http.StatusCreated, response.StatusCode)
	var definition PortfolioTrafficReportDefinitionResponse
	require.NoError(testingT, json.NewDecoder(response.Body).Decode(&definition))
	for _, scope := range []string{"", "&scope=owned"} {
		report := readVisiblePortfolioResponse(testingT, server, RoleUser, testVisiblePortfolioPath+"?report_id="+definition.ID+scope, http.StatusOK)
		require.Equal(testingT, definition.ID, report["report_id"])
		require.Equal(testingT, "owned", report["scope"])
		require.Equal(testingT, []string{testTrafficReportSiteID}, visiblePortfolioResponseIDs(report))
	}
	for _, query := range []string{"?scope=unknown", "?scope=VISIBLE", "?scope=visible&report_id=" + definition.ID} {
		readVisiblePortfolioResponse(testingT, server, RoleUser, testVisiblePortfolioPath+query, http.StatusBadRequest)
	}
	report := readVisiblePortfolioResponse(testingT, server, RoleUser, testVisiblePortfolioPath+"?scope=visible&report_id="+model.PortfolioTrafficReportDefaultID, http.StatusOK)
	require.Equal(testingT, "visible", report["scope"])
}

func TestPortfolioVisibleScopeIncludesAggregateTeamCounts(testingT *testing.T) {
	harness := buildTrafficReportHarness(testingT, true)
	insertTrafficReportSite(testingT, harness.database)
	require.NoError(testingT, harness.database.Model(&model.Site{}).Where("id = ?", testTrafficReportSiteID).Updates(map[string]any{"owner_email": testTrafficReportOtherEmail, "creator_email": testTrafficReportOtherEmail, "traffic_profile": model.TrafficProfileAggregate}).Error)
	insertTrafficReportTeamMember(testingT, harness.database, testTrafficReportOwnerEmail)
	require.NoError(testingT, harness.database.Create(&model.SiteVisitCount{SiteID: testTrafficReportSiteID, Date: time.Now().UTC().Format(visitTrendDateFormat), Count: 9}).Error)
	server := visiblePortfolioServer(testingT, harness)
	report := readVisiblePortfolioResponse(testingT, server, RoleUser, testVisiblePortfolioPath+"?scope=visible", http.StatusOK)
	require.Equal(testingT, float64(9), report["visit_count"])
	require.Nil(testingT, report["unique_visitor_count"])
	require.True(testingT, report["counts_only"].(bool))
	rows := report["sites"].([]any)
	require.Len(testingT, rows, 1)
	require.Nil(testingT, rows[0].(map[string]any)["unique_visitor_count"])
	owned := readVisiblePortfolioResponse(testingT, server, RoleUser, testVisiblePortfolioPath, http.StatusOK)
	require.Equal(testingT, float64(0), owned["site_count"])
}

func visiblePortfolioServer(testingT *testing.T, harness trafficReportHarness) *httptest.Server {
	testingT.Helper()
	router := gin.New()
	router.Use(func(context *gin.Context) {
		role := RoleUser
		if context.GetHeader("X-Test-Role") == string(RoleAdmin) {
			role = RoleAdmin
		}
		context.Set(contextKeyCurrentUser, &CurrentUser{Email: " TRAFFIC-OWNER@example.com ", Role: role})
	})
	sites := NewSiteHandlers(harness.database, zap.NewNop(), "https://loopaware.example", nil, NewDatabaseSiteStatisticsProvider(harness.database), nil)
	router.GET("/api/sites", sites.ListSites)
	router.GET(testVisiblePortfolioPath, harness.handlers.GetPortfolioReport)
	router.POST(testVisiblePortfolioPath+"/reports", harness.handlers.CreatePortfolioReport)
	server := httptest.NewServer(router)
	testingT.Cleanup(server.Close)
	return server
}

func readVisiblePortfolioResponse(testingT *testing.T, server *httptest.Server, role UserRole, path string, status int) map[string]any {
	testingT.Helper()
	request, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	require.NoError(testingT, err)
	request.Header.Set("X-Test-Role", string(role))
	response, err := server.Client().Do(request)
	require.NoError(testingT, err)
	defer response.Body.Close()
	require.Equal(testingT, status, response.StatusCode)
	var payload map[string]any
	require.NoError(testingT, json.NewDecoder(response.Body).Decode(&payload))
	return payload
}

func visiblePortfolioResponseIDs(payload map[string]any) []string {
	ids := []string{}
	for _, row := range payload["sites"].([]any) {
		ids = append(ids, row.(map[string]any)["site_id"].(string))
	}
	return ids
}
