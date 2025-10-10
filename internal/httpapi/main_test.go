package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/httpapi"
)

const (
	dashboardTitleText                      = "LoopAware Dashboard"
	dashboardSessionContextKey              = "httpapi_current_user"
	testDashboardAuthenticatedEmail         = "viewer@example.com"
	dashboardSitesListElementID             = "sites-list"
	dashboardNewSiteButtonElementID         = "new-site-button"
	dashboardLegacySelectorID               = "site-selector"
	dashboardFooterBrandPrefix              = "Built by"
	dashboardFooterBrandURL                 = "https://mprlab.com"
	dashboardFooterBrandName                = "Marco Polo Research Lab"
	dashboardButtonStatusToken              = "buttonStatusDisplayDuration"
	dashboardRestoreButtonToken             = "restoreButtonDefault"
	dashboardCreateButtonPattern            = "setButtonDefault(saveSiteButton, createButtonLabel, createButtonClass);"
	dashboardUpdateButtonPattern            = "setButtonDefault(saveSiteButton, updateButtonLabel, updateButtonClass);"
	dashboardButtonStylesToken              = "var buttonStyles = parsedConfig.button_styles || {};"
	dashboardButtonStylesPrimary            = "\"primary\":\"btn btn-outline-primary btn-sm\""
	dashboardButtonStylesSuccess            = "\"success\":\"btn btn-outline-success btn-sm\""
	dashboardButtonStylesSecondary          = "\"secondary\":\"btn btn-outline-secondary btn-sm\""
	dashboardButtonStylesDanger             = "\"danger\":\"btn btn-outline-danger btn-sm\""
	dashboardLegacyShowStatusFunction       = "function showStatus("
	dashboardNotificationTargetsToken       = "var notificationTargets ="
	dashboardLegacySiteSavedNotification    = "showStatus(statusMessages.siteSaved"
	dashboardLegacySiteCreatedNotification  = "showStatus(statusMessages.siteCreated"
	dashboardLegacyWidgetCopySuccess        = "showStatus(statusMessages.widgetCopied"
	dashboardLegacyWidgetCopyFailure        = "showStatus(statusMessages.widgetCopyFailed"
	dashboardLegacyRefreshLoading           = "showStatus('Refreshing...'"
	dashboardLegacyRefreshSuccess           = "showStatus('Feedback refreshed.'"
	dashboardLegacyRefreshFailure           = "showStatus('Refresh failed.'"
	dashboardLegacySelectSitePrompt         = "showStatus(statusMessages.selectSite"
	dashboardLoadMessagesSignature          = "function loadMessages(shouldUpdateButtonStatus)"
	dashboardManualLoadMessagesCall         = "loadMessages(true);"
	dashboardAutomaticLoadMessagesCall      = "loadMessages(false);"
	dashboardRefreshSuccessUpdateCall       = "updateButtonStatus(refreshMessagesButton, buttonLabels.refresh_success || '', buttonStyles.secondary || '');"
	dashboardRefreshFailureUpdateCall       = "updateButtonStatus(refreshMessagesButton, buttonLabels.refresh_failed || '', buttonStyles.danger || '');"
	dashboardRefreshLoadingUpdateCall       = "updateButtonStatus(refreshMessagesButton, buttonLabels.refresh_loading || '', buttonStyles.secondary || '');"
	dashboardSubmitGuardUpdateCall          = "updateButtonStatus(saveSiteButton, statusMessages.select_site || '', buttonStyles.secondary || '');"
	dashboardSaveButtonClassMarkup          = "class=\"btn btn-outline-success btn-sm\""
	dashboardNewSiteButtonClass             = "class=\"btn btn-outline-primary btn-sm\""
	dashboardLegacySaveButtonClass          = "btn btn-success\""
	dashboardDeleteButtonClassMarkup        = "class=\"btn btn-sm border-0 bg-transparent text-danger opacity-100 disabled\""
	dashboardDeleteIconMarkup               = "class=\"bi bi-trash3-fill text-danger\""
	dashboardFooterElementID                = "id=\"dashboard-footer\""
	dashboardFooterThemeConfigToken         = "\"footer_theme_classes\":{"
	dashboardBootstrapIconsIntegrityToken   = "integrity=\"sha384-XGjxtQfXaH2tnPFa9x+ruJTuLE3Aa6LhHSWRr1XeTyhezb4abCG4ccI5AkVDxqC+\""
	dashboardFaviconLinkToken               = "rel=\"icon\""
	dashboardValidationMessagesToken        = "\"validation_messages\":{"
	dashboardValidationScriptToken          = "var validationMessages = parsedConfig.validation_messages || {};"
	dashboardValidationGuardToken           = "if (!validateSiteForm()) {"
	dashboardValidationResetToken           = "function clearValidationFeedback() {"
	dashboardSiteNameHelpButtonID           = "site-name-help-button"
	dashboardAllowedOriginHelpButtonID      = "allowed-origin-help-button"
	dashboardOwnerEmailHelpButtonID         = "owner-email-help-button"
	dashboardFieldHelpPopoverToken          = "data-bs-toggle=\"popover\""
	dashboardMailtoPrefixToken              = "var mailtoSchemePrefix = 'mailto:';"
	dashboardRenderContactFunctionToken     = "function renderContactValue(cell, value)"
	dashboardContactLinkHrefToken           = "link.href = mailtoSchemePrefix + normalized;"
	dashboardContactAppendLinkToken         = "cell.appendChild(link);"
	dashboardSiteListHeaderClassToken       = "\"site_list_item_header\":\"d-flex align-items-center gap-2\""
	dashboardSiteListFaviconClassToken      = "\"site_list_item_favicon\":\"flex-shrink-0 rounded border bg-white\""
	dashboardFaviconURLToken                = "var faviconURL = (site.favicon_url || '').trim();"
	dashboardFaviconSrcAssignmentToken      = "faviconElement.src = faviconURL;"
	dashboardFaviconErrorHandlerToken       = "faviconElement.classList.add('d-none');"
	dashboardSiteCreatedAtElementID         = "site-created-at"
	dashboardFeedbackCountElementID         = "feedback-count"
	dashboardSiteCreatedAtVarToken          = "var siteCreatedAtElement = document.getElementById(elementIds.site_created_at);"
	dashboardFeedbackCountVarToken          = "var feedbackCountElement = document.getElementById(elementIds.feedback_count);"
	dashboardFeedbackCountHiddenClassToken  = "class=\"badge bg-secondary d-none\""
	dashboardFeedbackCountHideCallToken     = "feedbackCountElement.classList.add('d-none');"
	dashboardFeedbackCountShowCallToken     = "feedbackCountElement.classList.remove('d-none');"
	dashboardSetFeedbackCountToken          = "function setFeedbackCount(total, visible)"
	dashboardUpdateSelectedSiteSummaryToken = "function updateSelectedSiteSummary(site)"
	dashboardRegisteredPrefixToken          = "Registered at:"
	dashboardDateFormatterToken             = "date.toLocaleDateString()"
	dashboardLegacyDateFormatterToken       = "date.toLocaleString()"
	dashboardFormStatusSuccessThemeToken    = "bg-success-subtle"
	dashboardFormStatusDangerThemeToken     = "bg-danger-subtle"
	dashboardFormStatusDurationToken        = "var formStatusDisplayDuration ="
	dashboardFormStatusTimeoutToken         = "formStatusResetTimerId = window.setTimeout"
	dashboardFormStatusClearTimeoutToken    = "clearTimeout(formStatusResetTimerId)"
	dashboardFooterDropdownToggleToken      = "data-bs-toggle=\"dropdown\""
	dashboardFooterDropdownMenuToken        = "dropdown-menu"
	dashboardFooterLinkGravityToken         = "https://gravity.mprlab.com"
	dashboardFooterLinkLoopAwareToken       = "https://loopaware.mprlab.com"
	dashboardFooterLinkAllergyToken         = "https://allergy.mprlab.com"
	dashboardFooterLinkThreaderToken        = "https://threader.mprlab.com"
	dashboardFooterLinkRSVPToken            = "https://rsvp.mprlab.com"
	dashboardFooterLinkCountdownToken       = "https://countdown.mprlab.com"
	dashboardFooterLinkCrosswordToken       = "https://llm-crossword.mprlab.com"
	dashboardFooterLinkPromptsToken         = "https://prompts.mprlab.com"
	dashboardFooterLinkWallpapersToken      = "https://wallpapers.mprlab.com"
)

func TestDashboardPageRendersForAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/html")
	require.Contains(t, recorder.Body.String(), dashboardTitleText)
}

func TestDashboardTemplateUsesSitesListPanel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	body := recorder.Body.String()
	testCases := []struct {
		testName      string
		substring     string
		expectPresent bool
	}{
		{
			testName:      "sites list container",
			substring:     "id=\"" + dashboardSitesListElementID + "\"",
			expectPresent: true,
		},
		{
			testName:      "new site button",
			substring:     "id=\"" + dashboardNewSiteButtonElementID + "\"",
			expectPresent: true,
		},
		{
			testName:      "delete site button class",
			substring:     dashboardDeleteButtonClassMarkup,
			expectPresent: true,
		},
		{
			testName:      "delete site icon class",
			substring:     dashboardDeleteIconMarkup,
			expectPresent: true,
		},
		{
			testName:      "site list header class exported",
			substring:     dashboardSiteListHeaderClassToken,
			expectPresent: true,
		},
		{
			testName:      "site list favicon class exported",
			substring:     dashboardSiteListFaviconClassToken,
			expectPresent: true,
		},
		{
			testName:      "bootstrap icons integrity",
			substring:     dashboardBootstrapIconsIntegrityToken,
			expectPresent: true,
		},
		{
			testName:      "favicon link present",
			substring:     dashboardFaviconLinkToken,
			expectPresent: true,
		},
		{
			testName:      "site created at element",
			substring:     "id=\"" + dashboardSiteCreatedAtElementID + "\"",
			expectPresent: true,
		},
		{
			testName:      "feedback count element",
			substring:     "id=\"" + dashboardFeedbackCountElementID + "\"",
			expectPresent: true,
		},
		{
			testName:      "feedback count hidden by default",
			substring:     dashboardFeedbackCountHiddenClassToken,
			expectPresent: true,
		},
		{
			testName:      "feedback count hide call",
			substring:     dashboardFeedbackCountHideCallToken,
			expectPresent: true,
		},
		{
			testName:      "feedback count show call",
			substring:     dashboardFeedbackCountShowCallToken,
			expectPresent: true,
		},
		{
			testName:      "footer element id",
			substring:     dashboardFooterElementID,
			expectPresent: true,
		},
		{
			testName:      "legacy site selector removed",
			substring:     "id=\"" + dashboardLegacySelectorID + "\"",
			expectPresent: false,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.testName, func(t *testing.T) {
			if testCase.expectPresent {
				require.Contains(t, body, testCase.substring)
				return
			}
			require.NotContains(t, body, testCase.substring)
		})
	}
}

