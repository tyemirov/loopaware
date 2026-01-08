package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
)

const (
	webkitTestTimeout              = 30 * time.Second
	webkitTestPollInterval         = 100 * time.Millisecond
	webkitProfileToggleSelector    = `[data-loopaware-profile-toggle="true"]`
	webkitSettingsButtonSelector   = `[data-loopaware-settings="true"]`
	webkitLogoutButtonSelector     = `[data-loopaware-logout="true"]`
	webkitSettingsModalSelector    = "#settings-modal"
	webkitProfileMenuItemsSelector = `[data-loopaware-profile-menu-items="true"]`
	webkitProfileMenuSelector      = `[data-loopaware-profile-menu="true"]`
)

var (
	playwrightInstance *playwright.Playwright
	webkitBrowser      playwright.Browser
)

func setupWebKitBrowser(testingT *testing.T) playwright.Browser {
	testingT.Helper()

	if webkitBrowser != nil {
		return webkitBrowser
	}

	var initErr error
	playwrightInstance, initErr = playwright.Run()
	if initErr != nil {
		testingT.Skipf("playwright not available: %v", initErr)
	}

	webkitBrowser, initErr = playwrightInstance.WebKit.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if initErr != nil {
		testingT.Skipf("webkit browser not available: %v", initErr)
	}

	testingT.Cleanup(func() {
		if webkitBrowser != nil {
			webkitBrowser.Close()
			webkitBrowser = nil
		}
		if playwrightInstance != nil {
			playwrightInstance.Stop()
			playwrightInstance = nil
		}
	})

	return webkitBrowser
}

func setWebKitCookie(testingT *testing.T, context playwright.BrowserContext, baseURL string, cookie *http.Cookie) {
	testingT.Helper()

	if cookie == nil {
		return
	}

	sameSite := playwright.SameSiteAttributeLax
	if cookie.SameSite == http.SameSiteStrictMode {
		sameSite = playwright.SameSiteAttributeStrict
	} else if cookie.SameSite == http.SameSiteNoneMode {
		sameSite = playwright.SameSiteAttributeNone
	}

	cookiePath := cookie.Path
	if cookiePath == "" {
		cookiePath = "/"
	}

	require.NoError(testingT, context.AddCookies([]playwright.OptionalCookie{
		{
			Name:     cookie.Name,
			Value:    cookie.Value,
			URL:      playwright.String(baseURL + cookiePath),
			Secure:   playwright.Bool(cookie.Secure),
			HttpOnly: playwright.Bool(cookie.HttpOnly),
			SameSite: sameSite,
		},
	}))
}

func TestWebKitProfileMenuSettingsOpens(testingT *testing.T) {
	browser := setupWebKitBrowser(testingT)

	harness := buildDashboardIntegrationHarness(testingT, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "WebKit Settings Test Site",
		AllowedOrigin: "https://webkit-test.example",
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(testingT, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	context, contextErr := browser.NewContext()
	require.NoError(testingT, contextErr)
	defer context.Close()

	setWebKitCookie(testingT, context, harness.server.URL, sessionCookie)

	page, pageErr := context.NewPage()
	require.NoError(testingT, pageErr)
	defer page.Close()

	_, navigateErr := page.Goto(harness.server.URL + "/app")
	require.NoError(testingT, navigateErr)

	require.Eventually(testingT, func() bool {
		result, evalErr := page.Evaluate(`() => {
			var header = document.querySelector('mpr-header');
			if (!header) { return ''; }
			return header.getAttribute('data-loopaware-auth-bound') || '';
		}`)
		if evalErr != nil {
			return false
		}
		return result == "true"
	}, webkitTestTimeout, webkitTestPollInterval, "auth binding did not complete")

	require.Eventually(testingT, func() bool {
		result, evalErr := page.Evaluate(`() => {
			var menu = document.querySelector('[data-loopaware-profile-menu="true"]');
			if (!menu) { return ''; }
			return menu.getAttribute('data-loopaware-dropdown-bound') || '';
		}`)
		if evalErr != nil {
			return false
		}
		return result == "true"
	}, webkitTestTimeout, webkitTestPollInterval, "profile menu dropdown binding did not complete")

	profileToggle, toggleErr := page.WaitForSelector(webkitProfileToggleSelector, playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(webkitTestTimeout.Milliseconds())),
	})
	require.NoError(testingT, toggleErr)
	require.NotNil(testingT, profileToggle)

	require.NoError(testingT, profileToggle.Click())

	menuItems, menuErr := page.WaitForSelector(webkitProfileMenuItemsSelector+".show", playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(webkitTestTimeout.Milliseconds())),
	})
	require.NoError(testingT, menuErr, "profile menu did not open")
	require.NotNil(testingT, menuItems)

	settingsButton, settingsErr := page.WaitForSelector(webkitSettingsButtonSelector, playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(webkitTestTimeout.Milliseconds())),
	})
	require.NoError(testingT, settingsErr)
	require.NotNil(testingT, settingsButton)

	require.NoError(testingT, settingsButton.Click())

	require.Eventually(testingT, func() bool {
		result, evalErr := page.Evaluate(`() => {
			var modal = document.getElementById('settings-modal');
			if (!modal) { return false; }
			return modal.classList.contains('show');
		}`)
		if evalErr != nil {
			return false
		}
		boolResult, ok := result.(bool)
		return ok && boolResult
	}, webkitTestTimeout, webkitTestPollInterval, "settings modal did not open in WebKit")

	settingsModal, modalErr := page.WaitForSelector(webkitSettingsModalSelector+".show", playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(webkitTestTimeout.Milliseconds())),
	})
	require.NoError(testingT, modalErr, "settings modal not visible in WebKit")
	require.NotNil(testingT, settingsModal)
}

