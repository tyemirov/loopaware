package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/httpapi"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/testutil"
)

const (
	dashboardSessionContextKey = "httpapi_current_user"
	dashboardAdminEmail        = "dash-admin@example.com"
)

type dashboardTestHarness struct {
	handlers *httpapi.DashboardWebHandlers
	database *gorm.DB
}

func newDashboardTestHarness(t *testing.T) dashboardTestHarness {
	t.Helper()

	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, openErr)
	require.NoError(t, storage.AutoMigrate(database))

	service := httpapi.NewSiteService(database, zap.NewNop(), "https://dashboard.example")
	handlers := httpapi.NewDashboardWebHandlers(zap.NewNop(), service)

	return dashboardTestHarness{
		handlers: handlers,
		database: database,
	}
}

func TestDashboardPageRendersForAuthenticatedUser(t *testing.T) {
	harness := newDashboardTestHarness(t)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, httpapi.DashboardRoute, nil)
	context.Set(dashboardSessionContextKey, &httpapi.CurrentUser{Email: dashboardAdminEmail, IsAdmin: true})

	harness.handlers.RenderDashboard(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	require.Contains(t, body, "Sites")
	require.Contains(t, body, "Create site")
}

func TestDashboardPageRedirectsWhenUnauthenticated(t *testing.T) {
	harness := newDashboardTestHarness(t)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, httpapi.DashboardRoute, nil)

	harness.handlers.RenderDashboard(context)

	require.Equal(t, http.StatusFound, recorder.Code)
	require.Equal(t, "/login", recorder.Header().Get("Location"))
}
