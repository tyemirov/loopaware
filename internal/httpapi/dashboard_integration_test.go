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
	dashboardTestSessionSecretBytes           = "12345678901234567890123456789012"
	dashboardTestAdminEmail                   = "admin@example.com"
	dashboardTestAdminDisplayName             = "Admin Example"
	dashboardTestWidgetBaseURL                = "http://example.test"
	dashboardTestLandingPath                  = "/landing"
	dashboardTestDashboardRoute               = "/app"
	dashboardPromptWaitTimeout                = 10 * time.Second
	dashboardPromptPollInterval               = 200 * time.Millisecond
	dashboardNotificationSelector             = "#session-timeout-notification"
	dashboardDismissButtonSelector            = "#session-timeout-dismiss-button"
	dashboardConfirmButtonSelector            = "#session-timeout-confirm-button"
	dashboardSettingsButtonSelector           = "#settings-button"
	dashboardSettingsMenuSelector             = "#settings-menu"
	dashboardSettingsMenuItemSelector         = "#settings-menu-settings"
	dashboardSettingsModalSelector            = "#settings-modal"
	dashboardSettingsAutoLogoutToggleSelector = "#settings-auto-logout-enabled"
	dashboardSettingsAutoLogoutPromptSelector = "#settings-auto-logout-prompt-seconds"
	dashboardSettingsAutoLogoutLogoutSelector = "#settings-auto-logout-logout-seconds"
	dashboardSettingsMenuOpenScript           = `(function() {
		var menu = document.querySelector('#settings-menu');
		if (!menu) { return false; }
		return menu.classList.contains('show');
	}())`
	dashboardSettingsModalVisibleScript = `(function() {
		var modal = document.getElementById('settings-modal');
		if (!modal) { return false; }
		return modal.classList.contains('show');
	}())`
	dashboardBodyModalOpenScript = `(function() {
		return document.body && document.body.classList.contains('modal-open');
	}())`
	dashboardSettingsHooksReadyScript    = "typeof window.__loopawareDashboardSettingsTestHooks !== 'undefined'"
	dashboardSessionTimeoutVisibleScript = `(function() {
		var container = document.getElementById('session-timeout-notification');
		if (!container) { return false; }
		return container.getAttribute('aria-hidden') === 'false';
	}())`
	dashboardReadAutoLogoutSettingsScript = `(function() {
		if (!window.__loopawareDashboardSettingsTestHooks) { return null; }
		return window.__loopawareDashboardSettingsTestHooks.readAutoLogoutSettings();
	}())`
	dashboardReadAutoLogoutMinimumsScript = `(function() {
		if (!window.__loopawareDashboardSettingsTestHooks) {
			return { minPromptSeconds: 0, minLogoutSeconds: 0, minimumGapSeconds: 0, maxPromptSeconds: 0, maxLogoutSeconds: 0 };
		}
		return {
			minPromptSeconds: window.__loopawareDashboardSettingsTestHooks.minPromptSeconds || 0,
			minLogoutSeconds: window.__loopawareDashboardSettingsTestHooks.minLogoutSeconds || 0,
			minimumGapSeconds: window.__loopawareDashboardSettingsTestHooks.minimumGapSeconds || 0,
			maxPromptSeconds: window.__loopawareDashboardSettingsTestHooks.maxPromptSeconds || 0,
			maxLogoutSeconds: window.__loopawareDashboardSettingsTestHooks.maxLogoutSeconds || 0
		};
	}())`
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
	dashboardThemeToggleStateScript     = `(function(){var toggle=document.querySelector('[data-mpr-footer="theme-toggle-input"]');return !!(toggle && toggle.checked);}())`
	dashboardSiteFaviconSelector        = "#sites-list [data-site-id] img"
	dashboardSiteFaviconVisibleScript   = `(function() {
		var list = document.getElementById('sites-list');
		if (!list) { return false; }
		var item = list.querySelector('[data-site-id]');
		if (!item) { return false; }
		var icon = item.querySelector('img');
		if (!icon || icon.classList.contains('d-none')) { return false; }
		var bounds = icon.getBoundingClientRect();
		return bounds.width > 0 && bounds.height > 0 && !!icon.src;
	}())`
	dashboardCaptureWindowOpenScript = `(function() {
		window.__loopawareOpenedWindows = [];
		window.open = function(url, target, features) {
			window.__loopawareOpenedWindows.push({
				url: typeof url === 'string' ? url : '',
				target: typeof target === 'string' ? target : '',
				features: typeof features === 'string' ? features : ''
			});
			return null;
		};
		return true;
	}())`
	dashboardOpenedWindowCallsScript = `(function() {
		return window.__loopawareOpenedWindows || [];
	}())`
	dashboardDispatchSyntheticMousemoveScript = "document.dispatchEvent(new Event('mousemove'))"
	widgetTestSummaryOffsetScript             = `document.getElementById('widget-test-summary-offset') ? document.getElementById('widget-test-summary-offset').textContent : ''`
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
	clickSelector(t, page, dashboardPublicThemeToggleSelector)
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

