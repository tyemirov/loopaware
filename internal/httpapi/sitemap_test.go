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
	sitemapLandingLocationToken = "<loc>https://loopaware.mprlab.com/login</loc>"
	sitemapPrivacyLocationToken = "<loc>https://loopaware.mprlab.com/privacy</loc>"
)

func TestSitemapIncludesLandingAndPrivacyRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)

	handlers := httpapi.NewSitemapHandlers("https://loopaware.mprlab.com/")
	handlers.RenderSitemap(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/xml")
	body := recorder.Body.String()
	require.Contains(t, body, sitemapLandingLocationToken)
	require.Contains(t, body, sitemapPrivacyLocationToken)
}
