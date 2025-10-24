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
			if (cells.length < 3) { continue; }
			if (cells[1].textContent.indexOf('auto@example.com') !== -1 && cells[2].textContent.indexOf('Auto refresh message') !== -1) {
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
	dashboardSeedPublicThemeScript      = `localStorage.setItem('loopaware_public_theme','dark');localStorage.removeItem('loopaware_dashboard_theme');localStorage.removeItem('loopaware_theme');`
	dashboardThemeToggleStateScript     = `(function(){var toggle=document.querySelector("#settings-theme-toggle");return !!(toggle && toggle.checked);}())`
	widgetTestSummaryOffsetScript       = `document.getElementById('widget-test-summary-offset') ? document.getElementById('widget-test-summary-offset').textContent : ''`
)

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
	screenshotsDirectory := createScreenshotsDirectory(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	currentPath := evaluateScriptString(t, page, dashboardLocationPathScript)
	require.Equal(t, dashboardTestDashboardRoute, currentPath)
	waitForVisibleElement(t, page, dashboardUserEmailSelector)

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

	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
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
	waitForVisibleElement(t, page, dashboardUserEmailSelector)

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

	navigateToPage(t, page, harness.baseURL+dashboardTestLandingPath)
	evaluateScriptInto(t, page, dashboardSeedPublicThemeScript, nil)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	waitForVisibleElement(t, page, dashboardUserEmailSelector)

	documentTheme := evaluateScriptString(t, page, dashboardDocumentThemeScript)
	require.Equal(t, "dark", documentTheme)

	toggleChecked := evaluateScriptBoolean(t, page, dashboardThemeToggleStateScript)
	require.True(t, toggleChecked)

	storedTheme := evaluateScriptString(t, page, dashboardStoredDashboardThemeScript)
	require.Equal(t, "dark", storedTheme)
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
	publicHandlers := httpapi.NewPublicHandlers(gormDatabase, logger, feedbackBroadcaster)

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
