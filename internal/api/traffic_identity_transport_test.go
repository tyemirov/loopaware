package api_test

import (
	"bytes"
	"encoding/json"
	"image/gif"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/api"
	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestTrafficIdentityProxyFiltering(testingT *testing.T) {
	for _, scenario := range []struct {
		name, peer, forwarded, storedIP string
		allowPrivate                    bool
		errorStatus                     int
	}{
		{name: "public IPv4 through Caddy", peer: "192.168.96.5", forwarded: "203.0.113.12", storedIP: "203.0.113.12"},
		{name: "public IPv6 through Caddy", peer: "192.168.96.5", forwarded: "2001:db8::12", storedIP: "2001:db8::12"},
		{name: "private 10", peer: "192.168.96.5", forwarded: "10.20.30.40"},
		{name: "private 172", peer: "192.168.96.5", forwarded: "172.20.1.2"},
		{name: "private 192", peer: "192.168.96.5", forwarded: "192.168.1.2"},
		{name: "IPv4 loopback", peer: "192.168.96.5", forwarded: "127.0.0.1"},
		{name: "IPv4 link local", peer: "192.168.96.5", forwarded: "169.254.1.2"},
		{name: "IPv4 unspecified", peer: "192.168.96.5", forwarded: "0.0.0.0"},
		{name: "IPv4 multicast", peer: "192.168.96.5", forwarded: "239.1.2.3"},
		{name: "IPv6 private", peer: "192.168.96.5", forwarded: "fd12::1"},
		{name: "IPv6 loopback", peer: "192.168.96.5", forwarded: "::1"},
		{name: "IPv6 link local", peer: "192.168.96.5", forwarded: "fe80::1"},
		{name: "IPv6 unspecified", peer: "192.168.96.5", forwarded: "::"},
		{name: "IPv6 multicast", peer: "192.168.96.5", forwarded: "ff02::1"},
		{name: "mapped private", peer: "::ffff:192.168.96.5", forwarded: "::ffff:10.20.30.40"},
		{name: "mapped loopback", peer: "192.168.96.5", forwarded: "::ffff:127.0.0.1"},
		{name: "mapped link local", peer: "192.168.96.5", forwarded: "::ffff:169.254.1.2"},
		{name: "mapped unspecified", peer: "192.168.96.5", forwarded: "::ffff:0.0.0.0"},
		{name: "mapped multicast", peer: "192.168.96.5", forwarded: "::ffff:239.1.2.3"},
		{name: "mapped public", peer: "::ffff:192.168.96.5", forwarded: "::ffff:203.0.113.12", storedIP: "::ffff:203.0.113.12"},
		{name: "untrusted private peer spoofs public", peer: "10.20.30.40", forwarded: "203.0.113.12"},
		{name: "untrusted public peer spoofs private", peer: "203.0.113.12", forwarded: "10.20.30.40", storedIP: "203.0.113.12"},
		{name: "nearest untrusted hop is private", peer: "192.168.96.5", forwarded: "203.0.113.12, 10.20.30.40"},
		{name: "trusted chain", peer: "192.168.96.5", forwarded: "203.0.113.12, 192.168.96.6", storedIP: "203.0.113.12"},
		{name: "missing forwarding", peer: "192.168.96.5"},
		{name: "malformed forwarding", peer: "192.168.96.5", forwarded: "not-an-address"},
		{name: "unresolved socket peer", peer: "not-an-address", forwarded: "203.0.113.12", errorStatus: http.StatusInternalServerError},
		{name: "controlled private collection", peer: "10.20.30.40", storedIP: "10.20.30.40", allowPrivate: true},
	} {
		testingT.Run(scenario.name, func(testingT *testing.T) {
			harness := newSiteTestHarness(testingT)
			site := insertSite(testingT, harness.database, "Identity traffic", testVisitOrigin, testUserEmailAddress)
			collector := api.NewPublicHandlers(harness.database, zap.NewNop(), nil, nil, nil, nil, false, testVisitOrigin, "traffic-identity-test", nil, api.VisitCollectionPolicy{ExcludePrivateIPs: !scenario.allowPrivate})
			router := gin.New()
			require.NoError(testingT, router.SetTrustedProxies([]string{"192.168.96.0/20"}))
			router.RemoteIPHeaders = []string{"X-Forwarded-For"}
			router.Use(api.TrustedProxyHeaders([]netip.Prefix{netip.MustParsePrefix("192.168.96.0/20")}, nil))
			router.Use(func(context *gin.Context) {
				context.Set(testSessionContextKey, &api.CurrentUser{Email: testUserEmailAddress, Role: api.RoleUser})
			})
			router.GET(testVisitPath, collector.CollectVisit)
			router.GET("/api/sites/:id/visits/stats", harness.handlers.VisitStats)
			// Inject only the socket peer; forwarding resolution, collection and reports stay real.
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				request.RemoteAddr = net.JoinHostPort(scenario.peer, "43000")
				router.ServeHTTP(response, request)
			}))
			defer server.Close()
			request, err := http.NewRequest(http.MethodGet, server.URL+testVisitQueryPrefix+site.ID+"&url="+url.QueryEscape(testVisitOriginPage)+"&visitor_id="+storage.NewID(), nil)
			require.NoError(testingT, err)
			request.Header.Set("Origin", testVisitOrigin)
			request.Header.Set("X-Forwarded-For", scenario.forwarded)
			response, err := server.Client().Do(request)
			require.NoError(testingT, err)
			body, err := io.ReadAll(response.Body)
			require.NoError(testingT, err)
			require.NoError(testingT, response.Body.Close())
			if scenario.errorStatus != 0 {
				require.Equal(testingT, scenario.errorStatus, response.StatusCode)
				require.Contains(testingT, string(body), "client_address_unavailable")
				var count int64
				require.NoError(testingT, harness.database.Model(&model.SiteVisit{}).Count(&count).Error)
				require.Zero(testingT, count)
				return
			}
			require.Equal(testingT, http.StatusOK, response.StatusCode)
			require.Equal(testingT, "image/gif", response.Header.Get("Content-Type"))
			require.Equal(testingT, "no-store, no-cache, must-revalidate, max-age=0", response.Header.Get("Cache-Control"))
			require.Equal(testingT, "no-cache", response.Header.Get("Pragma"))
			pixel, err := gif.Decode(bytes.NewReader(body))
			require.NoError(testingT, err)
			require.Equal(testingT, 1, pixel.Bounds().Dx())
			require.Equal(testingT, 1, pixel.Bounds().Dy())
			var visits []model.SiteVisit
			require.NoError(testingT, harness.database.Find(&visits).Error)
			expectedCount := 0
			if scenario.storedIP != "" {
				expectedCount = 1
				require.Len(testingT, visits, expectedCount)
				require.Equal(testingT, scenario.storedIP, visits[0].IP)
			} else {
				require.Empty(testingT, visits)
			}
			statsResponse, err := server.Client().Get(server.URL + "/api/sites/" + site.ID + "/visits/stats?interval=all")
			require.NoError(testingT, err)
			defer statsResponse.Body.Close()
			require.Equal(testingT, http.StatusOK, statsResponse.StatusCode)
			var stats map[string]any
			require.NoError(testingT, json.NewDecoder(statsResponse.Body).Decode(&stats))
			require.Equal(testingT, float64(expectedCount), stats["visit_count"])
			require.Equal(testingT, float64(expectedCount), stats["unique_visitor_count"])
		})
	}
}