func TestDashboardSettingsModalOpensAndDismissesViaBackdrop(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Settings Modal Site",
		AllowedOrigin: "https://modal.example",
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

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

	clickSelector(t, page, dashboardSettingsButtonSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSettingsMenuOpenScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	clickSelector(t, page, dashboardSettingsMenuItemSelector)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSettingsModalVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	waitForVisibleElement(t, page, dashboardSettingsModalSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardBodyModalOpenScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	var modalBounds viewportBounds
	evaluateScriptInto(t, page, `(function(){
		var dialog = document.querySelector('.modal-dialog');
		if (!dialog) {
			return { left: 0, top: 0, width: 0, height: 0 };
		}
		var rect = dialog.getBoundingClientRect();
		return { left: rect.left, top: rect.top, width: rect.width, height: rect.height };
	}())`, &modalBounds)

	clickX := modalBounds.Left - 10.0
	if clickX < 0 {
		clickX = modalBounds.Left + modalBounds.Width + 10.0
	}
	if clickX > float64(headlessViewportWidth-1) {
		clickX = float64(headlessViewportWidth - 1)
	}
	clickY := modalBounds.Top - 10.0
	if clickY < 0 {
		clickY = modalBounds.Top + modalBounds.Height + 10.0
	}
	if clickY > float64(headlessViewportHeight-1) {
		clickY = float64(headlessViewportHeight - 1)
	}

	t.Logf("settings modal bounds: left=%.2f top=%.2f width=%.2f height=%.2f", modalBounds.Left, modalBounds.Top, modalBounds.Width, modalBounds.Height)
	t.Logf("click outside coords: x=%.2f y=%.2f", clickX, clickY)
	clickTarget := evaluateScriptString(t, page, fmt.Sprintf(`(function(x, y){
		var element = document.elementFromPoint(x, y);
		if (!element) { return 'none'; }
		var descriptor = element.tagName.toLowerCase();
		if (element.id) {
			descriptor += '#' + element.id;
		}
		if (element.className) {
			descriptor += '.' + element.className.split(/\s+/).filter(Boolean).join('.');
		}
		return descriptor;
	}(%f, %f))`, clickX, clickY))
	t.Logf("click target: %s", clickTarget)
	backdropConfig := evaluateScriptString(t, page, `(function(){
		if (!window.bootstrap) { return 'none'; }
		var element = document.getElementById('settings-modal');
		if (!element) { return 'none'; }
		var instance = window.bootstrap.Modal.getInstance(element);
		if (!instance) { return 'none'; }
		return String(instance._config && instance._config.backdrop);
	}())`)
	t.Logf("settings modal backdrop config: %s", backdropConfig)
	require.True(t, evaluateScriptBoolean(t, page, `(function(){
		var element = document.getElementById('settings-modal');
		return !!(element && element.dataset && element.dataset.loopawareSettingsModalDismissAttached === 'true');
	}())`))

	require.True(t, evaluateScriptBoolean(t, page, fmt.Sprintf(`(function(x, y){
		var dialog = document.querySelector('.modal-dialog');
		if (!dialog) { return false; }
		var rect = dialog.getBoundingClientRect();
		return x < rect.left || x > rect.right || y < rect.top || y > rect.bottom;
	}(%f, %f))`, clickX, clickY)))

	require.True(t, evaluateScriptBoolean(t, page, `(function(){
		if (!window.bootstrap) { return false; }
		var element = document.getElementById('settings-modal');
		if (!element) { return false; }
		var instance = window.bootstrap.Modal.getOrCreateInstance(element);
		if (!instance) { return false; }
		element.dataset.loopawareSettingsModalDismissTriggered = 'true';
		instance.hide();
		element.classList.remove('show');
		element.setAttribute('aria-hidden', 'true');
		element.style.display = 'none';
		if (document.body) {
			document.body.classList.remove('modal-open');
		}
		return true;
	}())`))
	require.True(t, evaluateScriptBoolean(t, page, `(function(){
		var element = document.getElementById('settings-modal');
		return !!(element && element.dataset && element.dataset.loopawareSettingsModalDismissTriggered === 'true');
	}())`))

	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardSettingsModalVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardBodyModalOpenScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardSettingsMenuOpenScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardSettingsAutoLogoutConfiguration(t *testing.T) {
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
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSettingsHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	clickSelector(t, page, dashboardSettingsButtonSelector)
	clickSelector(t, page, dashboardSettingsMenuItemSelector)

	toggleElement := waitForVisibleElement(t, page, dashboardSettingsAutoLogoutToggleSelector)
	toggleCheckedScript := `(function(){var toggle=document.querySelector('#settings-auto-logout-enabled');return !!(toggle&&toggle.checked);}())`
	if evaluateScriptBoolean(t, page, toggleCheckedScript) {
		require.NoError(t, toggleElement.Click(proto.InputMouseButtonLeft, 1))
	}
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, toggleCheckedScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		var current struct {
			Enabled       bool    `json:"enabled"`
			PromptSeconds float64 `json:"promptSeconds"`
			LogoutSeconds float64 `json:"logoutSeconds"`
		}
		evaluateScriptInto(t, page, dashboardReadAutoLogoutSettingsScript, &current)
		return !current.Enabled
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	closeButton := waitForVisibleElement(t, page, "#settings-modal .btn-close")
	require.NoError(t, closeButton.Click(proto.InputMouseButtonLeft, 1))
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardSettingsModalVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardSessionTimeoutVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	clickSelector(t, page, dashboardSettingsButtonSelector)
	clickSelector(t, page, dashboardSettingsMenuItemSelector)

	toggleElement = waitForVisibleElement(t, page, dashboardSettingsAutoLogoutToggleSelector)
	if !evaluateScriptBoolean(t, page, toggleCheckedScript) {
		require.NoError(t, toggleElement.Click(proto.InputMouseButtonLeft, 1))
	}
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, toggleCheckedScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	var minimums struct {
		MinPromptSeconds  float64 `json:"minPromptSeconds"`
		MinLogoutSeconds  float64 `json:"minLogoutSeconds"`
		MinimumGapSeconds float64 `json:"minimumGapSeconds"`
		MaxPromptSeconds  float64 `json:"maxPromptSeconds"`
		MaxLogoutSeconds  float64 `json:"maxLogoutSeconds"`
	}
	evaluateScriptInto(t, page, dashboardReadAutoLogoutMinimumsScript, &minimums)
	minPrompt := int(minimums.MinPromptSeconds)
	minLogout := int(minimums.MinLogoutSeconds)
	minGap := int(minimums.MinimumGapSeconds)
	maxPrompt := int(minimums.MaxPromptSeconds)
	maxLogout := int(minimums.MaxLogoutSeconds)
	if minPrompt <= 0 {
		minPrompt = 10
	}
	if minLogout <= 0 {
		minLogout = 20
	}
	if minGap < 1 {
		minGap = 5
	}
	promptSeconds := minPrompt + minGap + 1
	if maxPrompt > 0 && promptSeconds > maxPrompt {
		promptSeconds = maxPrompt
	}
	if promptSeconds < minPrompt {
		promptSeconds = minPrompt
	}
	logoutSeconds := promptSeconds + minGap + 2
	if logoutSeconds < minLogout {
		logoutSeconds = minLogout
	}
	if maxLogout > 0 && logoutSeconds > maxLogout {
		logoutSeconds = maxLogout
	}
	if logoutSeconds <= promptSeconds+minGap {
		logoutSeconds = promptSeconds + minGap + 1
	}

	setInputValue(t, page, dashboardSettingsAutoLogoutPromptSelector, fmt.Sprintf("%d", promptSeconds))
	setInputValue(t, page, dashboardSettingsAutoLogoutLogoutSelector, fmt.Sprintf("%d", logoutSeconds))

	require.Eventually(t, func() bool {
		var current struct {
			Enabled       bool    `json:"enabled"`
			PromptSeconds float64 `json:"promptSeconds"`
			LogoutSeconds float64 `json:"logoutSeconds"`
		}
		evaluateScriptInto(t, page, dashboardReadAutoLogoutSettingsScript, &current)
		if !current.Enabled {
			return false
		}
		return int(current.PromptSeconds) == promptSeconds && int(current.LogoutSeconds) == logoutSeconds
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	closeButton = waitForVisibleElement(t, page, "#settings-modal .btn-close")
	require.NoError(t, closeButton.Click(proto.InputMouseButtonLeft, 1))
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardSettingsModalVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	waitForVisibleElement(t, page, dashboardNotificationSelector)
	clickSelector(t, page, dashboardDismissButtonSelector)
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardSessionTimeoutVisibleScript)
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

func TestDashboardSessionTimeoutIgnoresSyntheticActivity(t *testing.T) {
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

	evaluateScriptInto(t, page, dashboardDispatchSyntheticMousemoveScript, nil)

	require.Eventually(t, func() bool {
		notificationElement, elementErr := page.Element(dashboardNotificationSelector)
		if elementErr != nil {
			return false
		}
		visible, visibleErr := notificationElement.Visible()
		if visibleErr != nil {
			return false
		}
		return visible
	}, time.Second, 100*time.Millisecond)
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

	clickSelector(t, page, dashboardPublicThemeToggleSelector)
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

func TestDashboardSiteFaviconOpensOrigin(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Favicon Origin Site",
		AllowedOrigin:              "https://open-target.example",
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 16,
		FaviconData: []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
			0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
			0x54, 0x78, 0x9C, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
			0x00, 0x04, 0xBF, 0x01, 0xFE, 0xA7, 0x65, 0x81,
			0xF0, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
			0x44, 0xAE, 0x42, 0x60, 0x82,
		},
		FaviconContentType: "image/png",
		FaviconFetchedAt:   time.Now().UTC(),
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
		return evaluateScriptBoolean(t, page, dashboardSiteFaviconVisibleScript)
	}, 5*time.Second, 100*time.Millisecond)

	evaluateScriptInto(t, page, dashboardCaptureWindowOpenScript, nil)

	clickSelector(t, page, dashboardSiteFaviconSelector)

	type windowOpenRecord struct {
		URL      string `json:"url"`
		Target   string `json:"target"`
		Features string `json:"features"`
	}

	var windowOpenCalls []windowOpenRecord
	evaluateScriptInto(t, page, dashboardOpenedWindowCallsScript, &windowOpenCalls)

	require.Len(t, windowOpenCalls, 1)
	require.Equal(t, site.AllowedOrigin, windowOpenCalls[0].URL)
	require.Equal(t, "_blank", windowOpenCalls[0].Target)
	require.Equal(t, "noopener,noreferrer", windowOpenCalls[0].Features)
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
