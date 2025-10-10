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
