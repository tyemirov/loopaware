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
	landingDetailedAuthCopyToken     = "Google Sign-In powered by GAuss keeps every login secure."
	landingDetailedWidgetCopyToken   = "Origin-locked widgets and APIs capture feedback where customers already are."
	landingDetailedWorkflowCopyToken = "Role-aware workflows assign owners, surface trends, and track resolution."
	landingThemeToggleIDToken        = "id=\"landing-theme-toggle\""
	landingThemeScriptKeyToken       = "var landingThemeStorageKey = 'landing_theme'"
	landingThemeApplyFunctionToken   = "function applyLandingTheme(theme)"
	landingThemeDataAttributeToken   = "data-bs-theme"
)

func TestLandingPageIncludesDetailedCopy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingDetailedAuthCopyToken)
	require.Contains(t, body, landingDetailedWidgetCopyToken)
	require.Contains(t, body, landingDetailedWorkflowCopyToken)
}

func TestLandingPageProvidesThemeSwitch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handlers := httpapi.NewLandingPageHandlers(zap.NewNop())
	handlers.RenderLandingPage(context)

	body := recorder.Body.String()
	require.Contains(t, body, landingThemeToggleIDToken)
	require.Contains(t, body, landingThemeScriptKeyToken)
	require.Contains(t, body, landingThemeApplyFunctionToken)
	require.Contains(t, body, landingThemeDataAttributeToken)
}
