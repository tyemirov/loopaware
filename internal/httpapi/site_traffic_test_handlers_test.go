package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

const (
	testTrafficSiteID      = "traffic-site-id"
	testTrafficSiteName    = "Traffic Site"
	testTrafficOwnerEmail  = "traffic-owner@example.com"
	testTrafficOriginValue = "https://traffic.example.com"
	testTrafficHost        = "traffic.local"
)

func buildTrafficHandlers(testingT *testing.T) *SiteTrafficTestHandlers {
	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	handlers := NewSiteTrafficTestHandlers(database, zap.NewNop(), AuthClientConfig{})
	return handlers
}

func TestTrafficTestPageRequiresSiteID(testingT *testing.T) {
	handlers := buildTrafficHandlers(testingT)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app/sites//traffic-test", nil)

	handlers.RenderTrafficTestPage(context)

	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestTrafficTestPageRequiresUser(testingT *testing.T) {
	handlers := buildTrafficHandlers(testingT)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "id", Value: testTrafficSiteID}}
	context.Request = httptest.NewRequest(http.MethodGet, "/app/sites/"+testTrafficSiteID+"/traffic-test", nil)

	handlers.RenderTrafficTestPage(context)

	require.Equal(testingT, http.StatusFound, recorder.Code)
	require.Equal(testingT, LandingPagePath, recorder.Header().Get("Location"))
}

func TestTrafficTestPageRendersHTML(testingT *testing.T) {
	handlers := buildTrafficHandlers(testingT)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "id", Value: testTrafficSiteID}}
	request := httptest.NewRequest(http.MethodGet, "https://"+testTrafficHost+"/app/sites/"+testTrafficSiteID+"/traffic-test", nil)
	context.Request = request
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testTrafficOwnerEmail, Role: RoleUser})

	handlers.RenderTrafficTestPage(context)

	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), testTrafficSiteID)
	require.Contains(testingT, recorder.Body.String(), "traffic-test-site-name")
	require.Contains(testingT, recorder.Body.String(), "Loading...")
	require.Contains(testingT, recorder.Body.String(), "/api/visits")
	require.Contains(testingT, recorder.Body.String(), "/api/sites/"+testTrafficSiteID+"/visits/stats")
}

func TestDefaultSampleURLFallbacks(testingT *testing.T) {
	handlers := &SiteTrafficTestHandlers{}
	require.Equal(testingT, "https://example.com/", handlers.defaultSampleURL(nil))

	request := &http.Request{URL: &url.URL{}}
	require.Equal(testingT, "https://example.com/", handlers.defaultSampleURL(request))
}
