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
	landingDetailedAuthCopyToken        = "Google Sign-In powered by GAuss keeps every login secure."
	landingDetailedWidgetCopyToken      = "Origin-locked widgets and APIs capture feedback where customers already are."
	landingDetailedWorkflowCopyToken    = "Role-aware workflows assign owners, surface trends, and track resolution."
	landingThemeToggleIDToken           = "id=\"public-theme-toggle\""
	landingThemeScriptKeyToken          = "var publicThemeStorageKey = 'loopaware_public_theme'"
	landingThemeFallbackKeyToken        = "var landingThemeStorageKey = 'loopaware_landing_theme'"
	landingThemeFallbackLoadToken       = "var landingStoredTheme = localStorage.getItem(landingThemeStorageKey);"
	landingThemeLegacyKeyToken          = "var legacyThemeStorageKey = 'landing_theme'"
	landingThemeMigrationToken          = "var legacyStoredTheme = localStorage.getItem(legacyThemeStorageKey);"
	landingThemeApplyFunctionToken      = "function applyPublicTheme(theme)"
	landingThemeDataAttributeToken      = "data-bs-theme"
	landingLogoImageClassToken          = "class=\"landing-logo-image\""
	landingLogoAltToken                 = "alt=\"LoopAware logo\""
	landingLogoDataToken                = "src=\"data:image/png;base64,"
	landingLogoImageWidthToken          = "width: 48px;"
	landingLogoImageHeightToken         = "height: 48px;"
	landingLogoLegacyWidthToken         = "width: 36px;"
	landingLogoLegacyDarkBackground     = "background-color: rgba(59, 130, 246, 0.18);"
	landingLogoLegacyLightBackground    = "background-color: rgba(37, 99, 235, 0.12);"
	landingHeaderStickyToken            = "<header class=\"landing-header\">"
	landingFooterDropdownToggleToken    = "data-bs-toggle=\"dropdown\""
	landingFooterDropdownMenuToken      = "dropdown-menu"
	landingFooterLinkGravityToken       = "https://gravity.mprlab.com"
	landingFooterLinkLoopAwareToken     = "https://loopaware.mprlab.com"
	landingFooterLinkAllergyToken       = "https://allergy.mprlab.com"
	landingFooterLinkThreaderToken      = "https://threader.mprlab.com"
	landingFooterLinkRSVPToken          = "https://rsvp.mprlab.com"
	landingFooterLinkCountdownToken     = "https://countdown.mprlab.com"
	landingFooterLinkCrosswordToken     = "https://llm-crossword.mprlab.com"
	landingFooterLinkPromptsToken       = "https://prompts.mprlab.com"
	landingFooterLinkWallpapersToken    = "https://wallpapers.mprlab.com"
	landingFooterLayoutToken            = "footer-layout"
	landingFooterBrandWrapperToken      = "footer-brand d-inline-flex align-items-center"
	landingFooterPrivacyClassToken      = "footer-privacy-link text-body-secondary text-decoration-none small"
	landingFooterPaddingToken           = "landing-footer border-top mt-auto py-2"
	landingCardHoverToken               = ".landing-card:hover"
	landingCardFocusToken               = ".landing-card:focus-visible"
	landingHeroLoginButtonToken         = "btn btn-primary btn-lg\" href=\"/auth/google\">Login"
	landingHeaderLoginButtonToken       = "btn btn-primary btn-sm\" href=\"/auth/google\">Login"
	landingHeaderStickyStyleToken       = ".landing-header {\n        position: sticky;\n        top: 0;\n        z-index: 1030;\n        padding: 0;"
	landingHeaderHeroDataToken          = "data-public-hero=\"true\""
	landingHeaderHeroScrollToken        = "data-scroll-to-top=\"true\""
	landingHeaderHeroScrollHrefToken    = "href=\"#top\""
	landingHeaderHeroDashboardHrefToken = "href=\"/app\""
	landingHeaderHeroLandingHrefToken   = "href=\"/login\""
	landingFaviconLinkToken             = "<link rel=\"icon\" type=\"image/svg+xml\" href=\"data:image/svg&#43;xml"
	landingPrivacyLinkToken             = "href=\"/privacy\">Privacy • Terms"
)

func TestLandingPageIncludesDetailedCopy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{})
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

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{})
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingFaviconLinkToken)
}

func TestLandingPageProvidesThemeSwitch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{})
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingThemeToggleIDToken)
	require.Contains(t, body, landingThemeScriptKeyToken)
	require.Contains(t, body, landingThemeFallbackKeyToken)
	require.Contains(t, body, landingThemeFallbackLoadToken)
	require.Contains(t, body, landingThemeLegacyKeyToken)
	require.Contains(t, body, landingThemeMigrationToken)
	require.Contains(t, body, landingThemeApplyFunctionToken)
	require.Contains(t, body, landingThemeDataAttributeToken)
	require.Contains(t, body, landingHeaderStickyToken)
}

func TestLandingHeaderProvidesStickyStyles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{})
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingHeaderStickyStyleToken)
}

func TestLandingPageDisplaysHeaderLogo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{})
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

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{})
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingLogoImageWidthToken)
	require.Contains(t, body, landingLogoImageHeightToken)
	require.NotContains(t, body, landingLogoLegacyWidthToken)
	require.NotContains(t, body, landingLogoLegacyDarkBackground)
	require.NotContains(t, body, landingLogoLegacyLightBackground)
}

func TestLandingHeroScrollsToTopWhenUnauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{})
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingHeaderHeroDataToken)
	require.Contains(t, body, landingHeaderHeroScrollToken)
	require.Contains(t, body, landingHeaderHeroScrollHrefToken)
	require.NotContains(t, body, landingHeaderHeroDashboardHrefToken)
	require.NotContains(t, body, landingHeaderHeroLandingHrefToken)
}

func TestLandingHeroNavigatesToDashboardWhenAuthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{authenticated: true})
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingHeaderHeroDataToken)
	require.Contains(t, body, landingHeaderHeroDashboardHrefToken)
	require.NotContains(t, body, landingHeaderHeroScrollToken)
	require.NotContains(t, body, landingHeaderHeroScrollHrefToken)
	require.NotContains(t, body, landingHeaderHeroLandingHrefToken)
}

func TestLandingFooterDisplaysProductMenu(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{})
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
	require.Contains(t, body, landingFooterLayoutToken)
	require.Contains(t, body, landingFooterBrandWrapperToken)
	require.Contains(t, body, landingFooterPaddingToken)
}

func TestLandingPageDisplaysPrivacyLink(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{})
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingPrivacyLinkToken)
	require.Contains(t, body, landingFooterPrivacyClassToken)
}
func TestLandingCardsProvideInteractiveStates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{})
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

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{})
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingHeaderLoginButtonToken)
	require.NotContains(t, body, "View dashboard")
	require.NotContains(t, body, landingHeroLoginButtonToken)
}