func TestTrafficIdentityHistoricalToolReports(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := insertSite(testingT, harness.database, "Historical audits", testVisitOrigin, testUserEmailAddress)
	sharedVisitor := storage.NewID()
	for index, userAgent := range []string{"Mozilla/5.0 Chrome/143.0", "Mozilla/5.0 HeadlessChrome/143.0", "Google-InspectionTool/1.0"} {
		visitorID := sharedVisitor
		if index == 2 {
			visitorID = storage.NewID()
		}
		visit, err := model.NewSiteVisit(model.SiteVisitInput{SiteID: site.ID, URL: testVisitOriginPage, VisitorID: visitorID, IP: "192.168.96.5", UserAgent: userAgent, Timezone: "America/Los_Angeles", Locale: "en-US", Occurred: time.Now().UTC()})
		require.NoError(testingT, err)
		require.NoError(testingT, harness.database.Create(&visit).Error)
	}
	router := gin.New()
	router.Use(func(context *gin.Context) {
		context.Set(testSessionContextKey, &api.CurrentUser{Email: testUserEmailAddress, Role: api.RoleUser})
	})
	router.GET("/api/sites/:id/visits/stats", harness.handlers.VisitStats)
	router.GET("/api/sites/:id/visits/locations", harness.handlers.LocationDistribution)
	server := httptest.NewServer(router)
	defer server.Close()
	checkReports := func(expectedViews, expectedUnique float64) {
		testingT.Helper()
		for _, endpoint := range []string{"stats", "locations"} {
			response, err := server.Client().Get(server.URL + "/api/sites/" + site.ID + "/visits/" + endpoint + "?interval=all")
			require.NoError(testingT, err)
			var payload map[string]any
			require.NoError(testingT, json.NewDecoder(response.Body).Decode(&payload))
			require.NoError(testingT, response.Body.Close())
			require.Equal(testingT, http.StatusOK, response.StatusCode)
			if endpoint == "locations" {
				rows := payload["locations"].([]any)
				require.Len(testingT, rows, 1)
				payload = rows[0].(map[string]any)
			}
			require.Equal(testingT, expectedViews, payload["visit_count"])
			require.Equal(testingT, expectedUnique, payload["unique_visitor_count"])
		}
	}
	checkReports(3, 2)
	changed, err := storage.MigrateBrowserAuditVisits(harness.database)
	require.NoError(testingT, err)
	require.Equal(testingT, int64(2), changed)
	checkReports(1, 1)
}

