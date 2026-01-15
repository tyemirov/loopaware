package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/httpapi"
)

const (
	sitemapLandingLocationToken        = "<loc>https://loopaware.mprlab.com/login</loc>"
	sitemapPrivacyLocationToken        = "<loc>https://loopaware.mprlab.com/privacy</loc>"
	sitemapDefaultBaseURL              = "http://localhost:8080"
	sitemapDefaultLandingLocationToken = "<loc>" + sitemapDefaultBaseURL + "/login</loc>"
)

func TestSitemapIncludesLandingAndPrivacyRoutes(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)

	handlers := httpapi.NewSitemapHandlers("https://loopaware.mprlab.com/")
	handlers.RenderSitemap(context)

	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Contains(testingT, recorder.Header().Get("Content-Type"), "application/xml")
	body := recorder.Body.String()
	require.Contains(testingT, body, sitemapLandingLocationToken)
	require.Contains(testingT, body, sitemapPrivacyLocationToken)
}

func TestSitemapDefaultsToLocalhostBase(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, httpapi.SitemapRoutePath, nil)

	handlers := httpapi.NewSitemapHandlers("   ")
	handlers.RenderSitemap(context)

	body := recorder.Body.String()
	require.Contains(testingT, body, sitemapDefaultLandingLocationToken)
}

func TestSitemapDefaultsBaseURL(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)

	handlers := httpapi.NewSitemapHandlers(" ")
	handlers.RenderSitemap(context)

	require.Equal(testingT, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	require.Contains(testingT, body, "<loc>http://localhost:8080/login</loc>")
}
