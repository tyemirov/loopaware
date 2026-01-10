package httpapi_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
)

const (
	widgetAllowedOriginsListSelector              = "#widget-allowed-origins-list"
	widgetAllowedOriginsPlaceholderInputSelector  = widgetAllowedOriginsListSelector + " input[data-widget-origin-placeholder]"
	widgetAllowedOriginsAddButtonSelector         = widgetAllowedOriginsListSelector + " button[data-widget-origin-add]"
	widgetAllowedOriginExample                    = "http://widget.example"
	trafficAllowedOriginsListSelector             = "#traffic-allowed-origins-list"
	trafficAllowedOriginsPlaceholderInputSelector = trafficAllowedOriginsListSelector + " input[data-traffic-origin-placeholder]"
	trafficAllowedOriginsAddButtonSelector        = trafficAllowedOriginsListSelector + " button[data-traffic-origin-add]"
	trafficAllowedOriginExample                   = "http://traffic.example"
)

func TestDashboardPersistsWidgetAllowedOrigins(testingT *testing.T) {
	harness := buildDashboardIntegrationHarness(testingT, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Widget Origins Dashboard",
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

	waitForVisibleElement(testingT, page, widgetAllowedOriginsPlaceholderInputSelector)
	waitForVisibleElement(testingT, page, widgetAllowedOriginsAddButtonSelector)

	setInputValue(testingT, page, widgetAllowedOriginsPlaceholderInputSelector, widgetAllowedOriginExample)
	clickSelector(testingT, page, widgetAllowedOriginsAddButtonSelector)

	require.Eventually(testingT, func() bool {
		var updated model.Site
		if err := harness.database.First(&updated, "id = ?", site.ID).Error; err != nil {
			return false
		}
		return updated.WidgetAllowedOrigins == widgetAllowedOriginExample
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardPersistsTrafficAllowedOrigins(testingT *testing.T) {
	harness := buildDashboardIntegrationHarness(testingT, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Traffic Origins Dashboard",
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

	clickSelector(testingT, page, dashboardSectionTabTrafficSelector)

	waitForVisibleElement(testingT, page, trafficAllowedOriginsPlaceholderInputSelector)
	waitForVisibleElement(testingT, page, trafficAllowedOriginsAddButtonSelector)

	setInputValue(testingT, page, trafficAllowedOriginsPlaceholderInputSelector, trafficAllowedOriginExample)
	clickSelector(testingT, page, trafficAllowedOriginsAddButtonSelector)

	require.Eventually(testingT, func() bool {
		var updated model.Site
		if err := harness.database.First(&updated, "id = ?", site.ID).Error; err != nil {
			return false
		}
		return updated.TrafficAllowedOrigins == trafficAllowedOriginExample
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}
