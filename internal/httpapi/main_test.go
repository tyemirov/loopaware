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
