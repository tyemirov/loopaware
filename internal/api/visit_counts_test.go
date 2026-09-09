package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/api"
	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

func TestVisitCountsRejectDetailedSite(testingT *testing.T) {
	harness := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, harness.database, "Detailed site", "https://game.example", "owner@example.com")
	server := httptest.NewServer(harness.router)
	defer server.Close()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/public/sites/"+site.ID+"/visit-counts", strings.NewReader("{}"))
	require.NoError(testingT, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", site.AllowedOrigin)
	response, err := server.Client().Do(request)
	require.NoError(testingT, err)
	defer response.Body.Close()
	require.Equal(testingT, http.StatusConflict, response.StatusCode)
}

func TestVisitCountsContract(testingT *testing.T) {
	harness := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, harness.database, "Aggregate site", "https://game.example", "owner@example.com")
	require.NoError(testingT, harness.database.Model(&site).Update("traffic_profile", model.TrafficProfileAggregate).Error)
	core, logs := observer.New(zap.InfoLevel)
	router := gin.New()
	router.Use(api.RequestLogger(zap.New(core)))
	date := time.Date(2026, 9, 8, 12, 30, 0, 0, time.UTC)
	router.POST(api.VisitCountsPath, api.NewVisitCountHandler(harness.database, func() time.Time { return date }))
	server := httptest.NewServer(router)
	defer server.Close()
	path := "/public/sites/" + site.ID + "/visit-counts"
	requestCount := 24
	statuses := make(chan int, requestCount)
	var requests sync.WaitGroup
	for index := 0; index < requestCount; index++ {
		requests.Add(1)
		go func() {
			defer requests.Done()
			request, err := http.NewRequest(http.MethodPost, server.URL+path, strings.NewReader("{}"))
			if err != nil {
				statuses <- 0
				return
			}
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Origin", site.AllowedOrigin)
			request.Header.Set("User-Agent", "private-device-marker")
			response, err := server.Client().Do(request)
			if err != nil {
				statuses <- 0
				return
			}
			response.Body.Close()
			statuses <- response.StatusCode
		}()
	}
	requests.Wait()
	close(statuses)
	for status := range statuses {
		require.Equal(testingT, http.StatusNoContent, status)
	}

	cases := []struct {
		name, suffix, body, origin, media string
		status                            int
	}{
		{"unknown field", "", `{"visitor_id":"private-visitor-marker"}`, site.AllowedOrigin, "application/json", 400},
		{"query", "?url=private-query-marker", "{}", site.AllowedOrigin, "application/json", 400},
		{"null", "", "null", site.AllowedOrigin, "application/json", 400},
		{"array", "", "[]", site.AllowedOrigin, "application/json", 400},
		{"trailing JSON", "", "{}{}", site.AllowedOrigin, "application/json", 400},
		{"large body", "", strings.Repeat(" ", 257) + "{}", site.AllowedOrigin, "application/json", 413},
		{"media type", "", "{}", site.AllowedOrigin, "text/plain", 415},
		{"origin", "", "{}", "https://forbidden.example", "application/json", 403},
		{"missing origin", "", "{}", "", "application/json", 403},
	}
	for _, testCase := range cases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			request, err := http.NewRequest(http.MethodPost, server.URL+path+testCase.suffix, strings.NewReader(testCase.body))
			require.NoError(testingT, err)
			request.Header.Set("Content-Type", testCase.media)
			request.Header.Set("Origin", testCase.origin)
			request.Header.Set("User-Agent", "private-device-marker")
			response, err := server.Client().Do(request)
			require.NoError(testingT, err)
			defer response.Body.Close()
			require.Equal(testingT, testCase.status, response.StatusCode)
		})
	}
	var counts []model.SiteVisitCount
	require.NoError(testingT, harness.database.Find(&counts).Error)
	require.Equal(testingT, []model.SiteVisitCount{{SiteID: site.ID, Date: "2026-09-08", Count: int64(requestCount)}}, counts)
	var rawCount int64
	require.NoError(testingT, harness.database.Model(&model.SiteVisit{}).Count(&rawCount).Error)
	require.Zero(testingT, rawCount)

	// A database error must be visible as failure without logging request data.
	require.NoError(testingT, harness.database.Callback().Raw().Before("gorm:raw").Register("fail_count", func(database *gorm.DB) { database.AddError(io.ErrClosedPipe) }))
	request, err := http.NewRequest(http.MethodPost, server.URL+path, strings.NewReader("{}"))
	require.NoError(testingT, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", site.AllowedOrigin)
	response, err := server.Client().Do(request)
	require.NoError(testingT, err)
	response.Body.Close()
	require.Equal(testingT, http.StatusInternalServerError, response.StatusCode)
	for _, entry := range logs.All() {
		fields := entry.ContextMap()
		require.NotContains(testingT, fields, "ip")
		require.NotContains(testingT, fields, "ua")
		require.Equal(testingT, api.VisitCountsPath, fields["path"])
		encoded, err := json.Marshal(fields)
		require.NoError(testingT, err)
		require.NotContains(testingT, string(encoded), "private-")
	}
}
