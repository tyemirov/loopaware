package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestRenderWidgetTestPage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := openWidgetTestDatabase(t)
	defer closeWidgetTestDatabase(t, database)

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Preview Site",
		AllowedOrigin:              "https://preview.example",
		OwnerEmail:                 "owner@example.com",
		CreatorEmail:               "owner@example.com",
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 24,
	}
	require.NoError(t, database.Create(&site).Error)

	handler := httpapi.NewSiteWidgetTestHandlers(database, zap.NewNop(), "http://localhost:8080", nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/app/sites/"+site.ID+"/widget-test", nil)
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set("httpapi_current_user", &httpapi.CurrentUser{Email: site.OwnerEmail, Role: httpapi.RoleUser})

	handler.RenderWidgetTestPage(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	require.Contains(t, body, "Widget Test — ")
	require.Contains(t, body, site.Name)
	require.Contains(t, body, site.ID)
	require.Contains(t, body, "LOOPAWARE_WIDGET_TEST_MODE")
}

func TestSubmitWidgetTestFeedback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := openWidgetTestDatabase(t)
	defer closeWidgetTestDatabase(t, database)

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Feedback Site",
		AllowedOrigin:              "https://feedback.example",
		OwnerEmail:                 "owner@example.com",
		CreatorEmail:               "owner@example.com",
		WidgetBubbleSide:           "left",
		WidgetBubbleBottomOffsetPx: 32,
	}
	require.NoError(t, database.Create(&site).Error)

	handler := httpapi.NewSiteWidgetTestHandlers(database, zap.NewNop(), "http://localhost:8080", nil)

	payload := map[string]string{
		"contact": "tester@example.com",
		"message": "Hello from test widget",
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/app/sites/"+site.ID+"/widget-test/feedback", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set("httpapi_current_user", &httpapi.CurrentUser{Email: site.OwnerEmail, Role: httpapi.RoleUser})

	handler.SubmitWidgetTestFeedback(context)

	require.Equal(t, http.StatusOK, recorder.Code)

	var stored model.Feedback
	require.NoError(t, database.First(&stored, "site_id = ?", site.ID).Error)
	require.Equal(t, payload["contact"], stored.Contact)
	require.Equal(t, payload["message"], stored.Message)
}

func openWidgetTestDatabase(testingT *testing.T) *gorm.DB {
	testingT.Helper()
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	gormDatabase, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, err)
	gormDatabase = testutil.ConfigureDatabaseLogger(testingT, gormDatabase)
	require.NoError(testingT, storage.AutoMigrate(gormDatabase))
	return gormDatabase
}

func closeWidgetTestDatabase(testingT *testing.T, database *gorm.DB) {
	testingT.Helper()
	sqlDatabase, err := database.DB()
	require.NoError(testingT, err)
	require.NoError(testingT, sqlDatabase.Close())
}
