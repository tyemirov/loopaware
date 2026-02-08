package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/httpapi"
)

const (
	landingDetailedAuthCopyToken        = "Google Sign-In secured by TAuth keeps every login protected."
	landingDetailedWidgetCopyToken      = "Origin-locked widgets and APIs capture feedback where customers already are."
	landingDetailedWorkflowCopyToken    = "Role-aware workflows assign owners, surface trends, and track resolution."
	landingThemeScriptKeyToken          = "var publicThemeStorageKey = 'loopaware_public_theme'"
	landingThemeFallbackKeyToken        = "var landingThemeStorageKey = 'loopaware_landing_theme'"
	landingThemeFallbackLoadToken       = "var landingStoredTheme = localStorage.getItem(landingThemeStorageKey);"
	landingThemeLegacyKeyToken          = "var legacyThemeStorageKey = 'landing_theme'"
	landingThemeMigrationToken          = "var legacyStoredTheme = localStorage.getItem(legacyThemeStorageKey);"
	landingThemeApplyFunctionToken      = "function applyPublicTheme(theme)"
	landingThemeDataAttributeToken      = "data-bs-theme"
	landingThemeFooterQueryToken        = "var footerElement = document.querySelector('mpr-footer');"
	landingThemeFooterListenerToken     = "footerElement.addEventListener('mpr-footer:theme-change'"
	landingThemeFooterConfigToken       = "footerElement.setAttribute('theme-config', JSON.stringify(config));"
	landingLogoImageClassToken          = "class=\"landing-logo-image\""
	landingLogoAltToken                 = "alt=\"LoopAware logo\""
	landingLogoDataToken                = "src=\"data:image/png;base64,"
	landingLogoImageWidthToken          = "width: 48px;"
	landingLogoImageHeightToken         = "height: 48px;"
	landingLogoLegacyWidthToken         = "width: 36px;"
	landingLogoLegacyDarkBackground     = "background-color: rgba(59, 130, 246, 0.18);"
	landingLogoLegacyLightBackground    = "background-color: rgba(37, 99, 235, 0.12);"
	landingHeaderStickyToken            = "<mpr-header class=\"landing-header\""
	landingFooterComponentToken         = "<mpr-footer"
	landingFooterThemeSwitcherToken     = "theme-switcher=\"toggle\""
	landingFooterThemeConfigToken       = "theme-config="
	landingFooterStickyDisabledToken    = "sticky=\"false\""
	landingFooterLinksToken             = "links-collection="
	landingFooterLinkGravityToken       = "https://gravity.mprlab.com"
	landingFooterLinkLoopAwareToken     = "https://loopaware.mprlab.com"
	landingFooterLinkAllergyToken       = "https://allergy.mprlab.com"
	landingFooterLinkThreaderToken      = "https://threader.mprlab.com"
	landingFooterLinkRSVPToken          = "https://rsvp.mprlab.com"
	landingFooterLinkCountdownToken     = "https://countdown.mprlab.com"
	landingFooterLinkCrosswordToken     = "https://llm-crossword.mprlab.com"
	landingFooterLinkPromptsToken       = "https://prompts.mprlab.com"
	landingFooterLinkWallpapersToken    = "https://wallpapers.mprlab.com"
	landingFooterLayoutToken            = "mpr-footer__layout"
	landingFooterBrandWrapperToken      = "mpr-footer__brand"
	landingFooterPrivacyClassToken      = "mpr-footer__privacy"
	landingFooterPaddingToken           = "mpr-footer landing-footer border-top mt-auto py-2"
	landingCardHoverToken               = ".landing-card:hover"
	landingCardFocusToken               = ".landing-card:focus-visible"
	landingHeaderLoginPathToken         = "tauth-login-path=\"/auth/google\""
	landingHeaderNoncePathToken         = "tauth-nonce-path=\"/auth/nonce\""
	landingHeaderTauthTenantToken       = "tauth-tenant-id=\""
	landingHeaderGoogleClientToken      = "google-site-id=\""
	landingHeaderStickyStyleToken       = ".landing-header {\n        position: sticky;\n        top: 0;\n        z-index: 1030;\n        padding: 0;"
	landingHeaderHeroDataToken          = "data-public-hero=\"true\""
	landingHeaderHeroScrollToken        = "data-scroll-to-top=\"true\""
	landingHeaderHeroScrollHrefToken    = "href=\"#top\""
	landingHeaderHeroDashboardHrefToken = "href=\"/app\""
	landingHeaderHeroLandingHrefToken   = "href=\"/login\""
	landingFaviconLinkToken             = "<link rel=\"icon\" type=\"image/svg+xml\" href=\"data:image/svg&#43;xml"
	landingFooterPrivacyHrefToken       = "privacy-link-href=\"/privacy\""
	landingFooterPrivacyLabelToken      = "privacy-link-label=\"Privacy • Terms\""
	testAuthGoogleClientID              = "test-google-client-id"
	testAuthTauthBaseURL                = "https://tauth.example.com"
	testAuthTauthTenantID               = "test-tenant"
)

