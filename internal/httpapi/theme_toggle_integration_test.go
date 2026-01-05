package httpapi_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/httpapi"
)

const (
	landingThemeToggleControlSelector   = `mpr-footer [data-mpr-footer="theme-toggle"] [data-mpr-theme-toggle="control"]`
	dashboardThemeToggleControlSelector = `#dashboard-footer [data-mpr-footer="theme-toggle"] [data-mpr-theme-toggle="control"]`
	dashboardThemePaletteScript         = `(function() {
		function parseRGB(value) {
			if (!value) { return null; }
			var match = value.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
			if (!match) { return null; }
			return { r: Number(match[1]), g: Number(match[2]), b: Number(match[3]) };
		}
		function readBackground(element) {
			if (!element) { return null; }
			var style = window.getComputedStyle(element);
			return parseRGB(style ? style.backgroundColor : '');
		}
		var body = document.body;
		var headerHost = document.querySelector('mpr-header');
		var headerRoot = null;
		if (headerHost) {
			headerRoot = headerHost.querySelector('header.mpr-header');
			if (!headerRoot && headerHost.shadowRoot) {
				headerRoot = headerHost.shadowRoot.querySelector('header.mpr-header');
			}
		}
		var footerHost = document.getElementById('dashboard-footer');
		var footerRoot = null;
		if (footerHost) {
			footerRoot = footerHost.querySelector('[data-mpr-footer="root"]');
			if (!footerRoot && footerHost.shadowRoot) {
				footerRoot = footerHost.shadowRoot.querySelector('[data-mpr-footer="root"]');
			}
		}
		var bodyColor = readBackground(body);
		var headerColor = readBackground(headerRoot || headerHost);
		var footerColor = readBackground(footerRoot || footerHost);
		function delta(first, second) {
			if (!first || !second) { return 999; }
			return Math.abs(first.r - second.r) + Math.abs(first.g - second.g) + Math.abs(first.b - second.b);
		}
		return {
			headerDelta: delta(bodyColor, headerColor),
			footerDelta: delta(bodyColor, footerColor)
		};
	}())`
)

func TestLandingThemeToggleMatchesThemePreferenceAndMapping(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	page := buildHeadlessPage(t)

	testCases := []struct {
		name            string
		theme           string
		expectedChecked bool
	}{
		{name: "light", theme: "light", expectedChecked: false},
		{name: "dark", theme: "dark", expectedChecked: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			navigateToPage(t, page, harness.baseURL+dashboardTestLandingPath)

			seedScript := fmt.Sprintf(`localStorage.setItem('loopaware_public_theme','%s');localStorage.setItem('loopaware_landing_theme','%s');localStorage.setItem('landing_theme','%s');`, testCase.theme, testCase.theme, testCase.theme)
			evaluateScriptInto(t, page, seedScript, nil)

			navigateToPage(t, page, harness.baseURL+dashboardTestLandingPath)

			require.Eventually(t, func() bool {
				return evaluateScriptBoolean(t, page, selectorExistsScript(landingThemeToggleControlSelector))
			}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

			require.Equal(t, testCase.theme, evaluateScriptString(t, page, dashboardDocumentThemeScript))
			require.Equal(t, testCase.theme, evaluateScriptString(t, page, dashboardDocumentMprThemeScript))
			require.Equal(t, testCase.expectedChecked, evaluateScriptBoolean(t, page, toggleCheckedScript(landingThemeToggleControlSelector)))
		})
	}
}

func TestPrivacyThemeToggleMatchesThemePreferenceAndMapping(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	page := buildHeadlessPage(t)

	testCases := []struct {
		name            string
		theme           string
		expectedChecked bool
	}{
		{name: "light", theme: "light", expectedChecked: false},
		{name: "dark", theme: "dark", expectedChecked: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			navigateToPage(t, page, harness.baseURL+dashboardTestLandingPath)
			seedScript := fmt.Sprintf(`localStorage.setItem('loopaware_public_theme','%s');localStorage.setItem('loopaware_landing_theme','%s');localStorage.setItem('landing_theme','%s');`, testCase.theme, testCase.theme, testCase.theme)
			evaluateScriptInto(t, page, seedScript, nil)

			navigateToPage(t, page, harness.baseURL+httpapi.PrivacyPagePath)

			require.Eventually(t, func() bool {
				return evaluateScriptBoolean(t, page, selectorExistsScript(landingThemeToggleControlSelector))
			}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

			require.Equal(t, testCase.theme, evaluateScriptString(t, page, dashboardDocumentThemeScript))
			require.Equal(t, testCase.theme, evaluateScriptString(t, page, dashboardDocumentMprThemeScript))
			require.Equal(t, testCase.expectedChecked, evaluateScriptBoolean(t, page, toggleCheckedScript(landingThemeToggleControlSelector)))
		})
	}
}

func TestDashboardThemeToggleUsesLightLeftDarkRight(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestLandingPath)
	evaluateScriptInto(t, page, `localStorage.setItem('loopaware_public_theme','light');localStorage.setItem('loopaware_landing_theme','light');localStorage.removeItem('loopaware_dashboard_theme');localStorage.removeItem('loopaware_theme');`, nil)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, selectorExistsScript(dashboardThemeToggleControlSelector))
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Equal(t, "light", evaluateScriptString(t, page, dashboardDocumentThemeScript))
	require.False(t, evaluateScriptBoolean(t, page, toggleCheckedScript(dashboardThemeToggleControlSelector)))

	clickSelector(t, page, dashboardThemeToggleControlSelector)

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardDocumentThemeScript) == "dark"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.True(t, evaluateScriptBoolean(t, page, toggleCheckedScript(dashboardThemeToggleControlSelector)))
	require.Equal(t, "dark", evaluateScriptString(t, page, dashboardStoredDashboardThemeScript))
	require.Equal(t, "dark", evaluateScriptString(t, page, dashboardStoredPublicThemeScript))
}

func TestDashboardPaletteMatchesHeaderAndFooterThemes(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	testCases := []struct {
		name  string
		theme string
	}{
		{name: "light", theme: "light"},
		{name: "dark", theme: "dark"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			navigateToPage(t, page, harness.baseURL+dashboardTestLandingPath)
			seedScript := fmt.Sprintf(`localStorage.setItem('loopaware_public_theme','%s');localStorage.setItem('loopaware_landing_theme','%s');localStorage.setItem('loopaware_dashboard_theme','%s');localStorage.setItem('loopaware_theme','%s');`, testCase.theme, testCase.theme, testCase.theme, testCase.theme)
			evaluateScriptInto(t, page, seedScript, nil)

			navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)

			require.Eventually(t, func() bool {
				return evaluateScriptString(t, page, dashboardDocumentThemeScript) == testCase.theme
			}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

			var palette struct {
				HeaderDelta float64 `json:"headerDelta"`
				FooterDelta float64 `json:"footerDelta"`
			}
			evaluateScriptInto(t, page, dashboardThemePaletteScript, &palette)
			require.Less(t, palette.HeaderDelta, 15.0)
			require.Less(t, palette.FooterDelta, 15.0)
		})
	}
}

func selectorExistsScript(selector string) string {
	return fmt.Sprintf(`(function(){
    return !!document.querySelector(%q);
  }())`, selector)
}

func toggleCheckedScript(selector string) string {
	return fmt.Sprintf(`(function(){
    var control = document.querySelector(%q);
    return !!(control && control.checked);
  }())`, selector)
}
