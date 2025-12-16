package httpapi_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	landingFooterMenuButtonSelector   = `mpr-footer [data-mpr-footer="toggle-button"]`
	dashboardFooterMenuButtonSelector = `#dashboard-footer [data-mpr-footer="toggle-button"]`
)

func TestLandingFooterMenuLabelUsesReadableColorInBothThemes(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	page := buildHeadlessPage(t)

	testCases := []struct {
		name          string
		theme         string
		expectedColor string
	}{
		{name: "light", theme: "light", expectedColor: "rgb(15, 23, 42)"},
		{name: "dark", theme: "dark", expectedColor: "rgb(226, 232, 240)"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			navigateToPage(t, page, harness.baseURL+dashboardTestLandingPath)

			seedScript := fmt.Sprintf(`localStorage.setItem('loopaware_public_theme','%s');localStorage.setItem('loopaware_landing_theme','%s');`, testCase.theme, testCase.theme)
			evaluateScriptInto(t, page, seedScript, nil)

			navigateToPage(t, page, harness.baseURL+dashboardTestLandingPath)

			require.Eventually(t, func() bool {
				return evaluateScriptBoolean(t, page, selectorExistsScript(landingFooterMenuButtonSelector))
			}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

			require.Eventually(t, func() bool {
				menuButtonText := evaluateScriptString(t, page, elementTextContentScript(landingFooterMenuButtonSelector))
				menuButtonClass := evaluateScriptString(t, page, elementClassNameScript(landingFooterMenuButtonSelector))
				return menuButtonText != "" && menuButtonClass != ""
			}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

			menuButtonText := evaluateScriptString(t, page, elementTextContentScript(landingFooterMenuButtonSelector))
			require.Contains(t, menuButtonText, dashboardFooterBrandPrefix)
			require.Contains(t, menuButtonText, dashboardFooterBrandName)
			require.Contains(t, evaluateScriptString(t, page, elementClassNameScript(landingFooterMenuButtonSelector)), "mpr-footer__menu-button")

			menuButtonColor := evaluateScriptString(t, page, computedColorScript(landingFooterMenuButtonSelector))
			require.Equal(t, testCase.expectedColor, menuButtonColor)
		})
	}
}

func TestDashboardFooterMenuLabelUsesReadableColorInLightTheme(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestLandingPath)
	evaluateScriptInto(t, page, `localStorage.setItem('loopaware_public_theme','light');localStorage.setItem('loopaware_landing_theme','light');localStorage.removeItem('loopaware_dashboard_theme');`, nil)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, selectorExistsScript(dashboardFooterMenuButtonSelector))
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(t, func() bool {
		menuButtonText := evaluateScriptString(t, page, elementTextContentScript(dashboardFooterMenuButtonSelector))
		menuButtonClass := evaluateScriptString(t, page, elementClassNameScript(dashboardFooterMenuButtonSelector))
		return menuButtonText != "" && menuButtonClass != ""
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	menuButtonColor := evaluateScriptString(t, page, computedColorScript(dashboardFooterMenuButtonSelector))
	require.Equal(t, "rgb(15, 23, 42)", menuButtonColor)

	require.Equal(t, "light", evaluateScriptString(t, page, dashboardDocumentThemeScript))
}

func computedColorScript(selector string) string {
	return fmt.Sprintf(`(function(){
    var element = document.querySelector(%q);
    if (!element) { return ''; }
    var style = window.getComputedStyle(element);
    return style && style.color ? style.color : '';
  }())`, selector)
}

func elementTextContentScript(selector string) string {
	return fmt.Sprintf(`(function(){
    var element = document.querySelector(%q);
    return element ? String(element.textContent || '').trim() : '';
  }())`, selector)
}

func elementClassNameScript(selector string) string {
	return fmt.Sprintf(`(function(){
    var element = document.querySelector(%q);
    return element ? String(element.getAttribute('class') || '').trim() : '';
  }())`, selector)
}
