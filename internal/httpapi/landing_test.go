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
	landingLogoImageClassToken       = "class=\"landing-logo-image\""
	landingLogoAltToken              = "alt=\"LoopAware logo\""
	landingLogoDataToken             = "src=\"data:image/png;base64,"
	landingLogoContainerWidthToken   = "width: 56px;"
	landingLogoContainerHeightToken  = "height: 56px;"
	landingLogoImageWidthToken       = "width: 36px;"
	landingLogoImageHeightToken      = "height: 36px;"
	landingHeaderStickyToken         = "<header class=\"landing-header\">"
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
	landingFooterPaddingToken        = "landing-footer border-top mt-auto py-3"
	landingCardHoverToken            = ".landing-card:hover"
	landingCardFocusToken            = ".landing-card:focus-visible"
	landingHeroLoginButtonToken      = "btn btn-primary btn-lg\" href=\"/auth/google\">Login"
	landingHeaderLoginButtonToken    = "btn btn-primary btn-sm\" href=\"/auth/google\">Login"
	landingHeaderStickyStyleToken    = ".landing-header {\n        position: sticky;\n        top: 0;\n        z-index: 1030;"
	landingHeaderBrandAnchorToken    = "<a class=\"navbar-brand"
	landingHeaderBrandSpanToken      = "<span class=\"navbar-brand"
	landingFaviconLinkToken          = "<link rel=\"icon\" type=\"image/svg+xml\" href=\"data:image/svg&#43;xml"
)

func TestLandingPageIncludesDetailedCopy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingDetailedAuthCopyToken)
	require.Contains(t, body, landingDetailedWidgetCopyToken)
	require.Contains(t, body, landingDetailedWorkflowCopyToken)
}

func TestLandingPageExposesFaviconLink(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingFaviconLinkToken)
}

func TestLandingPageProvidesThemeSwitch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingThemeToggleIDToken)
	require.Contains(t, body, landingThemeScriptKeyToken)
	require.Contains(t, body, landingThemeApplyFunctionToken)
	require.Contains(t, body, landingThemeDataAttributeToken)
	require.Contains(t, body, landingHeaderStickyToken)
}

func TestLandingHeaderProvidesStickyStyles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingHeaderStickyStyleToken)
}


func TestLandingPageDisplaysHeaderLogo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingLogoImageClassToken)
	require.Contains(t, body, landingLogoAltToken)
	require.Contains(t, body, landingLogoDataToken)
}

func TestLandingPageLogoUsesProminentDimensions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingLogoContainerWidthToken)
	require.Contains(t, body, landingLogoContainerHeightToken)
	require.Contains(t, body, landingLogoImageWidthToken)
	require.Contains(t, body, landingLogoImageHeightToken)
}

func TestLandingLogoDoesNotTriggerNavigation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.NotContains(t, body, landingHeaderBrandAnchorToken)
	require.Contains(t, body, landingHeaderBrandSpanToken)
}

func TestLandingFooterDisplaysProductMenu(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

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
	require.Contains(t, body, landingFooterPaddingToken)
}

func TestLandingCardsProvideInteractiveStates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingCardHoverToken)
	require.Contains(t, body, landingCardFocusToken)
	require.Contains(t, body, "tabindex=\"0\"")
}

func TestLandingPageProvidesHeaderLoginOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingHeaderLoginButtonToken)
	require.NotContains(t, body, "View dashboard")
	require.NotContains(t, body, landingHeroLoginButtonToken)
}