func TestTrafficIdentityAuditTools(testingT *testing.T) {
	for _, scenario := range []struct {
		userAgent string
		bot       bool
	}{
		{"Mozilla/5.0 HeadlessChrome/143.0.0.0 Safari/537.36", true},
		{"Mozilla/5.0 Google-InspectionTool/1.0", true},
		{"Mozilla/5.0 Chrome/143.0.0.0 Chrome-Lighthouse", true},
		{"Mozilla/5.0 Chrome/143.0.0.0 Safari/537.36", false},
		{testVisitWhatsAppUserAgent, false},
	} {
		testingT.Run(scenario.userAgent, func(testingT *testing.T) {
			harness := newSiteTestHarness(testingT)
			site := insertSite(testingT, harness.database, "Audit tool", testVisitOrigin, testUserEmailAddress)
			collector := api.NewPublicHandlers(harness.database, zap.NewNop(), nil, nil, nil, nil, false, testVisitOrigin, "traffic-identity-test", nil, api.VisitCollectionPolicy{})
			router := gin.New()
			router.Use(func(context *gin.Context) {
				context.Set(testSessionContextKey, &api.CurrentUser{Email: testUserEmailAddress, Role: api.RoleUser})
			})
			router.GET(testVisitPath, collector.CollectVisit)
			router.GET("/api/sites/:id/visits/stats", harness.handlers.VisitStats)
			router.GET("/api/sites/:id/visits/locations", harness.handlers.LocationDistribution)
			server := httptest.NewServer(router)
			defer server.Close()
			request, err := http.NewRequest(http.MethodGet, server.URL+testVisitQueryPrefix+site.ID+"&url="+url.QueryEscape(testVisitOriginPage)+"&visitor_id="+storage.NewID()+"&timezone=America%2FLos_Angeles&locale=en-US", nil)
			require.NoError(testingT, err)
			request.Header.Set("Origin", testVisitOrigin)
			request.Header.Set("User-Agent", scenario.userAgent)
			response, err := server.Client().Do(request)
			require.NoError(testingT, err)
			require.NoError(testingT, response.Body.Close())
			require.Equal(testingT, http.StatusOK, response.StatusCode)
			var stored model.SiteVisit
			require.NoError(testingT, harness.database.First(&stored).Error)
			require.Equal(testingT, scenario.bot, stored.IsBot)
			for _, endpoint := range []string{"stats", "locations"} {
				response, err := server.Client().Get(server.URL + "/api/sites/" + site.ID + "/visits/" + endpoint + "?interval=all")
				require.NoError(testingT, err)
				var payload map[string]any
				require.NoError(testingT, json.NewDecoder(response.Body).Decode(&payload))
				require.NoError(testingT, response.Body.Close())
				require.Equal(testingT, http.StatusOK, response.StatusCode)
				expectedCount := float64(1)
				if scenario.bot {
					expectedCount = 0
				}
				if endpoint == "stats" {
					require.Equal(testingT, expectedCount, payload["visit_count"])
					require.Equal(testingT, expectedCount, payload["unique_visitor_count"])
				} else if scenario.bot {
					require.Empty(testingT, payload["locations"])
				} else {
					require.Len(testingT, payload["locations"], 1)
				}
			}
		})
	}
}
