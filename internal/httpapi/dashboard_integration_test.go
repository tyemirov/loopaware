package httpapi_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod/lib/proto"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/temirov/GAuss/pkg/constants"
	"github.com/temirov/GAuss/pkg/session"

	"github.com/MarkoPoloResearchLab/loopaware/internal/httpapi"
	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
	"github.com/MarkoPoloResearchLab/loopaware/pkg/favicon"
)

const (
	dashboardTestSessionSecretBytes       = "12345678901234567890123456789012"
	dashboardTestAdminEmail               = "admin@example.com"
	dashboardTestAdminDisplayName         = "Admin Example"
	dashboardTestWidgetBaseURL            = "http://example.test"
	dashboardTestLandingPath              = "/landing"
	dashboardTestDashboardRoute           = "/app"
	dashboardPromptWaitTimeout            = 10 * time.Second
	dashboardPromptPollInterval           = 200 * time.Millisecond
	dashboardNotificationSelector         = "#session-timeout-notification"
	dashboardDismissButtonSelector        = "#session-timeout-dismiss-button"
	dashboardConfirmButtonSelector        = "#session-timeout-confirm-button"
	dashboardSettingsButtonSelector       = "#settings-button"
	dashboardSettingsMenuSelector         = "#settings-menu"
	dashboardThemeToggleSelector          = "#settings-theme-toggle"
	dashboardPublicThemeToggleSelector    = "#public-theme-toggle"
	dashboardUserEmailSelector            = "#user-email"
	dashboardFooterSelector               = "#dashboard-footer"
	dashboardLightPromptScreenshotName    = "dashboard-session-timeout-light"
	dashboardDarkPromptScreenshotName     = "dashboard-session-timeout-dark"
	dashboardForcePromptScript            = "window.__loopawareDashboardIdleTestHooks.forcePrompt();"
	dashboardForceLogoutScript            = "window.__loopawareDashboardIdleTestHooks.forceLogout();"
	dashboardNotificationBackgroundScript = `window.getComputedStyle(document.querySelector("#session-timeout-notification")).backgroundColor`
	dashboardLocationPathScript           = "window.location.pathname"
	dashboardIdleHooksReadyScript         = "typeof window.__loopawareDashboardIdleTestHooks !== 'undefined'"
	dashboardPromptColorPresenceRatio     = colorPresenceMinimumRatio / 5.0
	dashboardSelectFirstSiteScript        = `(function() {
		var list = document.getElementById('sites-list');
		if (!list) { return false; }
		var item = list.querySelector('[data-site-id]');
		if (!item) { return false; }
		item.click();
		return true;
	}())`
	dashboardNoMessagesPlaceholderScript = `(function() {
		var body = document.getElementById('feedback-table-body');
		if (!body) { return false; }
		return body.textContent.indexOf('No feedback yet.') !== -1;
	}())`
	dashboardFeedbackRenderedScript = `(function() {
		var body = document.getElementById('feedback-table-body');
		if (!body) { return false; }
		var rows = body.querySelectorAll('tr');
		if (!rows.length) { return false; }
		for (var index = 0; index < rows.length; index++) {
			var cells = rows[index].querySelectorAll('td');
			if (cells.length < 4) { continue; }
			if (cells[1].textContent.indexOf('auto@example.com') !== -1 && cells[2].textContent.indexOf('Auto refresh message') !== -1 && cells[3].textContent.trim() === 'mailed') {
				return true;
			}
		}
		return false;
	}())`
	dashboardFeedbackCountScript = `(function() {
		var badge = document.getElementById('feedback-count');
		if (!badge) { return false; }
		if (badge.classList.contains('d-none')) { return false; }
		var textContent = (badge.textContent || '').trim();
		return textContent === '1';
	}())`
	dashboardDocumentThemeScript        = "document.documentElement.getAttribute('data-bs-theme') || ''"
	dashboardStoredDashboardThemeScript = "localStorage.getItem('loopaware_dashboard_theme') || ''"
	dashboardStoredPublicThemeScript    = "localStorage.getItem('loopaware_public_theme') || ''"
	dashboardSeedPublicThemeScript      = `localStorage.setItem('loopaware_public_theme','dark');localStorage.removeItem('loopaware_dashboard_theme');localStorage.removeItem('loopaware_theme');`
	dashboardThemeToggleStateScript     = `(function(){var toggle=document.querySelector("#settings-theme-toggle");return !!(toggle && toggle.checked);}())`
	widgetTestSummaryOffsetScript       = `document.getElementById('widget-test-summary-offset') ? document.getElementById('widget-test-summary-offset').textContent : ''`
)