func TestDashboardFooterIncludesBranding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	body := recorder.Body.String()
	testCases := []struct {
		testName      string
		substring     string
		expectPresent bool
	}{
		{
			testName:      "footer prefix",
			substring:     dashboardFooterBrandPrefix,
			expectPresent: true,
		},
		{
			testName:      "footer link text",
			substring:     dashboardFooterBrandName,
			expectPresent: true,
		},
		{
			testName:      "footer link target",
			substring:     dashboardFooterBrandURL,
			expectPresent: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.testName, func(t *testing.T) {
			if testCase.expectPresent {
				require.Contains(t, body, testCase.substring)
				return
			}
			require.NotContains(t, body, testCase.substring)
		})
	}
}

func TestDashboardFooterDisplaysProductMenu(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	body := recorder.Body.String()
	require.Contains(t, body, dashboardFooterDropdownToggleToken)
	require.Contains(t, body, dashboardFooterDropdownMenuToken)
	require.Contains(t, body, dashboardFooterLinkGravityToken)
	require.Contains(t, body, dashboardFooterLinkLoopAwareToken)
	require.Contains(t, body, dashboardFooterLinkAllergyToken)
	require.Contains(t, body, dashboardFooterLinkThreaderToken)
	require.Contains(t, body, dashboardFooterLinkRSVPToken)
	require.Contains(t, body, dashboardFooterLinkCountdownToken)
	require.Contains(t, body, dashboardFooterLinkCrosswordToken)
	require.Contains(t, body, dashboardFooterLinkPromptsToken)
	require.Contains(t, body, dashboardFooterLinkWallpapersToken)
}

func TestDashboardTemplateDisplaysRegistrationInline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	body := recorder.Body.String()
	require.Contains(t, body, dashboardRegisteredPrefixToken)
	require.NotContains(t, body, "text-muted small text-end mt-3")
}

func TestDashboardTimestampFormattedAsDateOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	body := recorder.Body.String()
	require.Contains(t, body, dashboardDateFormatterToken)
	require.NotContains(t, body, dashboardLegacyDateFormatterToken)
}

func TestDashboardFormStatusUsesThemeAwareBackgrounds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	body := recorder.Body.String()
	require.Contains(t, body, dashboardFormStatusSuccessThemeToken)
	require.Contains(t, body, dashboardFormStatusDangerThemeToken)
}

func TestDashboardFormStatusClearsAfterTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	body := recorder.Body.String()
	require.Contains(t, body, dashboardFormStatusDurationToken)
	require.Contains(t, body, dashboardFormStatusTimeoutToken)
	require.Contains(t, body, dashboardFormStatusClearTimeoutToken)
}

