package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/httpapi"
)

const (
	landingPageTitleToken            = "LoopAware • Close the feedback loop"
	landingPageHeroHeadingToken      = "Customer feedback you can act on"
	landingPageHeroSubheadingToken   = "LoopAware captures every signal in one workspace."
	landingPageDashboardLinkToken    = `href="/app"`
	landingPageLoginLinkToken        = `href="/auth/google"`
	landingPageFooterDropupClass     = "dropup"
	landingPageFooterDropdownMenu    = "dropdown-menu"
	landingPageFooterGravityNotesURL = "https://gravity.mprlab.com"
)

func TestLandingPageRendersHeroContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), landingPageTitleToken)
	require.Contains(t, recorder.Body.String(), landingPageHeroHeadingToken)
	require.Contains(t, recorder.Body.String(), landingPageHeroSubheadingToken)
}

func TestLandingPageIncludesNavigationLinks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingPageDashboardLinkToken)
	require.Contains(t, body, landingPageLoginLinkToken)
}

func TestLandingPageReusesFooterDropup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingPageFooterDropupClass)
	require.Contains(t, body, landingPageFooterDropdownMenu)
	require.Contains(t, body, landingPageFooterGravityNotesURL)
}
