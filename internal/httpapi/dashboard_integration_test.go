package httpapi_test

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
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
	dashboardTestSessionSecretBytes = "12345678901234567890123456789012"
	dashboardTestAdminEmail         = "admin@example.com"
	dashboardTestAdminDisplayName   = "Admin Example"
	dashboardTestWidgetBaseURL      = "http://example.test"
	dashboardTestLandingPath        = "/landing"
	dashboardTestDashboardRoute     = "/app"
	dashboardPromptWaitTimeout      = 10 * time.Second
	dashboardPromptPollInterval     = 200 * time.Millisecond
	dashboardNotificationSelector   = "#session-timeout-notification"
	dashboardPromptVisibleScript    = `(function(){
		var element = document.querySelector('#session-timeout-notification');
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		var rect = element.getBoundingClientRect();
		return style.display !== 'none' && style.visibility !== 'hidden' && rect.width > 0 && rect.height > 0;
	}())`
	dashboardDismissButtonSelector              = "#session-timeout-dismiss-button"
	dashboardConfirmButtonSelector              = "#session-timeout-confirm-button"
	dashboardSettingsButtonSelector             = "#settings-button"
	dashboardSettingsMenuSelector               = "#settings-menu"
	dashboardSettingsMenuItemSelector           = "#settings-menu-settings"
	dashboardSettingsModalSelector              = "#settings-modal"
	dashboardWidgetBottomOffsetInputSelector    = "#widget-placement-bottom-offset"
	dashboardWidgetBottomOffsetIncreaseSelector = "#widget-bottom-offset-increase"
	dashboardWidgetBottomOffsetDecreaseSelector = "#widget-bottom-offset-decrease"
	dashboardSubscribeTestButtonSelector        = "#subscribe-test-button"
	subscribeTestFormContainerSelector          = "#subscribe-test-inline-preview"
	subscribeTestLogSelector                    = "#subscribe-test-log"
	dashboardTrafficTestButtonSelector          = "#traffic-test-button"
	trafficTestURLInputSelector                 = "#traffic-test-url"
	trafficTestSendButtonSelector               = "#traffic-test-send-hit"
	trafficTestTotalSelector                    = "#traffic-test-visit-total"
	trafficTestUniqueSelector                   = "#traffic-test-visit-unique"
	widgetBottomOffsetStepPixels                = 10
	dashboardSaveSiteButtonSelector             = "#save-site-button"
	dashboardReadWidgetBottomOffsetScript       = `(function() {
		var input = document.getElementById('widget-placement-bottom-offset');
		if (!input) { return ''; }
		return String(input.value || '');
	}())`
	dashboardSettingsAutoLogoutToggleSelector = "#settings-auto-logout-enabled"
	dashboardSettingsAutoLogoutPromptSelector = "#settings-auto-logout-prompt-seconds"
	dashboardSettingsAutoLogoutLogoutSelector = "#settings-auto-logout-logout-seconds"
	dashboardFeedbackWidgetSnippetSelector    = "#widget-snippet"
	dashboardSubscribeWidgetSnippetSelector   = "#subscribe-widget-snippet"
	dashboardTrafficWidgetSnippetSelector     = "#traffic-widget-snippet"
	dashboardFeedbackCopyButtonSelector       = "#copy-widget-snippet"
	dashboardSubscribeCopyButtonSelector      = "#copy-subscribe-widget-snippet"
	dashboardTrafficCopyButtonSelector        = "#copy-traffic-widget-snippet"
	dashboardFeedbackWidgetCardSelector       = `[data-widget-card="feedback"]`
	dashboardSubscribeWidgetCardSelector      = `[data-widget-card="subscribe"]`
	dashboardTrafficWidgetCardSelector        = `[data-widget-card="traffic"]`
	dashboardDashboardCardSelector            = "[data-dashboard-card]"
	dashboardFeedbackMessagesCardSelector     = `[data-dashboard-card="feedback"]`
	dashboardSubscribersCardSelector          = `[data-dashboard-card="subscribers"]`
	dashboardTrafficCardSelector              = `[data-dashboard-card="traffic"]`
	dashboardSubscribersTableBodySelector     = "#subscribers-table-body"
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
	dashboardUserEmailSelector            = "#user-email"
	dashboardFooterSelector               = "#dashboard-footer"
	dashboardTrafficStatusSelector        = "#traffic-status"
	dashboardVisitCountSelector           = "#visit-count"
	dashboardUniqueVisitorCountSelector   = "#unique-visitor-count"
	dashboardTopPagesTableBodySelector    = "#top-pages-table-body"
	dashboardTopPagesPlaceholderText      = "No visits yet."
	dashboardForcePromptScript            = "if (window.__loopawareDashboardIdleTestHooks && typeof window.__loopawareDashboardIdleTestHooks.forcePrompt === 'function') { window.__loopawareDashboardIdleTestHooks.forcePrompt(); }"
	dashboardForceLogoutScript            = "if (window.__loopawareDashboardIdleTestHooks && typeof window.__loopawareDashboardIdleTestHooks.forceLogout === 'function') { window.__loopawareDashboardIdleTestHooks.forceLogout(); }"
	dashboardNotificationBackgroundScript = `window.getComputedStyle(document.querySelector("#session-timeout-notification")).backgroundColor`
	dashboardLocationPathScript           = "window.location.pathname"
	dashboardIdleHooksReadyScript         = "typeof window.__loopawareDashboardIdleTestHooks !== 'undefined'"
	dashboardSelectFirstSiteScript        = `(function() {
	                var list = document.getElementById('sites-list');
	                if (!list) { return false; }
	                var item = list.querySelector('[data-site-id]');
                if (!item) { return false; }
                item.click();
                return true;
        }())`
	dashboardTrafficStatusHiddenScript = `(function() {
                var status = document.querySelector('#traffic-status');
                if (!status) { return false; }
                return status.classList.contains('d-none');
        }())`
	dashboardTopPagesPlaceholderScript = `(function() {
                var body = document.querySelector('#top-pages-table-body');
                if (!body) { return ''; }
                var row = body.querySelector('tr');
                if (!row) { return ''; }
                return row.textContent.trim();
        }())`
	dashboardTopPagesRowsScript = `(function() {
                var body = document.querySelector('#top-pages-table-body');
                if (!body) { return []; }
                var rows = body.querySelectorAll('tr');
                var results = [];
                rows.forEach(function(row) {
                        var cells = row.querySelectorAll('td');
                        if (cells.length < 2) { return; }
                        results.push({ path: cells[0].textContent.trim(), count: cells[1].textContent.trim() });
                });
                return results;
        }())`
	dashboardVisitCountsScript = `(function() {
                var total = document.getElementById('visit-count');
                var unique = document.getElementById('unique-visitor-count');
                return {
                        totalVisible: !!(total && !total.classList.contains('d-none')),
                        totalText: total ? total.textContent.trim() : '',
                        uniqueVisible: !!(unique && !unique.classList.contains('d-none')),
                        uniqueText: unique ? unique.textContent.trim() : ''
                };
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
	dashboardFooterThemeModeScript      = `(function(){var footer=document.getElementById('dashboard-footer');return footer ? (footer.getAttribute('theme-mode') || '') : '';}())`
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
	widgetTestBottomOffsetInputSelector       = "#widget-test-bottom-offset"
	widgetTestOffsetIncreaseSelector          = "#widget-test-offset-increase"
	widgetTestOffsetDecreaseSelector          = "#widget-test-offset-decrease"
	widgetTestSaveButtonSelector              = "#widget-test-save"
	widgetTestReadOffsetInputScript           = `(function() {
		var input = document.getElementById('widget-test-bottom-offset');
		if (!input) { return ''; }
		return String(input.value || '');
	}())`
	widgetTestReadSummaryOffsetScript = `(function() {
	var summary = document.getElementById('widget-test-summary-offset');
	if (!summary) { return ''; }
	return String(summary.textContent || '');
}())`
	widgetTestBubbleBottomScript = `(function() {
	var bubble = document.getElementById('mp-feedback-bubble');
	if (!bubble) { return ''; }
	return bubble.style.bottom || '';
}())`
	widgetTestPanelBottomScript = `(function() {
var panel = document.getElementById('mp-feedback-panel');
if (!panel) { return ''; }
return panel.style.bottom || '';
}())`
	dashboardWidgetBottomOffsetPresenceScript = `(function() {
	return !!document.getElementById('widget-placement-bottom-offset');
}())`
	widgetTestEnsurePreviewElementsScript = `(function() {
	var bubble = document.getElementById('mp-feedback-bubble');
	if (!bubble) {
		bubble = document.createElement('div');
		bubble.id = 'mp-feedback-bubble';
		bubble.style.position = 'fixed';
		document.body.appendChild(bubble);
	}
	var panel = document.getElementById('mp-feedback-panel');
	if (!panel) {
		panel = document.createElement('div');
		panel.id = 'mp-feedback-panel';
			panel.style.position = 'fixed';
			document.body.appendChild(panel);
		}
		return true;
	}())`
	widgetBubblePresenceScript = `(function() {
		return !!document.getElementById('mp-feedback-bubble');
	}())`
)

type stubDashboardNotifier struct{}

func (stubDashboardNotifier) NotifyFeedback(ctx context.Context, site model.Site, feedback model.Feedback) (string, error) {
	return model.FeedbackDeliveryMailed, nil
}

func (stubDashboardNotifier) NotifySubscription(ctx context.Context, site model.Site, subscriber model.Subscriber) error {
	return nil
}

type dashboardIntegrationHarness struct {
	router             *gin.Engine
	authManager        *httpapi.AuthManager
	faviconManager     *httpapi.SiteFaviconManager
	database           *gorm.DB
	sqlDB              *sql.DB
	server             *httptest.Server
	baseURL            string
	subscriptionEvents *httpapi.SubscriptionTestEventBroadcaster
}

type dashboardHarnessOptions struct {
	subscriptionNotifier httpapi.SubscriptionNotifier
}

type dashboardHarnessOption func(*dashboardHarnessOptions)

func withSubscriptionNotifier(notifier httpapi.SubscriptionNotifier) dashboardHarnessOption {
	return func(options *dashboardHarnessOptions) {
		if notifier != nil {
			options.subscriptionNotifier = notifier
		}
	}
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
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardPromptVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	lightBackgroundColor := evaluateScriptString(t, page, dashboardNotificationBackgroundScript)
	lightColor := mustParseRGBColor(t, lightBackgroundColor)

	evaluateScriptInto(t, page, `(function(){
		var footer = document.getElementById('dashboard-footer');
		if (!footer || typeof footer.dispatchEvent !== 'function') { return false; }
		footer.dispatchEvent(new CustomEvent('mpr-footer:theme-change', { detail: { theme: 'dark' } }));
		return true;
	}())`, nil)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardPromptVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	darkBackgroundColor := evaluateScriptString(t, page, dashboardNotificationBackgroundScript)
	darkColor := mustParseRGBColor(t, darkBackgroundColor)
	require.NotEqual(t, lightColor, darkColor)

	dismissClickScript := fmt.Sprintf("document.querySelector(%q).click()", dashboardDismissButtonSelector)
	evaluateScriptInto(t, page, dismissClickScript, nil)
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

func TestDashboardShowsDistinctWidgetSnippets(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Widget Snippet Site",
		AllowedOrigin: harness.baseURL,
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
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	snippetScript := fmt.Sprintf(`(function(){
		var feedback = document.querySelector(%q);
		var subscribe = document.querySelector(%q);
		var traffic = document.querySelector(%q);
		var copyFeedback = document.querySelector(%q);
		var copySubscribe = document.querySelector(%q);
		var copyTraffic = document.querySelector(%q);
		return {
			feedbackCard: !!document.querySelector(%q),
			subscribeCard: !!document.querySelector(%q),
			trafficCard: !!document.querySelector(%q),
			feedback: feedback ? (feedback.value || '').trim() : '',
			subscribe: subscribe ? (subscribe.value || '').trim() : '',
			traffic: traffic ? (traffic.value || '').trim() : '',
			feedbackCopyDisabled: copyFeedback ? !!copyFeedback.disabled : true,
			subscribeCopyDisabled: copySubscribe ? !!copySubscribe.disabled : true,
			trafficCopyDisabled: copyTraffic ? !!copyTraffic.disabled : true
		};
	}())`, dashboardFeedbackWidgetSnippetSelector, dashboardSubscribeWidgetSnippetSelector, dashboardTrafficWidgetSnippetSelector, dashboardFeedbackCopyButtonSelector, dashboardSubscribeCopyButtonSelector, dashboardTrafficCopyButtonSelector, dashboardFeedbackWidgetCardSelector, dashboardSubscribeWidgetCardSelector, dashboardTrafficWidgetCardSelector)

	var snippets struct {
		FeedbackCard          bool   `json:"feedbackCard"`
		SubscribeCard         bool   `json:"subscribeCard"`
		TrafficCard           bool   `json:"trafficCard"`
		Feedback              string `json:"feedback"`
		Subscribe             string `json:"subscribe"`
		Traffic               string `json:"traffic"`
		FeedbackCopyDisabled  bool   `json:"feedbackCopyDisabled"`
		SubscribeCopyDisabled bool   `json:"subscribeCopyDisabled"`
		TrafficCopyDisabled   bool   `json:"trafficCopyDisabled"`
	}

	require.Eventually(t, func() bool {
		evaluateScriptInto(t, page, snippetScript, &snippets)
		return snippets.FeedbackCard && snippets.SubscribeCard && snippets.TrafficCard &&
			strings.Contains(snippets.Feedback, site.ID) &&
			strings.Contains(snippets.Subscribe, site.ID) &&
			strings.Contains(snippets.Traffic, site.ID)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.True(t, snippets.FeedbackCard)
	require.True(t, snippets.SubscribeCard)
	require.True(t, snippets.TrafficCard)
	require.Contains(t, snippets.Feedback, "/widget.js?site_id="+site.ID)
	require.Contains(t, snippets.Subscribe, "/subscribe.js?site_id="+site.ID)
	require.Contains(t, snippets.Traffic, "/pixel.js?site_id="+site.ID)
	require.False(t, snippets.FeedbackCopyDisabled)
	require.False(t, snippets.SubscribeCopyDisabled)
	require.False(t, snippets.TrafficCopyDisabled)
}

func TestDashboardSeparatesSubscribersAndTrafficCards(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Dashboard Cards Site",
		AllowedOrigin: harness.baseURL,
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
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	cardScript := fmt.Sprintf(`(function(){
		var sequence = Array.prototype.slice.call(document.querySelectorAll(%q)).map(function(node) {
			return (node.getAttribute('data-dashboard-card') || '').trim();
		});
		var feedbackCard = document.querySelector(%q);
		var subscribersCard = document.querySelector(%q);
		var trafficCard = document.querySelector(%q);
		return {
			cards: sequence,
			subscribersInFeedback: !!(feedbackCard && feedbackCard.querySelector(%q)),
			trafficInFeedback: !!(feedbackCard && feedbackCard.querySelector(%q)),
			subscribersInOwnCard: !!(subscribersCard && subscribersCard.querySelector(%q)),
			trafficInOwnCard: !!(trafficCard && trafficCard.querySelector(%q))
		};
	}())`, dashboardDashboardCardSelector, dashboardFeedbackMessagesCardSelector, dashboardSubscribersCardSelector, dashboardTrafficCardSelector, dashboardSubscribersTableBodySelector, dashboardTopPagesTableBodySelector, dashboardSubscribersTableBodySelector, dashboardTopPagesTableBodySelector)

	var layout struct {
		Cards                 []string `json:"cards"`
		SubscribersInFeedback bool     `json:"subscribersInFeedback"`
		TrafficInFeedback     bool     `json:"trafficInFeedback"`
		SubscribersInOwnCard  bool     `json:"subscribersInOwnCard"`
		TrafficInOwnCard      bool     `json:"trafficInOwnCard"`
	}

	evaluateScriptInto(t, page, cardScript, &layout)

	require.Equal(t, []string{"feedback", "subscribers", "traffic"}, layout.Cards)
	require.False(t, layout.SubscribersInFeedback)
	require.False(t, layout.TrafficInFeedback)
	require.True(t, layout.SubscribersInOwnCard)
	require.True(t, layout.TrafficInOwnCard)
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
	dismissClickScript := fmt.Sprintf("document.querySelector(%q).click()", dashboardDismissButtonSelector)
	evaluateScriptInto(t, page, dismissClickScript, nil)
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

func TestDashboardSessionTimeoutPromptPersistsAfterMousemove(t *testing.T) {
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

	require.NoError(t, page.Mouse.MoveTo(proto.Point{X: 200, Y: 200}))

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSessionTimeoutVisibleScript)
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

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardFooterThemeModeScript) == "dark"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

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
		name          string
		storedTheme   string
		publicTheme   string
		expectedTheme string
	}{
		{
			name:          "public_dark_overrides_stored_light",
			storedTheme:   "light",
			publicTheme:   "dark",
			expectedTheme: "dark",
		},
		{
			name:          "public_light_overrides_stored_dark",
			storedTheme:   "dark",
			publicTheme:   "light",
			expectedTheme: "light",
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

			require.Eventually(t, func() bool {
				return evaluateScriptString(t, page, dashboardFooterThemeModeScript) == testCase.expectedTheme
			}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
		})
	}
}

func TestDashboardTrafficPlaceholderWithoutVisits(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Traffic Placeholder",
		AllowedOrigin: harness.baseURL,
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
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardTrafficStatusHiddenScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	placeholder := evaluateScriptString(t, page, dashboardTopPagesPlaceholderScript)
	require.Equal(t, dashboardTopPagesPlaceholderText, placeholder)

	var visitBadges struct {
		TotalVisible  bool   `json:"totalVisible"`
		TotalText     string `json:"totalText"`
		UniqueVisible bool   `json:"uniqueVisible"`
		UniqueText    string `json:"uniqueText"`
	}
	evaluateScriptInto(t, page, dashboardVisitCountsScript, &visitBadges)
	require.False(t, visitBadges.TotalVisible)
	require.Empty(t, visitBadges.TotalText)
	require.False(t, visitBadges.UniqueVisible)
	require.Empty(t, visitBadges.UniqueText)
}

func TestDashboardTrafficRendersTopPages(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Traffic Data",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	visitorA := storage.NewID()
	visitorB := storage.NewID()
	visits := []model.SiteVisitInput{
		{SiteID: site.ID, URL: harness.baseURL + "/home", VisitorID: visitorA},
		{SiteID: site.ID, URL: harness.baseURL + "/home", VisitorID: visitorB},
		{SiteID: site.ID, URL: harness.baseURL + "/about", VisitorID: visitorA},
	}
	for _, input := range visits {
		visit, err := model.NewSiteVisit(input)
		require.NoError(t, err)
		require.NoError(t, harness.database.Create(&visit).Error)
	}

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(t, func() bool {
		var rows []struct {
			Path  string `json:"path"`
			Count string `json:"count"`
		}
		evaluateScriptInto(t, page, dashboardTopPagesRowsScript, &rows)
		if len(rows) != 2 {
			return false
		}
		return rows[0].Path == "/home" && rows[0].Count == "2" && rows[1].Path == "/about" && rows[1].Count == "1"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	var visitBadges struct {
		TotalVisible  bool   `json:"totalVisible"`
		TotalText     string `json:"totalText"`
		UniqueVisible bool   `json:"uniqueVisible"`
		UniqueText    string `json:"uniqueText"`
	}
	evaluateScriptInto(t, page, dashboardVisitCountsScript, &visitBadges)
	require.True(t, visitBadges.TotalVisible)
	require.Equal(t, "3 visits", visitBadges.TotalText)
	require.True(t, visitBadges.UniqueVisible)
	require.Equal(t, "2 unique", visitBadges.UniqueText)
	require.True(t, evaluateScriptBoolean(t, page, dashboardTrafficStatusHiddenScript))
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

	require.True(t, evaluateScriptBoolean(t, page, `(function(){
    var body = document.body;
    if (!body) { return false; }
    return body.classList.contains('bg-dark') && body.classList.contains('text-light');
  }())`))

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardFooterThemeModeScript) == "dark"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	evaluateScriptInto(t, page, `(function(){
		var footer = document.getElementById('dashboard-footer');
		if (!footer || typeof footer.dispatchEvent !== 'function') { return false; }
		footer.dispatchEvent(new CustomEvent('mpr-footer:theme-change', { detail: { theme: 'light' } }));
		return true;
	}())`, nil)

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardDocumentThemeScript) == "light"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	finalDashboardStoredTheme := evaluateScriptString(t, page, dashboardStoredDashboardThemeScript)
	require.Equal(t, "light", finalDashboardStoredTheme)

	finalPublicStoredTheme := evaluateScriptString(t, page, dashboardStoredPublicThemeScript)
	require.Equal(t, "light", finalPublicStoredTheme)

	require.True(t, evaluateScriptBoolean(t, page, `(function(){
    var body = document.body;
    if (!body) { return false; }
    return body.classList.contains('bg-light') && body.classList.contains('text-dark');
  }())`))
}

func TestWidgetTestFeedbackSubmissionSucceeds(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Widget Test Feedback",
		AllowedOrigin:              harness.baseURL,
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 24,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	widgetTestURL := fmt.Sprintf("%s/app/sites/%s/widget-test", harness.baseURL, site.ID)
	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(widgetTestURL))
	waitNavigation()

	waitForVisibleElement(t, page, widgetBubbleSelector)
	clickSelector(t, page, widgetBubbleSelector)
	waitForVisibleElement(t, page, widgetPanelSelector)

	testContact := "tester@example.com"
	testMessage := "Widget test feedback message"

	setInputValue(t, page, widgetContactSelector, testContact)
	setInputValue(t, page, widgetMessageSelector, testMessage)
	clickSelector(t, page, widgetSendButtonSelector)

	var statusText string
	require.Eventually(t, func() bool {
		statusText = evaluateScriptString(t, page, widgetStatusResolveScript)
		return strings.Contains(statusText, widgetSuccessStatusMessage)
	}, 5*time.Second, 100*time.Millisecond)

	var storedFeedback model.Feedback
	require.Eventually(t, func() bool {
		return harness.database.Where("site_id = ? AND contact = ?", site.ID, testContact).Order("created_at desc").First(&storedFeedback).Error == nil
	}, 5*time.Second, 100*time.Millisecond)
	require.Equal(t, testMessage, storedFeedback.Message)
}

func TestWidgetTestPlacementSavePersists(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Widget Test Placement",
		AllowedOrigin:              harness.baseURL,
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 24,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	widgetTestURL := fmt.Sprintf("%s/app/sites/%s/widget-test", harness.baseURL, site.ID)
	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(widgetTestURL))
	waitNavigation()

	interceptFetchRequests(t, page)
	clickSelector(t, page, "#widget-test-side-left")
	setInputValue(t, page, "#widget-test-bottom-offset", "72")
	endpointValue := evaluateScriptString(t, page, `(function(){
  var body = document.body;
  return body ? body.getAttribute('data-update-endpoint') || '' : '';
}())`)
	require.NotEmpty(t, endpointValue)
	t.Logf("widget test update endpoint: %s", endpointValue)
	clickSelector(t, page, "#widget-test-save")

	type siteUpdatePayload struct {
		WidgetBubbleSide         string `json:"widget_bubble_side"`
		WidgetBubbleBottomOffset int    `json:"widget_bubble_bottom_offset"`
	}

	var payload siteUpdatePayload
	var payloadStatus int
	require.Eventually(t, func() bool {
		requests := readCapturedFetchRequests(t, page)
		t.Logf("captured widget test placement requests: %#v", requests)
		for _, record := range requests {
			if !strings.HasSuffix(record.URL, "/api/sites/"+site.ID) {
				continue
			}
			if !strings.EqualFold(record.Method, http.MethodPatch) {
				continue
			}
			if record.Body == "" {
				continue
			}
			if record.Status == 0 {
				continue
			}
			if err := json.Unmarshal([]byte(record.Body), &payload); err != nil {
				return false
			}
			payloadStatus = record.Status
			return true
		}
		return false
	}, 5*time.Second, 100*time.Millisecond)

	require.Equal(t, "left", payload.WidgetBubbleSide)
	require.Equal(t, 72, payload.WidgetBubbleBottomOffset)
	require.Equal(t, http.StatusOK, payloadStatus)

	var updatedSite model.Site
	require.Eventually(t, func() bool {
		return harness.database.First(&updatedSite, "id = ?", site.ID).Error == nil
	}, 20*time.Second, 100*time.Millisecond)
	require.Equal(t, "left", updatedSite.WidgetBubbleSide)
	require.Equal(t, 72, updatedSite.WidgetBubbleBottomOffsetPx)
}

func TestSubscribeWidgetTestFlowSubmitsSubscription(t *testing.T) {
	subscriptionNotifier := &recordingSubscriptionNotifier{t: t}
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail, withSubscriptionNotifier(subscriptionNotifier))
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Subscribe Test Flow",
		AllowedOrigin: "https://widget.example",
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(harness.baseURL+dashboardTestDashboardRoute))
	waitNavigation()

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.True(t, evaluateScriptBoolean(t, page, `(function(){
        var originalOpen = window.open;
        window.open = function(url, target, features) {
          if (url) {
            window.location.assign(url);
          }
          return window;
        };
        window.open._original = originalOpen;
        return true;
      }())`))

	waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	clickSelector(t, page, dashboardSubscribeTestButtonSelector)
	waitNavigation()

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardLocationPathScript) == fmt.Sprintf("/app/sites/%s/subscribe-test", site.ID)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	waitForVisibleElement(t, page, subscribeTestFormContainerSelector)
	waitForVisibleElement(t, page, "#"+subscribeEmailInputID)
	waitForVisibleElement(t, page, "#"+subscribeSubmitButtonID)

	testEmail := "preview-subscriber@example.com"
	setInputValue(t, page, "#"+subscribeEmailInputID, testEmail)
	setInputValue(t, page, "#"+subscribeNameInputID, "Preview User")
	clickSelector(t, page, "#"+subscribeSubmitButtonID)

	var inlineStatus string
	require.Eventually(t, func() bool {
		inlineStatus = evaluateScriptString(t, page, fmt.Sprintf(`(function(){
        var status = document.getElementById(%q);
        if (!status) { return ""; }
        return status.innerText || "";
      }())`, subscribeStatusElementID))
		return strings.Contains(inlineStatus, "You're on the list")
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)

	require.Eventually(t, func() bool {
		var stored model.Subscriber
		return harness.database.
			Where("site_id = ? AND email = ?", site.ID, testEmail).
			Order("created_at desc").
			First(&stored).Error == nil
	}, 5*time.Second, 100*time.Millisecond)

	require.Equal(t, 1, subscriptionNotifier.CallCount())
	notification := subscriptionNotifier.LastCall()
	require.Equal(t, site.ID, notification.Site.ID)
	require.Equal(t, testEmail, notification.Subscriber.Email)

}

func TestSubscribeTestPageExposesEventsEndpoint(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Subscribe Test Events Endpoint",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/app/sites/%s/subscribe-test", harness.baseURL, site.ID), nil)
	require.NoError(t, err)
	request.AddCookie(sessionCookie)

	response, err := harness.server.Client().Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	require.Equal(t, http.StatusOK, response.StatusCode)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	expectedEndpoint := fmt.Sprintf(`data-events-endpoint="/app/sites/%s/subscribe-test/events"`, site.ID)
	require.Contains(t, string(body), expectedEndpoint)
}

func TestSubscribeTestEventsEndpointStreamsBroadcasts(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Subscribe Test Event Stream",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/app/sites/%s/subscribe-test/events", harness.baseURL, site.ID), nil)
	require.NoError(t, err)
	request.AddCookie(sessionCookie)

	client := harness.server.Client()
	client.Timeout = 5 * time.Second

	response, err := client.Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)

	reader := bufio.NewReader(response.Body)

	expectedEvent := httpapi.SubscriptionTestEvent{
		SiteID:       site.ID,
		SubscriberID: "subscriber-123",
		Email:        "subscriber@example.com",
		EventType:    "submission",
		Status:       "received",
	}

	lineChannel := make(chan string, 1)
	errorChannel := make(chan error, 1)
	go func() {
		defer close(lineChannel)
		defer close(errorChannel)
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			errorChannel <- readErr
			return
		}
		lineChannel <- line
	}()

	harness.subscriptionEvents.Broadcast(expectedEvent)

	var line string
	select {
	case line = <-lineChannel:
	case readErr := <-errorChannel:
		require.NoError(t, readErr)
	case <-time.After(3 * time.Second):
		require.FailNow(t, "timed out waiting for subscription test event")
	}

	require.NoError(t, response.Body.Close())

	payloadText := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	require.NotEmpty(t, payloadText)

	var receivedEvent httpapi.SubscriptionTestEvent
	require.NoError(t, json.Unmarshal([]byte(payloadText), &receivedEvent))

	require.Equal(t, expectedEvent.SiteID, receivedEvent.SiteID)
	require.Equal(t, expectedEvent.SubscriberID, receivedEvent.SubscriberID)
	require.Equal(t, expectedEvent.Email, receivedEvent.Email)
	require.Equal(t, expectedEvent.EventType, receivedEvent.EventType)
	require.Equal(t, expectedEvent.Status, receivedEvent.Status)
	require.False(t, receivedEvent.Timestamp.IsZero())
}
func TestTrafficWidgetTestFlowRecordsVisit(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Traffic Test Flow",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(harness.baseURL+dashboardTestDashboardRoute))
	waitNavigation()

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.True(t, evaluateScriptBoolean(t, page, `(function(){
        var originalOpen = window.open;
        window.open = function(url){
          if (url) { window.location.assign(url); }
          return window;
        };
        window.open._original = originalOpen;
        return true;
      }())`))

	waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	clickSelector(t, page, dashboardTrafficTestButtonSelector)
	waitNavigation()

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardLocationPathScript) == fmt.Sprintf("/app/sites/%s/traffic-test", site.ID)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	testURL := harness.baseURL + "/preview-traffic"
	waitForVisibleElement(t, page, trafficTestURLInputSelector)
	setInputValue(t, page, trafficTestURLInputSelector, testURL)
	currentURLValue := evaluateScriptString(t, page, fmt.Sprintf(`document.querySelector(%q).value || ""`, trafficTestURLInputSelector))
	require.Equal(t, testURL, strings.TrimSpace(currentURLValue))
	clickSelector(t, page, trafficTestSendButtonSelector)

	require.Eventually(t, func() bool {
		var count int64
		if err := harness.database.Model(&model.SiteVisit{}).Where("site_id = ?", site.ID).Count(&count).Error; err != nil {
			return false
		}
		return count > 0
	}, 5*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		totalText := evaluateScriptString(t, page, fmt.Sprintf(`document.querySelector(%q).textContent || ""`, trafficTestTotalSelector))
		return strings.TrimSpace(totalText) != "" && strings.TrimSpace(totalText) != "0"
	}, 5*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		uniqueText := evaluateScriptString(t, page, fmt.Sprintf(`document.querySelector(%q).textContent || ""`, trafficTestUniqueSelector))
		return strings.TrimSpace(uniqueText) == "1"
	}, 5*time.Second, 100*time.Millisecond)

	clickSelector(t, page, trafficTestSendButtonSelector)

	require.Eventually(t, func() bool {
		totalText := evaluateScriptString(t, page, fmt.Sprintf(`document.querySelector(%q).textContent || ""`, trafficTestTotalSelector))
		return strings.TrimSpace(totalText) == "2"
	}, 5*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		uniqueText := evaluateScriptString(t, page, fmt.Sprintf(`document.querySelector(%q).textContent || ""`, trafficTestUniqueSelector))
		return strings.TrimSpace(uniqueText) == "1"
	}, 5*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		logText := evaluateScriptString(t, page, `document.getElementById("traffic-test-log").textContent || ""`)
		return strings.Contains(logText, "Country: Local network") && strings.Contains(logText, "IP:")
	}, 5*time.Second, 100*time.Millisecond)
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

func TestDashboardWidgetPlacementSavePersists(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Widget Placement Save",
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

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, 5*time.Second, 100*time.Millisecond)

	interceptFetchRequests(t, page)
	clickSelector(t, page, "#widget-placement-side-left")
	setInputValue(t, page, dashboardWidgetBottomOffsetInputSelector, "88")
	clickSelector(t, page, dashboardSaveSiteButtonSelector)

	type siteUpdatePayload struct {
		WidgetBubbleSide         string `json:"widget_bubble_side"`
		WidgetBubbleBottomOffset int    `json:"widget_bubble_bottom_offset"`
	}

	var payload siteUpdatePayload
	var payloadStatus int
	require.Eventually(t, func() bool {
		requests := readCapturedFetchRequests(t, page)
		for _, record := range requests {
			if !strings.HasSuffix(record.URL, "/api/sites/"+site.ID) {
				continue
			}
			if !strings.EqualFold(record.Method, http.MethodPatch) {
				continue
			}
			if record.Body == "" {
				continue
			}
			if record.Status == 0 {
				continue
			}
			if err := json.Unmarshal([]byte(record.Body), &payload); err != nil {
				return false
			}
			payloadStatus = record.Status
			return true
		}
		return false
	}, 20*time.Second, 100*time.Millisecond)

	require.Equal(t, "left", payload.WidgetBubbleSide)
	require.Equal(t, 88, payload.WidgetBubbleBottomOffset)
	require.Equal(t, http.StatusOK, payloadStatus)

	var updatedSite model.Site
	require.Eventually(t, func() bool {
		return harness.database.First(&updatedSite, "id = ?", site.ID).Error == nil
	}, 5*time.Second, 100*time.Millisecond)
	require.Equal(t, "left", updatedSite.WidgetBubbleSide)
	require.Equal(t, 88, updatedSite.WidgetBubbleBottomOffsetPx)
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

func TestDashboardWidgetBottomOffsetStepButtonsAdjustAndPersist(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Offset Step Site",
		AllowedOrigin:              harness.baseURL,
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 24,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	waitForVisibleElement(t, page, dashboardWidgetBottomOffsetInputSelector)

	initialOffset := readDashboardWidgetBottomOffset(t, page)
	require.Equal(t, site.WidgetBubbleBottomOffsetPx, initialOffset)

	clickSelector(t, page, dashboardWidgetBottomOffsetIncreaseSelector)
	incrementedOffset := readDashboardWidgetBottomOffset(t, page)
	require.Equal(t, initialOffset+widgetBottomOffsetStepPixels, incrementedOffset)

	manualOffset := 37
	setInputValue(t, page, dashboardWidgetBottomOffsetInputSelector, strconv.Itoa(manualOffset))
	currentManual := readDashboardWidgetBottomOffset(t, page)
	require.Equal(t, manualOffset, currentManual)

	offsetInput := waitForVisibleElement(t, page, dashboardWidgetBottomOffsetInputSelector)
	require.NoError(t, offsetInput.Focus())

	require.NoError(t, page.Keyboard.Press(input.ArrowUp))
	upperValue := readDashboardWidgetBottomOffset(t, page)
	require.Equal(t, manualOffset+widgetBottomOffsetStepPixels, upperValue)

	require.NoError(t, page.Keyboard.Press(input.ArrowDown))
	finalOffset := readDashboardWidgetBottomOffset(t, page)
	require.Equal(t, manualOffset, finalOffset)

	interceptFetchRequests(t, page)
	clickSelector(t, page, dashboardSaveSiteButtonSelector)

	type siteUpdatePayload struct {
		WidgetBubbleBottomOffset int `json:"widget_bubble_bottom_offset"`
	}
	var payload siteUpdatePayload
	payloadStatus := 0

	require.Eventually(t, func() bool {
		requests := readCapturedFetchRequests(t, page)
		for _, record := range requests {
			if !strings.HasSuffix(record.URL, "/api/sites/"+site.ID) {
				continue
			}
			if !strings.EqualFold(record.Method, http.MethodPatch) {
				continue
			}
			if record.Body == "" {
				continue
			}
			if err := json.Unmarshal([]byte(record.Body), &payload); err != nil {
				return false
			}
			if record.Status == 0 {
				return false
			}
			payloadStatus = record.Status
			return true
		}
		return false
	}, 5*time.Second, 100*time.Millisecond)

	require.Equal(t, finalOffset, payload.WidgetBubbleBottomOffset)
	require.Equal(t, http.StatusOK, payloadStatus)
}

func TestWidgetTestBottomOffsetControlsAdjustPreviewAndPersist(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Widget Offset Controls",
		AllowedOrigin:              harness.baseURL,
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 28,
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

	evaluateScriptInto(t, page, widgetTestEnsurePreviewElementsScript, nil)
	waitForVisibleElement(t, page, widgetTestBottomOffsetInputSelector)

	require.Eventually(t, func() bool {
		return strings.TrimSpace(evaluateScriptString(t, page, widgetTestBubbleBottomScript)) != ""
	}, 5*time.Second, 100*time.Millisecond)
	require.Eventually(t, func() bool {
		return strings.TrimSpace(evaluateScriptString(t, page, widgetTestPanelBottomScript)) != ""
	}, 5*time.Second, 100*time.Millisecond)

	initialOffset := readWidgetTestBottomOffset(t, page)
	require.Equal(t, site.WidgetBubbleBottomOffsetPx, initialOffset)

	initialSummary := parseOffsetValue(t, evaluateScriptString(t, page, widgetTestReadSummaryOffsetScript))
	require.Equal(t, initialOffset, initialSummary)

	initialBubbleBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestBubbleBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", initialOffset), initialBubbleBottom)

	initialPanelBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestPanelBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", initialOffset+widgetPanelVerticalSpacingPixels), initialPanelBottom)

	clickSelector(t, page, widgetTestOffsetIncreaseSelector)

	stepOffset := readWidgetTestBottomOffset(t, page)
	require.Equal(t, initialOffset+widgetBottomOffsetStepPixels, stepOffset)

	stepSummary := parseOffsetValue(t, evaluateScriptString(t, page, widgetTestReadSummaryOffsetScript))
	require.Equal(t, stepOffset, stepSummary)

	stepBubbleBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestBubbleBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", stepOffset), stepBubbleBottom)

	stepPanelBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestPanelBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", stepOffset+widgetPanelVerticalSpacingPixels), stepPanelBottom)

	manualOffset := 31
	setInputValue(t, page, widgetTestBottomOffsetInputSelector, strconv.Itoa(manualOffset))

	currentManual := readWidgetTestBottomOffset(t, page)
	require.Equal(t, manualOffset, currentManual)

	manualSummary := parseOffsetValue(t, evaluateScriptString(t, page, widgetTestReadSummaryOffsetScript))
	require.Equal(t, manualOffset, manualSummary)

	manualBubbleBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestBubbleBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", manualOffset), manualBubbleBottom)

	manualPanelBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestPanelBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", manualOffset+widgetPanelVerticalSpacingPixels), manualPanelBottom)

	offsetInput := waitForVisibleElement(t, page, widgetTestBottomOffsetInputSelector)
	require.NoError(t, offsetInput.Focus())

	require.NoError(t, page.Keyboard.Press(input.ArrowDown))

	finalOffset := readWidgetTestBottomOffset(t, page)
	require.Equal(t, manualOffset-widgetBottomOffsetStepPixels, finalOffset)

	finalSummary := parseOffsetValue(t, evaluateScriptString(t, page, widgetTestReadSummaryOffsetScript))
	require.Equal(t, finalOffset, finalSummary)

	finalBubbleBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestBubbleBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", finalOffset), finalBubbleBottom)

	finalPanelBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestPanelBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", finalOffset+widgetPanelVerticalSpacingPixels), finalPanelBottom)

	redirectWait := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	clickSelector(t, page, widgetTestSaveButtonSelector)
	redirectWait()

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	currentPath := evaluateScriptString(t, page, dashboardLocationPathScript)
	require.Equal(t, dashboardTestDashboardRoute, currentPath)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, 5*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardWidgetBottomOffsetPresenceScript)
	}, 5*time.Second, 100*time.Millisecond)

	dashboardOffset := readDashboardWidgetBottomOffset(t, page)
	require.Equal(t, finalOffset, dashboardOffset)
}

func TestExampleRouteIsUnavailable(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	response, err := http.Get(harness.baseURL + "/example")
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

func readDashboardWidgetBottomOffset(testingT *testing.T, page *rod.Page) int {
	testingT.Helper()
	value := evaluateScriptString(testingT, page, dashboardReadWidgetBottomOffsetScript)
	return parseOffsetValue(testingT, value)
}

func readWidgetTestBottomOffset(testingT *testing.T, page *rod.Page) int {
	testingT.Helper()
	value := evaluateScriptString(testingT, page, widgetTestReadOffsetInputScript)
	return parseOffsetValue(testingT, value)
}

func parseOffsetValue(testingT *testing.T, value string) int {
	testingT.Helper()
	trimmed := strings.TrimSpace(value)
	require.NotEmpty(testingT, trimmed)
	parsed, err := strconv.Atoi(trimmed)
	require.NoError(testingT, err)
	return parsed
}

func buildDashboardIntegrationHarness(testingT *testing.T, adminEmail string, opts ...dashboardHarnessOption) *dashboardIntegrationHarness {
	testingT.Helper()

	session.NewSession([]byte(dashboardTestSessionSecretBytes))

	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	config := dashboardHarnessOptions{
		subscriptionNotifier: stubDashboardNotifier{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}

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
	subscriptionEvents := httpapi.NewSubscriptionTestEventBroadcaster()

	statsProvider := httpapi.NewDatabaseSiteStatisticsProvider(gormDatabase)
	siteHandlers := httpapi.NewSiteHandlers(gormDatabase, logger, dashboardTestWidgetBaseURL, faviconManager, statsProvider, feedbackBroadcaster)
	landingHandlers := httpapi.NewLandingPageHandlers(logger, authManager)
	privacyHandlers := httpapi.NewPrivacyPageHandlers(authManager)
	sitemapHandlers := httpapi.NewSitemapHandlers(dashboardTestWidgetBaseURL)
	dashboardHandlers := httpapi.NewDashboardWebHandlers(logger, dashboardTestLandingPath)
	publicHandlers := httpapi.NewPublicHandlers(gormDatabase, logger, feedbackBroadcaster, subscriptionEvents, stubDashboardNotifier{}, config.subscriptionNotifier, true)
	widgetTestHandlers := httpapi.NewSiteWidgetTestHandlers(gormDatabase, logger, dashboardTestWidgetBaseURL, feedbackBroadcaster, stubDashboardNotifier{})
	trafficTestHandlers := httpapi.NewSiteTrafficTestHandlers(gormDatabase, logger)
	subscribeTestHandlers := httpapi.NewSiteSubscribeTestHandlers(gormDatabase, logger, subscriptionEvents, config.subscriptionNotifier, true)

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
	router.GET("/app/sites/:id/traffic-test", authManager.RequireAuthenticatedWeb(), trafficTestHandlers.RenderTrafficTestPage)
	router.GET("/app/sites/:id/subscribe-test", authManager.RequireAuthenticatedWeb(), subscribeTestHandlers.RenderSubscribeTestPage)
	router.GET("/app/sites/:id/subscribe-test/events", authManager.RequireAuthenticatedJSON(), subscribeTestHandlers.StreamSubscriptionTestEvents)
	router.POST("/app/sites/:id/subscribe-test/subscriptions", authManager.RequireAuthenticatedJSON(), subscribeTestHandlers.CreateSubscription)
	router.POST(constants.LogoutPath, func(context *gin.Context) {
		context.Status(http.StatusOK)
	})

	router.GET("/api/visits", publicHandlers.CollectVisit)
	router.POST("/api/feedback", publicHandlers.CreateFeedback)
	router.POST("/api/subscriptions", publicHandlers.CreateSubscription)
	router.GET("/subscribe.js", publicHandlers.SubscribeJS)
	router.GET("/subscribe-demo", publicHandlers.SubscribeDemo)
	router.GET("/widget.js", publicHandlers.WidgetJS)

	apiGroup := router.Group("/api")
	apiGroup.Use(authManager.RequireAuthenticatedJSON())
	apiGroup.GET("/me", siteHandlers.CurrentUser)
	apiGroup.GET("/me/avatar", siteHandlers.UserAvatar)
	apiGroup.GET("/sites", siteHandlers.ListSites)
	apiGroup.POST("/sites", siteHandlers.CreateSite)
	apiGroup.PATCH("/sites/:id", siteHandlers.UpdateSite)
	apiGroup.DELETE("/sites/:id", siteHandlers.DeleteSite)
	apiGroup.GET("/sites/:id/messages", siteHandlers.ListMessagesBySite)
	apiGroup.GET("/sites/:id/visits/stats", siteHandlers.VisitStats)
	apiGroup.GET("/sites/:id/favicon", siteHandlers.SiteFavicon)
	apiGroup.GET("/sites/favicons/events", siteHandlers.StreamFaviconUpdates)
	apiGroup.GET("/sites/feedback/events", siteHandlers.StreamFeedbackUpdates)

	server := httptest.NewServer(router)

	testingT.Cleanup(func() {
		managerCancel()
		faviconManager.Stop()
		feedbackBroadcaster.Close()
		subscriptionEvents.Close()
		server.Close()
		require.NoError(testingT, sqlDatabase.Close())
	})

	return &dashboardIntegrationHarness{
		router:             router,
		authManager:        authManager,
		faviconManager:     faviconManager,
		database:           gormDatabase,
		sqlDB:              sqlDatabase,
		server:             server,
		baseURL:            server.URL,
		subscriptionEvents: subscriptionEvents,
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