type stubDashboardNotifier struct{}

func (stubDashboardNotifier) NotifyFeedback(ctx context.Context, site model.Site, feedback model.Feedback) (string, error) {
	return model.FeedbackDeliveryMailed, nil
}

type dashboardIntegrationHarness struct {
	router         *gin.Engine
	authManager    *httpapi.AuthManager
	faviconManager *httpapi.SiteFaviconManager
	database       *gorm.DB
	sqlDB          *sql.DB
	server         *httptest.Server
	baseURL        string
}

func TestDashboardSessionTimeoutPromptHonorsThemeAndLogout(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Scenario Site",
		AllowedOrigin: "https://widget.example",
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	var waitNavigation func()
	screenshotsDirectory := createScreenshotsDirectory(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(harness.baseURL+dashboardTestDashboardRoute))
	waitNavigation()
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	currentPath := evaluateScriptString(t, page, dashboardLocationPathScript)
	require.Equal(t, dashboardTestDashboardRoute, currentPath)
	userEmailVisibleScript := fmt.Sprintf(`(function(){
        var element = document.querySelector(%q);
        if (!element) { return false; }
        var style = window.getComputedStyle(element);
        if (!style) { return false; }
        return style.display !== 'none' && style.visibility !== 'hidden';
    }())`, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	waitForVisibleElement(t, page, dashboardNotificationSelector)

	lightBackgroundColor := evaluateScriptString(t, page, dashboardNotificationBackgroundScript)
	lightColor := mustParseRGBColor(t, lightBackgroundColor)

	lightPromptBounds := resolveViewportBounds(t, page, dashboardNotificationSelector)
	lightPromptScreenshot := captureAndStoreScreenshot(t, page, screenshotsDirectory, dashboardLightPromptScreenshotName)
	require.FileExists(t, filepath.Join(screenshotsDirectory, dashboardLightPromptScreenshotName+".png"))
	analyzeScreenshotRegion(t, lightPromptScreenshot, lightPromptBounds, screenshotExpectation{
		MinimumVariance: screenshotMinimumVariance,
		ColorPresence: []colorPresenceExpectation{
			{
				Color:        lightColor,
				Tolerance:    colorChannelTolerance,
				MinimumRatio: dashboardPromptColorPresenceRatio,
			},
		},
	}, float64(headlessViewportWidth), float64(headlessViewportHeight))
	footerBounds := resolveViewportBounds(t, page, dashboardFooterSelector)
	require.LessOrEqual(t, lightPromptBounds.Top+lightPromptBounds.Height, footerBounds.Top)

	clickSelector(t, page, dashboardSettingsButtonSelector)
	waitForVisibleElement(t, page, dashboardSettingsMenuSelector)
	clickSelector(t, page, dashboardThemeToggleSelector)
	clickSelector(t, page, "body")

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	waitForVisibleElement(t, page, dashboardNotificationSelector)

	darkBackgroundColor := evaluateScriptString(t, page, dashboardNotificationBackgroundScript)
	darkColor := mustParseRGBColor(t, darkBackgroundColor)
	require.NotEqual(t, lightColor, darkColor)

	darkPromptBounds := resolveViewportBounds(t, page, dashboardNotificationSelector)
	darkPromptScreenshot := captureAndStoreScreenshot(t, page, screenshotsDirectory, dashboardDarkPromptScreenshotName)
	require.FileExists(t, filepath.Join(screenshotsDirectory, dashboardDarkPromptScreenshotName+".png"))
	analyzeScreenshotRegion(t, darkPromptScreenshot, darkPromptBounds, screenshotExpectation{
		MinimumVariance: screenshotMinimumVariance,
		ColorPresence: []colorPresenceExpectation{
			{
				Color:        darkColor,
				Tolerance:    colorChannelTolerance,
				MinimumRatio: dashboardPromptColorPresenceRatio,
			},
		},
	}, float64(headlessViewportWidth), float64(headlessViewportHeight))
	require.LessOrEqual(t, darkPromptBounds.Top+darkPromptBounds.Height, footerBounds.Top)

	clickSelector(t, page, dashboardDismissButtonSelector)
	require.Eventually(t, func() bool {
		notificationElement, elementErr := page.Element(dashboardNotificationSelector)
		if elementErr != nil {
			return true
		}
		visible, visibleErr := notificationElement.Visible()
		if visibleErr != nil {
			return false
		}
		return !visible
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	waitForVisibleElement(t, page, dashboardNotificationSelector)

	waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	confirmClickScript := fmt.Sprintf("document.querySelector(%q).click()", dashboardConfirmButtonSelector)
	evaluateScriptInto(t, page, confirmClickScript, nil)
	waitNavigation()

	require.Eventually(t, func() bool {
		info, infoErr := page.Info()
		if infoErr != nil {
			return false
		}
		parsed, parseErr := url.Parse(info.URL)
		if parseErr != nil {
			return false
		}
		return parsed.Path == dashboardTestLandingPath
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardSessionTimeoutAutoLogout(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	userEmailVisibleScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		return style.display !== 'none' && style.visibility !== 'hidden';
	}())`, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	waitForVisibleElement(t, page, dashboardNotificationSelector)

	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	evaluateScriptInto(t, page, dashboardForceLogoutScript, nil)
	waitNavigation()

	require.Eventually(t, func() bool {
		info, infoErr := page.Info()
		if infoErr != nil {
			return false
		}
		parsed, parseErr := url.Parse(info.URL)
		if parseErr != nil {
			return false
		}
		return parsed.Path == dashboardTestLandingPath
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardRestoresThemeFromPublicPreference(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(harness.baseURL+dashboardTestLandingPath))
	waitNavigation()
	evaluateScriptInto(t, page, dashboardSeedPublicThemeScript, nil)

	waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(harness.baseURL+dashboardTestDashboardRoute))
	waitNavigation()
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	userEmailVisibleScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		return style.display !== 'none' && style.visibility !== 'hidden';
	}())`, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	documentTheme := evaluateScriptString(t, page, dashboardDocumentThemeScript)
	require.Equal(t, "dark", documentTheme)

	toggleChecked := evaluateScriptBoolean(t, page, dashboardThemeToggleStateScript)
	require.True(t, toggleChecked)

	storedTheme := evaluateScriptString(t, page, dashboardStoredDashboardThemeScript)
	require.Equal(t, "dark", storedTheme)
}

func TestDashboardPrefersPublicThemeOverStoredPreference(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	testCases := []struct {
		name             string
		storedTheme      string
		publicTheme      string
		expectedTheme    string
		expectedToggleOn bool
	}{
		{
			name:             "public_dark_overrides_stored_light",
			storedTheme:      "light",
			publicTheme:      "dark",
			expectedTheme:    "dark",
			expectedToggleOn: true,
		},
		{
			name:             "public_light_overrides_stored_dark",
			storedTheme:      "dark",
			publicTheme:      "light",
			expectedTheme:    "light",
			expectedToggleOn: false,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
			require.NoError(t, page.Navigate(harness.baseURL+dashboardTestLandingPath))
			waitNavigation()

			seedScript := fmt.Sprintf(`localStorage.setItem('loopaware_dashboard_theme','%s');localStorage.setItem('loopaware_public_theme','%s');localStorage.setItem('loopaware_landing_theme','%s');localStorage.removeItem('loopaware_theme');`, testCase.storedTheme, testCase.publicTheme, testCase.publicTheme)
			evaluateScriptInto(t, page, seedScript, nil)

			waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
			require.NoError(t, page.Navigate(harness.baseURL+dashboardTestDashboardRoute))
			waitNavigation()
			require.Eventually(t, func() bool {
				return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
			}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

			documentTheme := evaluateScriptString(t, page, dashboardDocumentThemeScript)
			require.Equal(t, testCase.expectedTheme, documentTheme)

			storedDashboardTheme := evaluateScriptString(t, page, dashboardStoredDashboardThemeScript)
			require.Equal(t, testCase.expectedTheme, storedDashboardTheme)

			toggleChecked := evaluateScriptBoolean(t, page, dashboardThemeToggleStateScript)
			require.Equal(t, testCase.expectedToggleOn, toggleChecked)
		})
	}
}

func TestWidgetTestPageUsesDashboardChrome(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Widget Test Chrome",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(harness.baseURL+dashboardTestLandingPath))
	waitNavigation()
	evaluateScriptInto(t, page, dashboardSeedPublicThemeScript, nil)

	widgetTestURL := fmt.Sprintf("%s/app/sites/%s/widget-test", harness.baseURL, site.ID)
	waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(widgetTestURL))
	waitNavigation()

	currentWidgetTestPath := evaluateScriptString(t, page, dashboardLocationPathScript)
	require.Equal(t, fmt.Sprintf("/app/sites/%s/widget-test", site.ID), currentWidgetTestPath)

	headerVisibleScript := fmt.Sprintf(`(function(){
    var element = document.querySelector(%q);
    if (!element) { return false; }
    var style = window.getComputedStyle(element);
    if (!style) { return false; }
    return style.display !== 'none' && style.visibility !== 'hidden';
}())`, dashboardSettingsButtonSelector)
	footerVisibleScript := fmt.Sprintf(`(function(){
    var element = document.querySelector(%q);
    if (!element) { return false; }
    var style = window.getComputedStyle(element);
    if (!style) { return false; }
    return style.display !== 'none' && style.visibility !== 'hidden';
}())`, dashboardFooterSelector)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, headerVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, footerVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	initialTheme := evaluateScriptString(t, page, dashboardDocumentThemeScript)
	require.Equal(t, "dark", initialTheme)

	initialDashboardStoredTheme := evaluateScriptString(t, page, dashboardStoredDashboardThemeScript)
	require.Equal(t, "dark", initialDashboardStoredTheme)

	initialPublicStoredTheme := evaluateScriptString(t, page, dashboardStoredPublicThemeScript)
	require.Equal(t, "dark", initialPublicStoredTheme)

	clickSelector(t, page, dashboardSettingsButtonSelector)
	waitForVisibleElement(t, page, dashboardSettingsMenuSelector)

	currentToggleChecked := evaluateScriptBoolean(t, page, dashboardThemeToggleStateScript)
	require.True(t, currentToggleChecked)

	clickSelector(t, page, dashboardThemeToggleSelector)
	clickSelector(t, page, "body")

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardDocumentThemeScript) == "light"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	finalDashboardStoredTheme := evaluateScriptString(t, page, dashboardStoredDashboardThemeScript)
	require.Equal(t, "light", finalDashboardStoredTheme)

	finalPublicStoredTheme := evaluateScriptString(t, page, dashboardStoredPublicThemeScript)
	require.Equal(t, "light", finalPublicStoredTheme)
}

func TestDashboardFeedbackStreamRefreshesMessages(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Stream Feedback Site",
		AllowedOrigin:              harness.baseURL,
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 16,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	waitForVisibleElement(t, page, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, 5*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardNoMessagesPlaceholderScript)
	}, 5*time.Second, 100*time.Millisecond)

	requestBody := fmt.Sprintf(`{"site_id":"%s","contact":"auto@example.com","message":"Auto refresh message"}`, site.ID)
	request, err := http.NewRequest(http.MethodPost, harness.baseURL+"/api/feedback", bytes.NewBufferString(requestBody))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", site.AllowedOrigin)
	response, err := harness.server.Client().Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, response.Body.Close())

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardFeedbackRenderedScript)
	}, 10*time.Second, 200*time.Millisecond)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardFeedbackCountScript)
	}, 5*time.Second, 100*time.Millisecond)
}