func TestDashboardTemplateConfiguresButtonStatusManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	body := recorder.Body.String()
	testCases := []struct {
		testName      string
		substring     string
		expectPresent bool
	}{
		{
			testName:      "status duration token",
			substring:     dashboardButtonStatusToken,
			expectPresent: true,
		},
		{
			testName:      "restore helper token",
			substring:     dashboardRestoreButtonToken,
			expectPresent: true,
		},
		{
			testName:      "create mode styling",
			substring:     dashboardCreateButtonPattern,
			expectPresent: true,
		},
		{
			testName:      "update mode styling",
			substring:     dashboardUpdateButtonPattern,
			expectPresent: true,
		},
		{
			testName:      "button styles map declared",
			substring:     dashboardButtonStylesToken,
			expectPresent: true,
		},
		{
			testName:      "primary outline class",
			substring:     dashboardButtonStylesPrimary,
			expectPresent: true,
		},
		{
			testName:      "success outline class",
			substring:     dashboardButtonStylesSuccess,
			expectPresent: true,
		},
		{
			testName:      "secondary outline class",
			substring:     dashboardButtonStylesSecondary,
			expectPresent: true,
		},
		{
			testName:      "danger outline class",
			substring:     dashboardButtonStylesDanger,
			expectPresent: true,
		},
		{
			testName:      "load messages signature",
			substring:     dashboardLoadMessagesSignature,
			expectPresent: true,
		},
		{
			testName:      "manual refresh call",
			substring:     dashboardManualLoadMessagesCall,
			expectPresent: true,
		},
		{
			testName:      "automatic refresh call",
			substring:     dashboardAutomaticLoadMessagesCall,
			expectPresent: true,
		},
		{
			testName:      "refresh loading update",
			substring:     dashboardRefreshLoadingUpdateCall,
			expectPresent: true,
		},
		{
			testName:      "footer theme config present",
			substring:     dashboardFooterThemeConfigToken,
			expectPresent: true,
		},
		{
			testName:      "refresh success update",
			substring:     dashboardRefreshSuccessUpdateCall,
			expectPresent: true,
		},
		{
			testName:      "refresh failure update",
			substring:     dashboardRefreshFailureUpdateCall,
			expectPresent: true,
		},
		{
			testName:      "submit guard uses button update",
			substring:     dashboardSubmitGuardUpdateCall,
			expectPresent: true,
		},
		{
			testName:      "legacy showStatus helper removed",
			substring:     dashboardLegacyShowStatusFunction,
			expectPresent: false,
		},
		{
			testName:      "notification targets removed",
			substring:     dashboardNotificationTargetsToken,
			expectPresent: false,
		},
		{
			testName:      "legacy site saved badge removed",
			substring:     dashboardLegacySiteSavedNotification,
			expectPresent: false,
		},
		{
			testName:      "legacy site created badge removed",
			substring:     dashboardLegacySiteCreatedNotification,
			expectPresent: false,
		},
		{
			testName:      "legacy widget copy success badge removed",
			substring:     dashboardLegacyWidgetCopySuccess,
			expectPresent: false,
		},
		{
			testName:      "legacy widget copy failure badge removed",
			substring:     dashboardLegacyWidgetCopyFailure,
			expectPresent: false,
		},
		{
			testName:      "legacy refresh loading badge removed",
			substring:     dashboardLegacyRefreshLoading,
			expectPresent: false,
		},
		{
			testName:      "legacy refresh success badge removed",
			substring:     dashboardLegacyRefreshSuccess,
			expectPresent: false,
		},
		{
			testName:      "legacy refresh failure badge removed",
			substring:     dashboardLegacyRefreshFailure,
			expectPresent: false,
		},
		{
			testName:      "legacy select site badge removed",
			substring:     dashboardLegacySelectSitePrompt,
			expectPresent: false,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.testName, func(t *testing.T) {
			if testCase.expectPresent {
				require.Contains(t, body, testCase.substring)
				return
			}
			require.NotContains(t, body, testCase.substring)
		})
	}
}

