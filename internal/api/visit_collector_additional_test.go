package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	loopawareapi "github.com/MarkoPoloResearchLab/loopaware/internal/api"
	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	testVisitCreateCallbackName = "force_visit_create_error"
	testVisitCreateErrorMessage = "visit create failed"
	testVisitTableName          = "site_visits"
	testVisitPath               = "/public/visits"
	testVisitQueryPrefix        = "/public/visits?site_id="
	testVisitUnknownURL         = "http://unknown.example/page"
	testVisitOrigin             = "http://visitors.example"
	testVisitOriginPage         = "http://visitors.example/page"
	testVisitReferer            = "http://visitors.example/from-referer"
	testVisitRefererQuery       = "http://visitors.example/from-query"
	testVisitSaveErrorOrigin    = "http://savefail.example"
	testVisitSaveErrorURL       = "http://savefail.example/page"
	testVisitSaveFailedToken    = "save_failed"
	testVisitBotUserAgent       = "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
	testVisitWhatsAppUserAgent  = "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.0.0 Mobile Safari/537.36 WhatsApp/2.24.3.78"
)

func TestCollectVisitRequiresSiteID(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	response := performJSONRequest(testingT, api.router, http.MethodGet, testVisitPath, nil, nil)
	require.Equal(testingT, http.StatusBadRequest, response.Code)
	require.Contains(testingT, response.Body.String(), "missing site_id")
}

func TestCollectVisitRejectsUnknownSite(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	response := performJSONRequest(testingT, api.router, http.MethodGet, testVisitQueryPrefix+"missing&url="+testVisitUnknownURL, nil, nil)
	require.Equal(testingT, http.StatusNotFound, response.Code)
	require.Contains(testingT, response.Body.String(), "unknown site")
}

func TestCollectVisitRejectsInvalidVisitorID(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Visit Invalid Visitor", testVisitOrigin, "owner@example.com")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, testVisitQueryPrefix+site.ID+"&url="+testVisitOriginPage+"&visitor_id=bad", nil)
	request.Header.Set("Origin", testVisitOrigin)
	api.router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), "invalid_visitor")
}

func TestCollectVisitUsesRefererAndHeaderVisitorID(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Visit Referer", testVisitOrigin, "owner@example.com")

	visitorID := uuid.NewString()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, testVisitQueryPrefix+site.ID, nil)
	request.Header.Set("Origin", testVisitOrigin)
	request.Header.Set("Referer", testVisitReferer)
	request.Header.Set("X-Visitor-Id", visitorID)
	api.router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusOK, recorder.Code)

	var stored model.SiteVisit
	require.NoError(testingT, api.database.Order("occurred_at desc").First(&stored).Error)
	require.Equal(testingT, visitorID, stored.VisitorID)
	require.Equal(testingT, testVisitReferer, stored.URL)
}

func TestCollectVisitReportsSaveError(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Visit Save Error", testVisitSaveErrorOrigin, "owner@example.com")

	callbackName := testVisitCreateCallbackName
	api.database.Callback().Create().Before("gorm:create").Register(callbackName, func(database *gorm.DB) {
		if database.Statement != nil && database.Statement.Table == testVisitTableName {
			database.AddError(errors.New(testVisitCreateErrorMessage))
		}
	})
	testingT.Cleanup(func() {
		api.database.Callback().Create().Remove(callbackName)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, testVisitQueryPrefix+site.ID+"&url="+testVisitSaveErrorURL, nil)
	request.Header.Set("Origin", testVisitSaveErrorOrigin)
	api.router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), testVisitSaveFailedToken)
}

func TestCollectVisitUsesQueryReferrerWhenHeaderMissing(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Visit Query Referrer", testVisitOrigin, "owner@example.com")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, testVisitQueryPrefix+site.ID+"&referrer="+testVisitRefererQuery, nil)
	request.Header.Set("Origin", testVisitOrigin)
	api.router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusOK, recorder.Code)

	var stored model.SiteVisit
	require.NoError(testingT, api.database.Order("occurred_at desc").First(&stored).Error)
	require.Equal(testingT, testVisitRefererQuery, stored.URL)
	require.Equal(testingT, testVisitRefererQuery, stored.Referrer)
}

func TestCollectVisitMarksBotTraffic(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Visit Bot", testVisitOrigin, "owner@example.com")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, testVisitQueryPrefix+site.ID+"&url="+testVisitOriginPage, nil)
	request.Header.Set("Origin", testVisitOrigin)
	request.Header.Set("User-Agent", testVisitBotUserAgent)
	api.router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusOK, recorder.Code)

	var stored model.SiteVisit
	require.NoError(testingT, api.database.Order("occurred_at desc").First(&stored).Error)
	require.True(testingT, stored.IsBot)
}

func TestCollectVisitTreatsWhatsAppUserAgentAsHumanTraffic(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Visit WhatsApp", testVisitOrigin, "owner@example.com")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, testVisitQueryPrefix+site.ID+"&url="+testVisitOriginPage, nil)
	request.Header.Set("Origin", testVisitOrigin)
	request.Header.Set("User-Agent", testVisitWhatsAppUserAgent)
	api.router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusOK, recorder.Code)

	var stored model.SiteVisit
	require.NoError(testingT, api.database.Order("occurred_at desc").First(&stored).Error)
	require.False(testingT, stored.IsBot)

	statsProvider := loopawareapi.NewDatabaseSiteStatisticsProvider(api.database)
	visitCount, visitCountErr := statsProvider.VisitCount(context.Background(), site.ID)
	require.NoError(testingT, visitCountErr)
	require.Equal(testingT, int64(1), visitCount)
}
