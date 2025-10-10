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
	landingDetailedAuthCopyToken     = "Google Sign-In powered by GAuss keeps every login secure."
	landingDetailedWidgetCopyToken   = "Origin-locked widgets and APIs capture feedback where customers already are."
	landingDetailedWorkflowCopyToken = "Role-aware workflows assign owners, surface trends, and track resolution."
	landingThemeToggleIDToken        = "id=\"landing-theme-toggle\""
	landingThemeScriptKeyToken       = "var landingThemeStorageKey = 'landing_theme'"
	landingThemeApplyFunctionToken   = "function applyLandingTheme(theme)"
	landingThemeDataAttributeToken   = "data-bs-theme"
	landingHeaderLogoToken           = "aria-label=\"LoopAware logo\""
	landingFooterDropdownToggleToken = "data-bs-toggle=\"dropdown\""
	landingFooterDropdownMenuToken   = "dropdown-menu"
	landingFooterLinkGravityToken    = "https://gravity.mprlab.com"
	landingFooterLinkLoopAwareToken  = "https://loopaware.mprlab.com"
	landingFooterLinkAllergyToken    = "https://allergy.mprlab.com"
	landingFooterLinkThreaderToken   = "https://threader.mprlab.com"
	landingFooterLinkRSVPToken       = "https://rsvp.mprlab.com"
	landingFooterLinkCountdownToken  = "https://countdown.mprlab.com"
	landingFooterLinkCrosswordToken  = "https://llm-crossword.mprlab.com"
	landingFooterLinkPromptsToken    = "https://prompts.mprlab.com"
	landingFooterLinkWallpapersToken = "https://wallpapers.mprlab.com"
	landingFooterAlignmentToken      = "justify-content-md-end"
)

func TestLandingPageIncludesDetailedCopy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingDetailedAuthCopyToken)
	require.Contains(t, body, landingDetailedWidgetCopyToken)
	require.Contains(t, body, landingDetailedWorkflowCopyToken)
}

func TestLandingPageProvidesThemeSwitch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingThemeToggleIDToken)
	require.Contains(t, body, landingThemeScriptKeyToken)
	require.Contains(t, body, landingThemeApplyFunctionToken)
	require.Contains(t, body, landingThemeDataAttributeToken)
}

func TestLandingPageDisplaysHeaderLogo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingHeaderLogoToken)
}

func TestLandingFooterDisplaysProductMenu(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingFooterDropdownToggleToken)
	require.Contains(t, body, landingFooterDropdownMenuToken)
	require.Contains(t, body, landingFooterLinkGravityToken)
	require.Contains(t, body, landingFooterLinkLoopAwareToken)
	require.Contains(t, body, landingFooterLinkAllergyToken)
	require.Contains(t, body, landingFooterLinkThreaderToken)
	require.Contains(t, body, landingFooterLinkRSVPToken)
	require.Contains(t, body, landingFooterLinkCountdownToken)
	require.Contains(t, body, landingFooterLinkCrosswordToken)
	require.Contains(t, body, landingFooterLinkPromptsToken)
	require.Contains(t, body, landingFooterLinkWallpapersToken)
	require.Contains(t, body, landingFooterAlignmentToken)
}
