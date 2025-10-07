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
	dashboardTitleText              = "LoopAware Dashboard"
	dashboardSessionContextKey      = "httpapi_current_user"
	testDashboardAuthenticatedEmail = "viewer@example.com"
	dashboardSitesListElementID     = "sites-list"
	dashboardNewSiteButtonElementID = "new-site-button"
	dashboardLegacySelectorID       = "site-selector"
	dashboardFooterBrandPrefix      = "Built by"
	dashboardFooterBrandURL         = "https://mprlab.com"
	dashboardFooterBrandName        = "Marco Polo Research Lab"
	dashboardButtonStatusToken      = "buttonStatusDisplayDuration"
	dashboardRestoreButtonToken     = "restoreButtonDefault"
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
