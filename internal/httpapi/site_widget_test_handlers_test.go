package httpapi_test

import (
	"bytes"
	"context"
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

	handler := httpapi.NewSiteWidgetTestHandlers(database, zap.NewNop(), "http://localhost:8080", nil, nil)

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
	require.Contains(t, body, "id=\"widget-test-form\"")
	require.Contains(t, body, "id=\"widget-test-save\"")
	require.Contains(t, body, "id=\"widget-test-bottom-offset\"")
	require.Contains(t, body, "id=\"settings-button\"")
	require.NotContains(t, body, "id=\"settings-theme-toggle\"")
	require.Contains(t, body, "data-mpr-footer=\"theme-toggle-input\"")
	require.Contains(t, body, "id=\"logout-button\"")
	require.Contains(t, body, "id=\"dashboard-footer\"")
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

	handler := httpapi.NewSiteWidgetTestHandlers(database, zap.NewNop(), "http://localhost:8080", nil, nil)

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

func TestSubmitWidgetTestFeedbackNotifiesAndUpdatesDelivery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database := openWidgetTestDatabase(t)
	defer closeWidgetTestDatabase(t, database)

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Notifier Site",
		AllowedOrigin:              "https://notify.example",
		OwnerEmail:                 "owner@example.com",
		CreatorEmail:               "owner@example.com",
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 48,
	}
	require.NoError(t, database.Create(&site).Error)

	recordingNotifier := &widgetTestRecordingNotifier{delivery: model.FeedbackDeliveryMailed}
	handler := httpapi.NewSiteWidgetTestHandlers(database, zap.NewNop(), "http://localhost:8080", nil, recordingNotifier)

	payload := map[string]string{
		"contact": "captain@example.com",
		"message": "Requesting notification delivery status",
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
	require.Equal(t, 1, recordingNotifier.callCount)
	require.Equal(t, site.ID, recordingNotifier.lastSiteID)

	var stored model.Feedback
	require.NoError(t, database.First(&stored, "site_id = ?", site.ID).Error)
	require.Equal(t, recordingNotifier.lastFeedbackID, stored.ID)
	require.Equal(t, model.FeedbackDeliveryMailed, stored.Delivery)
}

type widgetTestRecordingNotifier struct {
	delivery       string
	callCount      int
	lastSiteID     string
	lastFeedbackID string
}

func (notifier *widgetTestRecordingNotifier) NotifyFeedback(ctx context.Context, site model.Site, feedback model.Feedback) (string, error) {
	notifier.callCount++
	notifier.lastSiteID = site.ID
	notifier.lastFeedbackID = feedback.ID
	return notifier.delivery, nil
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
