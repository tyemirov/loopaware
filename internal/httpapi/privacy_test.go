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
	privacyTitleToken          = "<title>Privacy Policy — LoopAware</title>"
	privacyHeadingToken        = "<h1 class=\"privacy-heading\">Privacy Policy — LoopAware</h1>"
	privacyEffectiveDateToken  = "<strong>Effective Date:</strong> 2025-10-11"
	privacyMetaRobotsToken     = "<meta name=\"robots\" content=\"noindex,nofollow\" />"
	privacySupportEmailToken   = "mailto:support@mprlab.com"
	privacyBodyFontStyleToken  = "body{font:16px/1.5 system-ui,Segoe UI,Roboto,Helvetica,Arial,sans-serif"
	privacyGeneratorCommentTag = "<!doctype html>"
	privacyFooterLayoutToken   = "mpr-footer__layout"
	privacyFooterBrandToken    = "mpr-footer__brand"
	privacyFooterPrivacyToken  = "mpr-footer__privacy"
)

func TestPrivacyPageRendersPolicyMarkup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/privacy", nil)

	handlers := httpapi.NewPrivacyPageHandlers(&stubCurrentUserProvider{}, testLandingAuthConfig)
	handlers.RenderPrivacyPage(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/html")
	body := recorder.Body.String()
	require.Contains(t, body, privacyGeneratorCommentTag)
	require.Contains(t, body, privacyTitleToken)
	require.Contains(t, body, privacyHeadingToken)
	require.Contains(t, body, privacyEffectiveDateToken)
	require.Contains(t, body, privacyMetaRobotsToken)
	require.Contains(t, body, privacySupportEmailToken)
	require.Contains(t, body, privacyBodyFontStyleToken)
	require.Contains(t, body, landingHeaderStickyToken)
	require.Contains(t, body, landingThemeScriptKeyToken)
	require.Contains(t, body, landingThemeFallbackKeyToken)
	require.Contains(t, body, landingThemeFallbackLoadToken)
	require.Contains(t, body, landingThemeLegacyKeyToken)
	require.Contains(t, body, landingThemeApplyFunctionToken)
	require.Contains(t, body, landingThemeDataAttributeToken)
	require.Contains(t, body, landingThemeFooterQueryToken)
	require.Contains(t, body, landingThemeFooterListenerToken)
	require.Contains(t, body, landingThemeFooterModeToken)
	require.Contains(t, body, landingFooterComponentToken)
	require.Contains(t, body, landingFooterThemeSwitcherToken)
	require.Contains(t, body, landingFooterThemeConfigToken)
	require.Contains(t, body, landingFooterStickyDisabledToken)
	require.Contains(t, body, privacyFooterLayoutToken)
	require.Contains(t, body, privacyFooterBrandToken)
	require.Contains(t, body, privacyFooterPrivacyToken)
	require.Contains(t, body, dashboardFooterBrandPrefix)
	require.Contains(t, body, dashboardFooterBrandName)
}

func TestPrivacyHeroNavigatesToLandingWhenUnauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/privacy", nil)

	handlers := httpapi.NewPrivacyPageHandlers(&stubCurrentUserProvider{}, testLandingAuthConfig)
	handlers.RenderPrivacyPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingHeaderHeroDataToken)
	require.Contains(t, body, landingHeaderHeroLandingHrefToken)
	require.NotContains(t, body, landingHeaderHeroScrollToken)
	require.NotContains(t, body, landingHeaderHeroScrollHrefToken)
	require.NotContains(t, body, landingHeaderHeroDashboardHrefToken)
}

func TestPrivacyHeroNavigatesToDashboardWhenAuthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/privacy", nil)

	handlers := httpapi.NewPrivacyPageHandlers(&stubCurrentUserProvider{authenticated: true}, testLandingAuthConfig)
	handlers.RenderPrivacyPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingHeaderHeroDataToken)
	require.Contains(t, body, landingHeaderHeroDashboardHrefToken)
	require.NotContains(t, body, landingHeaderHeroScrollToken)
	require.NotContains(t, body, landingHeaderHeroScrollHrefToken)
	require.NotContains(t, body, landingHeaderHeroLandingHrefToken)
}