func TestExampleRouteIsUnavailable(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	response, err := http.Get(harness.baseURL + "/example")
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

func buildDashboardIntegrationHarness(testingT *testing.T, adminEmail string) *dashboardIntegrationHarness {
	testingT.Helper()

	session.NewSession([]byte(dashboardTestSessionSecretBytes))

	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	gormDatabase, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	gormDatabase = testutil.ConfigureDatabaseLogger(testingT, gormDatabase)
	require.NoError(testingT, storage.AutoMigrate(gormDatabase))
	sqlDatabase, sqlErr := gormDatabase.DB()
	require.NoError(testingT, sqlErr)

	adminEmails := []string{adminEmail}
	authManager := httpapi.NewAuthManager(gormDatabase, logger, adminEmails, nil, dashboardTestLandingPath)

	noopResolver := &staticFaviconResolver{}
	faviconService := favicon.NewService(noopResolver)
	faviconManager := httpapi.NewSiteFaviconManager(gormDatabase, faviconService, logger)
	managerContext, managerCancel := context.WithCancel(context.Background())
	faviconManager.Start(managerContext)

	feedbackBroadcaster := httpapi.NewFeedbackEventBroadcaster()

	siteHandlers := httpapi.NewSiteHandlers(gormDatabase, logger, dashboardTestWidgetBaseURL, faviconManager, nil, feedbackBroadcaster)
	landingHandlers := httpapi.NewLandingPageHandlers(logger, authManager)
	privacyHandlers := httpapi.NewPrivacyPageHandlers(authManager)
	sitemapHandlers := httpapi.NewSitemapHandlers(dashboardTestWidgetBaseURL)
	dashboardHandlers := httpapi.NewDashboardWebHandlers(logger, dashboardTestLandingPath)
	publicHandlers := httpapi.NewPublicHandlers(gormDatabase, logger, feedbackBroadcaster, stubDashboardNotifier{})
	widgetTestHandlers := httpapi.NewSiteWidgetTestHandlers(gormDatabase, logger, dashboardTestWidgetBaseURL, feedbackBroadcaster)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpapi.RequestLogger(logger))

	router.GET("/", func(context *gin.Context) {
		context.Redirect(http.StatusFound, dashboardTestLandingPath)
	})
	router.GET(dashboardTestLandingPath, landingHandlers.RenderLandingPage)
	router.GET(constants.LoginPath, landingHandlers.RenderLandingPage)
	router.GET(httpapi.PrivacyPagePath, privacyHandlers.RenderPrivacyPage)
	router.GET(httpapi.SitemapRoutePath, sitemapHandlers.RenderSitemap)
	router.GET(dashboardTestDashboardRoute, authManager.RequireAuthenticatedWeb(), dashboardHandlers.RenderDashboard)
	router.GET("/app/sites/:id/widget-test", authManager.RequireAuthenticatedWeb(), widgetTestHandlers.RenderWidgetTestPage)
	router.POST("/app/sites/:id/widget-test/feedback", authManager.RequireAuthenticatedJSON(), widgetTestHandlers.SubmitWidgetTestFeedback)
	router.POST(constants.LogoutPath, func(context *gin.Context) {
		context.Status(http.StatusOK)
	})

	router.POST("/api/feedback", publicHandlers.CreateFeedback)
	router.GET("/widget.js", publicHandlers.WidgetJS)

	apiGroup := router.Group("/api")
	apiGroup.Use(authManager.RequireAuthenticatedJSON())
	apiGroup.GET("/me", siteHandlers.CurrentUser)
	apiGroup.GET("/me/avatar", siteHandlers.UserAvatar)
	apiGroup.GET("/sites", siteHandlers.ListSites)
	apiGroup.GET("/sites/:id/messages", siteHandlers.ListMessagesBySite)
	apiGroup.GET("/sites/:id/favicon", siteHandlers.SiteFavicon)
	apiGroup.GET("/sites/favicons/events", siteHandlers.StreamFaviconUpdates)
	apiGroup.GET("/sites/feedback/events", siteHandlers.StreamFeedbackUpdates)

	server := httptest.NewServer(router)

	testingT.Cleanup(func() {
		managerCancel()
		faviconManager.Stop()
		feedbackBroadcaster.Close()
		server.Close()
		require.NoError(testingT, sqlDatabase.Close())
	})

	return &dashboardIntegrationHarness{
		router:         router,
		authManager:    authManager,
		faviconManager: faviconManager,
		database:       gormDatabase,
		sqlDB:          sqlDatabase,
		server:         server,
		baseURL:        server.URL,
	}
}