func TestWebKitProfileMenuLogoutWorks(testingT *testing.T) {
	browser := setupWebKitBrowser(testingT)

	harness := buildDashboardIntegrationHarness(testingT, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(testingT, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	context, contextErr := browser.NewContext()
	require.NoError(testingT, contextErr)
	defer context.Close()

	setWebKitCookie(testingT, context, harness.server.URL, sessionCookie)

	page, pageErr := context.NewPage()
	require.NoError(testingT, pageErr)
	defer page.Close()

	_, navigateErr := page.Goto(harness.server.URL + "/app")
	require.NoError(testingT, navigateErr)

	require.Eventually(testingT, func() bool {
		result, evalErr := page.Evaluate(`() => {
			var header = document.querySelector('mpr-header');
			if (!header) { return ''; }
			return header.getAttribute('data-loopaware-auth-bound') || '';
		}`)
		if evalErr != nil {
			return false
		}
		return result == "true"
	}, webkitTestTimeout, webkitTestPollInterval, "auth binding did not complete")

	require.Eventually(testingT, func() bool {
		result, evalErr := page.Evaluate(`() => {
			var menu = document.querySelector('[data-loopaware-profile-menu="true"]');
			if (!menu) { return ''; }
			return menu.getAttribute('data-loopaware-dropdown-bound') || '';
		}`)
		if evalErr != nil {
			return false
		}
		return result == "true"
	}, webkitTestTimeout, webkitTestPollInterval, "profile menu dropdown binding did not complete")

	_, hookErr := page.Evaluate(`() => {
		if (window.sessionStorage) {
			window.sessionStorage.setItem('__webkitLogoutCalled', 'false');
		}
		var originalFetch = window.fetch ? window.fetch.bind(window) : null;
		window.fetch = function(input, options) {
			var url = '';
			if (typeof input === 'string') {
				url = input;
			} else if (input && typeof input.url === 'string') {
				url = input.url;
			}
			if (url.indexOf('/auth/logout') !== -1) {
				if (window.sessionStorage) {
					window.sessionStorage.setItem('__webkitLogoutCalled', 'true');
				}
				return Promise.resolve(new Response(null, { status: 204 }));
			}
			if (originalFetch) {
				return originalFetch(input, options);
			}
			return Promise.reject(new Error('fetch not available'));
		};
		window.logout = function() {
			return Promise.reject(new Error('logout failed'));
		};
		var originalLocationAssign = window.location.assign;
		window.location.assign = function(url) {
			if (window.sessionStorage) {
				window.sessionStorage.setItem('__webkitLogoutRedirect', url);
			}
		};
	}`)
	require.NoError(testingT, hookErr)

	profileToggle, toggleErr := page.WaitForSelector(webkitProfileToggleSelector, playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(webkitTestTimeout.Milliseconds())),
	})
	require.NoError(testingT, toggleErr)
	require.NotNil(testingT, profileToggle)

	require.NoError(testingT, profileToggle.Click())

	menuItems, menuErr := page.WaitForSelector(webkitProfileMenuItemsSelector+".show", playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(webkitTestTimeout.Milliseconds())),
	})
	require.NoError(testingT, menuErr, "profile menu did not open")
	require.NotNil(testingT, menuItems)

	logoutButton, logoutErr := page.WaitForSelector(webkitLogoutButtonSelector, playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(webkitTestTimeout.Milliseconds())),
	})
	require.NoError(testingT, logoutErr)
	require.NotNil(testingT, logoutButton)

	require.NoError(testingT, logoutButton.Click())

	require.Eventually(testingT, func() bool {
		result, evalErr := page.Evaluate(`() => {
			if (window.sessionStorage) {
				return window.sessionStorage.getItem('__webkitLogoutCalled') === 'true';
			}
			return false;
		}`)
		if evalErr != nil {
			return false
		}
		boolResult, ok := result.(bool)
		return ok && boolResult
	}, webkitTestTimeout, webkitTestPollInterval, "logout fetch was not called in WebKit")
}

