package httpapi_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLandingHeaderUsesUserMenu(testingT *testing.T) {
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
		return evaluateScriptBoolean(testingT, page, selectorExistsScript(dashboardUserMenuTriggerSelector))
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	openDashboardUserMenu(testingT, page)
	openDashboardUserMenu(testingT, page)

	var menuState struct {
		LoopawareMenuCount       int  `json:"loopawareMenuCount"`
		HeaderUserMenuCount      int  `json:"headerUserMenuCount"`
		ExtraHeaderUserMenuCount int  `json:"extraHeaderUserMenuCount"`
		HeaderAvatarCount        int  `json:"headerAvatarCount"`
		VisibleHeaderAvatarCount int  `json:"visibleHeaderAvatarCount"`
		AvatarCount              int  `json:"avatarCount"`
		AvatarVisible            bool `json:"avatarVisible"`
		NameVisible              bool `json:"nameVisible"`
	}
	require.Eventually(testingT, func() bool {
		evaluateScriptInto(testingT, page, dashboardUserMenuStateScript, &menuState)
		return menuState.LoopawareMenuCount == 1 && menuState.HeaderUserMenuCount == 1 && menuState.ExtraHeaderUserMenuCount == 0 && menuState.HeaderAvatarCount == 1 && menuState.VisibleHeaderAvatarCount == 1 && menuState.AvatarCount == 1 && menuState.AvatarVisible
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	var defaultState struct {
		HasUserMenu          bool `json:"hasUserMenu"`
		HasLegacyProfileMenu bool `json:"hasLegacyProfileMenu"`
	}
	evaluateScriptInto(testingT, page, dashboardHeaderDefaultProfileStateScript, &defaultState)
	require.True(testingT, defaultState.HasUserMenu)
	require.False(testingT, defaultState.HasLegacyProfileMenu)

	accountSettingsLabel := evaluateScriptString(testingT, page, `(function(){
		var menu = document.querySelector('mpr-user[data-loopaware-user-menu="true"]');
		if (!menu) { return ''; }
		var item = menu.querySelector('[data-mpr-user="menu-item"]');
		if (!item) { return ''; }
		return String(item.textContent || '').trim();
	}())`)
	require.Equal(testingT, "Account settings", accountSettingsLabel)

	logoutLabel := evaluateScriptString(testingT, page, `(function(){
		var menu = document.querySelector('mpr-user[data-loopaware-user-menu="true"]');
		if (!menu) { return ''; }
		var item = menu.querySelector('[data-mpr-user="logout"]');
		if (!item) { return ''; }
		return String(item.textContent || '').trim();
	}())`)
	require.Equal(testingT, "Logout", logoutLabel)
}