func (harness *dashboardIntegrationHarness) Close() {
	// cleanup handled via testing.Cleanup
}

type staticFaviconResolver struct{}

func (resolver *staticFaviconResolver) Resolve(ctx context.Context, allowedOrigin string) (string, error) {
	return "", nil
}

func (resolver *staticFaviconResolver) ResolveAsset(ctx context.Context, allowedOrigin string) (*favicon.Asset, error) {
	return nil, nil
}

func createAuthenticatedSessionCookie(testingT *testing.T, email string, name string) *http.Cookie {
	testingT.Helper()

	store := session.Store()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	sessionInstance, err := store.Get(request, constants.SessionName)
	require.NoError(testingT, err)

	sessionInstance.Values[constants.SessionKeyUserEmail] = email
	sessionInstance.Values[constants.SessionKeyUserName] = name
	sessionInstance.Values[constants.SessionKeyUserPicture] = ""
	sessionInstance.Values[constants.SessionKeyOAuthToken] = "test-token"

	require.NoError(testingT, sessionInstance.Save(request, recorder))

	response := recorder.Result()
	for _, cookie := range response.Cookies() {
		if cookie.Name == constants.SessionName {
			return cookie
		}
	}
	require.FailNow(testingT, "session cookie not found in recorder")
	return nil
}