func TestWebKitProfileMenuDropdownOpens(testingT *testing.T) {
	browser := setupWebKitBrowser(testingT)

	harness := buildDashboardIntegrationHarness(testingT, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(testingT, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	context, contextErr := browser.NewContext()
	require.NoError(testingT, contextErr)
	defer context.Close()

	setWebKitCookie(testingT, context, harness.server.URL, sessionCookie)

	page, pageErr := context.NewPage()
	require.NoError(testingT, pageErr)
	defer page.Close()

	_, navigateErr := page.Goto(harness.server.URL + "/app")
	require.NoError(testingT, navigateErr)

	require.Eventually(testingT, func() bool {
		result, evalErr := page.Evaluate(`() => {
			var header = document.querySelector('mpr-header');
			if (!header) { return ''; }
			return header.getAttribute('data-loopaware-auth-bound') || '';
		}`)
		if evalErr != nil {
			return false
		}
		return result == "true"
	}, webkitTestTimeout, webkitTestPollInterval, "auth binding did not complete")

	require.Eventually(testingT, func() bool {
		result, evalErr := page.Evaluate(`() => {
			var menu = document.querySelector('[data-loopaware-profile-menu="true"]');
			if (!menu) { return ''; }
			return menu.getAttribute('data-loopaware-dropdown-bound') || '';
		}`)
		if evalErr != nil {
			return false
		}
		return result == "true"
	}, webkitTestTimeout, webkitTestPollInterval, "profile menu dropdown binding did not complete")

	profileToggle, toggleErr := page.WaitForSelector(webkitProfileToggleSelector, playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(webkitTestTimeout.Milliseconds())),
	})
	require.NoError(testingT, toggleErr)
	require.NotNil(testingT, profileToggle)

	menuItemsBeforeClick, _ := page.QuerySelector(webkitProfileMenuItemsSelector + ".show")
	require.Nil(testingT, menuItemsBeforeClick, "menu should not be open before click")

	require.NoError(testingT, profileToggle.Click())

	menuItems, menuErr := page.WaitForSelector(webkitProfileMenuItemsSelector+".show", playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(webkitTestTimeout.Milliseconds())),
	})
	require.NoError(testingT, menuErr, "profile menu dropdown did not open in WebKit after clicking toggle")
	require.NotNil(testingT, menuItems)

	settingsVisible, settingsErr := page.WaitForSelector(webkitSettingsButtonSelector, playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(webkitTestTimeout.Milliseconds())),
	})
	require.NoError(testingT, settingsErr, "settings button not visible after menu opened")
	require.NotNil(testingT, settingsVisible)

	logoutVisible, logoutErr := page.WaitForSelector(webkitLogoutButtonSelector, playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(webkitTestTimeout.Milliseconds())),
	})
	require.NoError(testingT, logoutErr, "logout button not visible after menu opened")
	require.NotNil(testingT, logoutVisible)
}