func TestDashboardTemplateIncludesSiteValidationSupport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	body := recorder.Body.String()
	testCases := []struct {
		testName      string
		substring     string
		expectPresent bool
	}{
		{
			testName:      "validation messages config present",
			substring:     dashboardValidationMessagesToken,
			expectPresent: true,
		},
		{
			testName:      "validation script bootstrap",
			substring:     dashboardValidationScriptToken,
			expectPresent: true,
		},
		{
			testName:      "validation guard in submit handler",
			substring:     dashboardValidationGuardToken,
			expectPresent: true,
		},
		{
			testName:      "validation reset helper present",
			substring:     dashboardValidationResetToken,
			expectPresent: true,
		},
		{
			testName:      "site name help button present",
			substring:     "id=\"" + dashboardSiteNameHelpButtonID + "\"",
			expectPresent: true,
		},
		{
			testName:      "allowed origin help button present",
			substring:     "id=\"" + dashboardAllowedOriginHelpButtonID + "\"",
			expectPresent: true,
		},
		{
			testName:      "owner email help button present",
			substring:     "id=\"" + dashboardOwnerEmailHelpButtonID + "\"",
			expectPresent: true,
		},
		{
			testName:      "field help uses popover",
			substring:     dashboardFieldHelpPopoverToken,
			expectPresent: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.testName, func(t *testing.T) {
			if testCase.expectPresent {
				require.Contains(t, body, testCase.substring)
				return
			}
			require.NotContains(t, body, testCase.substring)
		})
	}
}

func TestDashboardTemplateUsesUniformActionButtonStyles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	body := recorder.Body.String()
	testCases := []struct {
		testName      string
		substring     string
		expectPresent bool
	}{
		{
			testName:      "save button uses outline class",
			substring:     dashboardSaveButtonClassMarkup,
			expectPresent: true,
		},
		{
			testName:      "new site button uses shared outline class",
			substring:     dashboardNewSiteButtonClass,
			expectPresent: true,
		},
		{
			testName:      "legacy solid success class removed",
			substring:     dashboardLegacySaveButtonClass,
			expectPresent: false,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.testName, func(t *testing.T) {
			if testCase.expectPresent {
				require.Contains(t, body, testCase.substring)
				return
			}
			require.NotContains(t, body, testCase.substring)
		})
	}
}

func TestDashboardTemplateSupportsMailtoLinksForFeedback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	body := recorder.Body.String()
	testCases := []struct {
		testName      string
		substring     string
		expectPresent bool
	}{
		{
			testName:      "defines mailto prefix constant",
			substring:     dashboardMailtoPrefixToken,
			expectPresent: true,
		},
		{
			testName:      "defines render contact helper",
			substring:     dashboardRenderContactFunctionToken,
			expectPresent: true,
		},
		{
			testName:      "assigns mailto href",
			substring:     dashboardContactLinkHrefToken,
			expectPresent: true,
		},
		{
			testName:      "appends anchor to cell",
			substring:     dashboardContactAppendLinkToken,
			expectPresent: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.testName, func(t *testing.T) {
			if testCase.expectPresent {
				require.Contains(t, body, testCase.substring)
				return
			}
			require.NotContains(t, body, testCase.substring)
		})
	}
}

func TestDashboardTemplateSupportsSiteFavicons(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	body := recorder.Body.String()
	testCases := []struct {
		testName      string
		substring     string
		expectPresent bool
	}{
		{
			testName:      "favicon url extraction",
			substring:     dashboardFaviconURLToken,
			expectPresent: true,
		},
		{
			testName:      "favicon source assignment",
			substring:     dashboardFaviconSrcAssignmentToken,
			expectPresent: true,
		},
		{
			testName:      "favicon error handler",
			substring:     dashboardFaviconErrorHandlerToken,
			expectPresent: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.testName, func(t *testing.T) {
			if testCase.expectPresent {
				require.Contains(t, body, testCase.substring)
				return
			}
			require.NotContains(t, body, testCase.substring)
		})
	}
}

func TestDashboardTemplateExposesSiteMetadataHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: testDashboardAuthenticatedEmail})

	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop())
	handlers.RenderDashboard(context)

	body := recorder.Body.String()
	testCases := []struct {
		testName      string
		substring     string
		expectPresent bool
	}{
		{
			testName:      "site created at accessor",
			substring:     dashboardSiteCreatedAtVarToken,
			expectPresent: true,
		},
		{
			testName:      "feedback count accessor",
			substring:     dashboardFeedbackCountVarToken,
			expectPresent: true,
		},
		{
			testName:      "feedback count helper",
			substring:     dashboardSetFeedbackCountToken,
			expectPresent: true,
		},
		{
			testName:      "site summary helper",
			substring:     dashboardUpdateSelectedSiteSummaryToken,
			expectPresent: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.testName, func(t *testing.T) {
			if testCase.expectPresent {
				require.Contains(t, body, testCase.substring)
				return
			}
			require.NotContains(t, body, testCase.substring)
		})
	}
}
