package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/httpapi"
	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

func TestExamplePageDemoMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := httpapi.NewExamplePageHandlers(zap.NewNop(), nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/example", nil)
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request

	handler.RenderExamplePage(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	require.Contains(t, body, `<script>window.LOOPAWARE_WIDGET_DEMO_MODE = true;</script>`)
	require.Contains(t, body, `demo mode`)
	require.Contains(t, body, `&lt;script defer src="/widget.js?site_id=__loopaware_widget_demo__"&gt;&lt;/script&gt;`)
	require.Equal(t, 1, strings.Count(body, "<script defer src="))
}

func TestExamplePageHonorsProvidedSiteIdentifier(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := openExampleTestDatabase(t)
	defer closeExampleTestDatabase(t, database)

	site := model.Site{
		ID:                         "example-site-id",
		Name:                       "Example Site",
		AllowedOrigin:              "http://localhost:8080",
		OwnerEmail:                 "owner@example.com",
		CreatorEmail:               "owner@example.com",
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 16,
	}
	require.NoError(t, database.Create(&site).Error)

	handler := httpapi.NewExamplePageHandlers(zap.NewNop(), database)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/example?site_id="+site.ID, nil)
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request

	handler.RenderExamplePage(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	require.NotContains(t, body, `window.LOOPAWARE_WIDGET_DEMO_MODE`)
	require.Contains(t, body, site.ID)
	require.Contains(t, body, site.Name)
}

func TestWidgetJSDemoUsesLeftBubble(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := zap.NewNop()
	handler := httpapi.NewPublicHandlers(nil, logger, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/widget.js?site_id=__loopaware_widget_demo__", nil)
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request

	handler.WidgetJS(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `var widgetPlacementSideValue = "left";`)
}

func openExampleTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)
	gormDatabase, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, err)
	gormDatabase = testutil.ConfigureDatabaseLogger(t, gormDatabase)
	require.NoError(t, storage.AutoMigrate(gormDatabase))
	return gormDatabase
}

func closeExampleTestDatabase(t *testing.T, database *gorm.DB) {
	t.Helper()
	sqlDatabase, err := database.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDatabase.Close())
}
