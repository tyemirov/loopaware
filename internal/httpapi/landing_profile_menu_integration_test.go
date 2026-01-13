package httpapi_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLandingHeaderUsesAvatarOnlyProfileMenu(testingT *testing.T) {
	harness := buildDashboardIntegrationHarness(testingT, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(testingT, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(testingT)
	setPageCookie(testingT, page, harness.baseURL, sessionCookie)

	_, evalErr := page.EvalOnNewDocument(landingMarkAuthenticatedScript)
	require.NoError(testingT, evalErr)

	navigateToPage(testingT, page, harness.baseURL+dashboardTestLandingPath)

	require.Eventually(testingT, func() bool {
		return evaluateScriptString(testingT, page, `(function(){
			var header = document.querySelector('mpr-header');
			if (!header) { return ''; }
			return header.getAttribute('data-loopaware-auth-bound') || '';
		}())`) == "true"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, page, selectorExistsScript(webkitProfileMenuSelector))
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	clickSelector(testingT, page, dashboardProfileToggleSelector)

	var menuState struct {
		ToggleText  string `json:"toggleText"`
		MenuName    string `json:"menuName"`
		NameVisible bool   `json:"nameVisible"`
	}
	require.Eventually(testingT, func() bool {
		evaluateScriptInto(testingT, page, dashboardProfileMenuStateScript, &menuState)
		return len(menuState.ToggleText) == 0 && len(menuState.MenuName) > 0 && menuState.NameVisible
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	var defaultState struct {
		HasProfile bool `json:"hasProfile"`
	}
	evaluateScriptInto(testingT, page, dashboardHeaderDefaultProfileStateScript, &defaultState)
	require.False(testingT, defaultState.HasProfile)
}
