package httpapi_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
)

const (
	subscribeAllowedOriginsListSelector             = "#subscribe-allowed-origins-list"
	subscribeAllowedOriginsInputSelector            = subscribeAllowedOriginsListSelector + " input[data-subscribe-origin]"
	subscribeAllowedOriginsItemSelector             = subscribeAllowedOriginsListSelector + " input[data-subscribe-origin]:not([data-subscribe-origin-placeholder])"
	subscribeAllowedOriginsPlaceholderInputSelector = subscribeAllowedOriginsListSelector + " input[data-subscribe-origin-placeholder]"
	subscribeAllowedOriginsAddButtonSelector        = subscribeAllowedOriginsListSelector + " button[data-subscribe-origin-add]"
	subscribeAllowedOriginsRemoveButtonSelector     = subscribeAllowedOriginsListSelector + " button[data-subscribe-origin-remove]"
	subscribeAllowedOriginExample                   = "http://newsletter.example"
	subscribeAllowedOriginSecondary                 = "http://secondary.example"
)

func subscribeAllowedOriginsCountScript(expectedCount int) string {
	return fmt.Sprintf(`(function(){
      return document.querySelectorAll('%s').length === %d;
    }())`, subscribeAllowedOriginsInputSelector, expectedCount)
}

func subscribeAllowedOriginsMatchScript(primaryOrigin string, secondaryOrigin string) string {
	return fmt.Sprintf(`(function(){
      var inputs = document.querySelectorAll('%s');
      var values = [];
      for (var index = 0; index < inputs.length; index += 1) {
        var value = (inputs[index].value || '').trim();
        if (value) {
          values.push(value);
        }
      }
      if (values.length !== 2) {
        return false;
      }
      return values.indexOf('%s') !== -1 && values.indexOf('%s') !== -1;
    }())`, subscribeAllowedOriginsItemSelector, primaryOrigin, secondaryOrigin)
}

func TestDashboardPersistsSubscribeAllowedOrigins(testingT *testing.T) {
	harness := buildDashboardIntegrationHarness(testingT, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Subscribe Origins Dashboard",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(testingT, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(testingT)
	setPageCookie(testingT, page, harness.baseURL, sessionCookie)

	navigateToPage(testingT, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	clickSelector(testingT, page, dashboardSectionTabSubscriptionsSelector)

	waitForVisibleElement(testingT, page, subscribeAllowedOriginsPlaceholderInputSelector)
	waitForVisibleElement(testingT, page, subscribeAllowedOriginsAddButtonSelector)

	setInputValue(testingT, page, subscribeAllowedOriginsPlaceholderInputSelector, subscribeAllowedOriginExample)
	clickSelector(testingT, page, subscribeAllowedOriginsAddButtonSelector)

	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, page, subscribeAllowedOriginsCountScript(2))
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	clickSelector(testingT, page, subscribeAllowedOriginsRemoveButtonSelector)

	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, page, subscribeAllowedOriginsCountScript(1))
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	setInputValue(testingT, page, subscribeAllowedOriginsPlaceholderInputSelector, subscribeAllowedOriginExample)
	clickSelector(testingT, page, subscribeAllowedOriginsAddButtonSelector)

	require.Eventually(testingT, func() bool {
		var updated model.Site
		if err := harness.database.First(&updated, "id = ?", site.ID).Error; err != nil {
			return false
		}
		return updated.SubscribeAllowedOrigins == subscribeAllowedOriginExample
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardRehydratesSubscribeAllowedOriginsAfterLogin(testingT *testing.T) {
	harness := buildDashboardIntegrationHarness(testingT, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Subscribe Origins Dashboard Reload",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(testingT, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(testingT)
	setPageCookie(testingT, page, harness.baseURL, sessionCookie)

	navigateToPage(testingT, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	clickSelector(testingT, page, dashboardSectionTabSubscriptionsSelector)
	waitForVisibleElement(testingT, page, subscribeAllowedOriginsPlaceholderInputSelector)
	waitForVisibleElement(testingT, page, subscribeAllowedOriginsAddButtonSelector)

	setInputValue(testingT, page, subscribeAllowedOriginsPlaceholderInputSelector, subscribeAllowedOriginExample)
	clickSelector(testingT, page, subscribeAllowedOriginsAddButtonSelector)
	setInputValue(testingT, page, subscribeAllowedOriginsPlaceholderInputSelector, subscribeAllowedOriginSecondary)
	clickSelector(testingT, page, subscribeAllowedOriginsAddButtonSelector)

	require.Eventually(testingT, func() bool {
		var updated model.Site
		if err := harness.database.First(&updated, "id = ?", site.ID).Error; err != nil {
			return false
		}
		return updated.SubscribeAllowedOrigins == subscribeAllowedOriginExample+" "+subscribeAllowedOriginSecondary
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	reloadSessionCookie := createAuthenticatedSessionCookie(testingT, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	reloadPage := buildHeadlessPage(testingT)
	setPageCookie(testingT, reloadPage, harness.baseURL, reloadSessionCookie)

	navigateToPage(testingT, reloadPage, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, reloadPage, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, reloadPage, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	clickSelector(testingT, reloadPage, dashboardSectionTabSubscriptionsSelector)

	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, reloadPage, subscribeAllowedOriginsMatchScript(subscribeAllowedOriginExample, subscribeAllowedOriginSecondary))
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}
