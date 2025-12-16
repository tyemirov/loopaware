package httpapi_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
)

func TestDashboardPersistsSubscribeAllowedOrigins(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Subscribe Origins Dashboard",
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

	clickSelector(t, page, dashboardSectionTabSubscriptionsSelector)

	waitForVisibleElement(t, page, "#subscribe-allowed-origins-add")
	waitForVisibleElement(t, page, `#subscribe-allowed-origins-list input[data-subscribe-origin]`)

	clickSelector(t, page, "#subscribe-allowed-origins-add")

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, `(function(){
      return document.querySelectorAll('#subscribe-allowed-origins-list input[data-subscribe-origin]').length === 2;
    }())`)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	clickSelector(t, page, `#subscribe-allowed-origins-list button[data-subscribe-origin-remove]`)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, `(function(){
      return document.querySelectorAll('#subscribe-allowed-origins-list input[data-subscribe-origin]').length === 1;
    }())`)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	setInputValue(t, page, `#subscribe-allowed-origins-list input[data-subscribe-origin]`, "http://newsletter.example")

	clickSelector(t, page, dashboardSaveSiteButtonSelector)

	require.Eventually(t, func() bool {
		var updated model.Site
		if err := harness.database.First(&updated, "id = ?", site.ID).Error; err != nil {
			return false
		}
		return updated.SubscribeAllowedOrigins == "http://newsletter.example"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}