var testLandingAuthConfig = httpapi.NewAuthClientConfig(testAuthGoogleClientID, testAuthTauthBaseURL, testAuthTauthTenantID)

func TestLandingPageIncludesDetailedCopy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{}, testLandingAuthConfig, "")
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

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{}, testLandingAuthConfig, "")
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingFaviconLinkToken)
}

func TestLandingPageProvidesThemeSwitch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{}, testLandingAuthConfig, "")
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingThemeScriptKeyToken)
	require.Contains(t, body, landingThemeFallbackKeyToken)
	require.Contains(t, body, landingThemeFallbackLoadToken)
	require.Contains(t, body, landingThemeLegacyKeyToken)
	require.Contains(t, body, landingThemeMigrationToken)
	require.Contains(t, body, landingThemeApplyFunctionToken)
	require.Contains(t, body, landingThemeDataAttributeToken)
	require.Contains(t, body, landingThemeFooterQueryToken)
	require.Contains(t, body, landingThemeFooterListenerToken)
	require.Contains(t, body, landingThemeFooterConfigToken)
	require.Contains(t, body, landingHeaderStickyToken)
	require.Contains(t, body, landingFooterComponentToken)
	require.Contains(t, body, landingFooterThemeSwitcherToken)
}

func TestLandingHeaderProvidesStickyStyles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{}, testLandingAuthConfig, "")
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingHeaderStickyStyleToken)
}

func TestLandingPageDisplaysHeaderLogo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{}, testLandingAuthConfig, "")
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

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{}, testLandingAuthConfig, "")
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

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{}, testLandingAuthConfig, "")
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

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{authenticated: true}, testLandingAuthConfig, "")
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

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{}, testLandingAuthConfig, "")
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingFooterComponentToken)
	require.Contains(t, body, landingFooterLinksToken)
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
	require.Contains(t, body, landingFooterThemeConfigToken)
	require.Contains(t, body, landingFooterStickyDisabledToken)
}

func TestLandingPageDisplaysPrivacyLink(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{}, testLandingAuthConfig, "")
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingFooterPrivacyHrefToken)
	require.Contains(t, body, landingFooterPrivacyLabelToken)
	require.Contains(t, body, landingFooterPrivacyClassToken)
}
func TestLandingCardsProvideInteractiveStates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{}, testLandingAuthConfig, "")
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingCardHoverToken)
	require.Contains(t, body, landingCardFocusToken)
	require.Contains(t, body, "tabindex=\"0\"")
}

func TestLandingPageProvidesTauthHeaderConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop(), &stubCurrentUserProvider{}, testLandingAuthConfig, "")
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingHeaderLoginPathToken)
	require.Contains(t, body, landingHeaderNoncePathToken)
	require.Contains(t, body, landingHeaderTauthTenantToken)
	require.Contains(t, body, landingHeaderGoogleClientToken)
}
