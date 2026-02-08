package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

const (
	testWidgetTestSiteID             = "widget-site-id"
	testWidgetTestOwnerEmail         = "widget-owner@example.com"
	testWidgetTestOtherEmail         = "other@example.com"
	testWidgetTestMissingSiteID      = "missing"
	testWidgetTestFeedbackPath       = "/api/sites/" + testWidgetTestSiteID + "/widget-test/feedback"
	testWidgetTestCreateCallbackName = "force_widget_feedback_create_error"
	testWidgetTestCreateErrorMessage = "feedback_create_failed"
	testWidgetTestContactEmail       = "contact@example.com"
	testWidgetTestMessage            = "Test widget message"
)

type widgetTestHarness struct {
	handlers *SiteWidgetTestHandlers
	database *gorm.DB
}

func buildWidgetTestHarness(testingT *testing.T) widgetTestHarness {
	testingT.Helper()
	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	feedbackBroadcaster := NewFeedbackEventBroadcaster()
	testingT.Cleanup(feedbackBroadcaster.Close)

	handlers := NewSiteWidgetTestHandlers(database, zap.NewNop(), feedbackBroadcaster, nil)
	return widgetTestHarness{handlers: handlers, database: database}
}

func buildWidgetTestContext(method string, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	context.Request = request
	return context, recorder
}

func insertWidgetTestSite(testingT *testing.T, database *gorm.DB) {
	testingT.Helper()
	site := model.Site{
		ID:           testWidgetTestSiteID,
		Name:         "Widget Site",
		OwnerEmail:   testWidgetTestOwnerEmail,
		CreatorEmail: testWidgetTestOwnerEmail,
	}
	require.NoError(testingT, database.Create(&site).Error)
}

func TestSubmitWidgetTestFeedbackRequiresSiteID(testingT *testing.T) {
	harness := buildWidgetTestHarness(testingT)
	context, recorder := buildWidgetTestContext(http.MethodPost, "/api/sites//widget-test/feedback", nil)

	harness.handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestSubmitWidgetTestFeedbackRequiresUser(testingT *testing.T) {
	harness := buildWidgetTestHarness(testingT)
	context, recorder := buildWidgetTestContext(http.MethodPost, testWidgetTestFeedbackPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testWidgetTestSiteID}}

	harness.handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusUnauthorized, recorder.Code)
}

func TestSubmitWidgetTestFeedbackReturnsNotFound(testingT *testing.T) {
	harness := buildWidgetTestHarness(testingT)
	context, recorder := buildWidgetTestContext(http.MethodPost, "/api/sites/missing/widget-test/feedback", nil)
	context.Params = gin.Params{{Key: "id", Value: testWidgetTestMissingSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testWidgetTestOwnerEmail, Role: RoleUser})

	harness.handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusNotFound, recorder.Code)
}

func TestSubmitWidgetTestFeedbackRejectsForbidden(testingT *testing.T) {
	harness := buildWidgetTestHarness(testingT)
	insertWidgetTestSite(testingT, harness.database)

	context, recorder := buildWidgetTestContext(http.MethodPost, testWidgetTestFeedbackPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testWidgetTestSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testWidgetTestOtherEmail, Role: RoleUser})

	harness.handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)
}

func TestSubmitWidgetTestFeedbackRequiresFields(testingT *testing.T) {
	harness := buildWidgetTestHarness(testingT)
	insertWidgetTestSite(testingT, harness.database)

	context, recorder := buildWidgetTestContext(http.MethodPost, testWidgetTestFeedbackPath, []byte(`{"contact":"","message":""}`))
	context.Params = gin.Params{{Key: "id", Value: testWidgetTestSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testWidgetTestOwnerEmail, Role: RoleUser})

	harness.handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestSubmitWidgetTestFeedbackRejectsInvalidJSON(testingT *testing.T) {
	harness := buildWidgetTestHarness(testingT)
	insertWidgetTestSite(testingT, harness.database)

	context, recorder := buildWidgetTestContext(http.MethodPost, testWidgetTestFeedbackPath, []byte("{"))
	context.Params = gin.Params{{Key: "id", Value: testWidgetTestSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testWidgetTestOwnerEmail, Role: RoleUser})

	harness.handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestSubmitWidgetTestFeedbackReportsSaveError(testingT *testing.T) {
	harness := buildWidgetTestHarness(testingT)
	insertWidgetTestSite(testingT, harness.database)

	callbackName := testWidgetTestCreateCallbackName
	harness.database.Callback().Create().Before("gorm:create").Register(callbackName, func(database *gorm.DB) {
		database.AddError(errors.New(testWidgetTestCreateErrorMessage))
	})
	testingT.Cleanup(func() {
		harness.database.Callback().Create().Remove(callbackName)
	})

	payload := map[string]string{
		"contact": testWidgetTestContactEmail,
		"message": testWidgetTestMessage,
	}
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)

	context, recorder := buildWidgetTestContext(http.MethodPost, testWidgetTestFeedbackPath, body)
	context.Params = gin.Params{{Key: "id", Value: testWidgetTestSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testWidgetTestOwnerEmail, Role: RoleUser})

	harness.handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)

	var savedFeedback model.Feedback
	require.Error(testingT, harness.database.First(&savedFeedback).Error)
}

func TestSubmitWidgetTestFeedbackCreatesRecord(testingT *testing.T) {
	harness := buildWidgetTestHarness(testingT)
	insertWidgetTestSite(testingT, harness.database)

	payload := map[string]string{
		"contact": testWidgetTestContactEmail,
		"message": testWidgetTestMessage,
	}
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)

	context, recorder := buildWidgetTestContext(http.MethodPost, testWidgetTestFeedbackPath, body)
	context.Params = gin.Params{{Key: "id", Value: testWidgetTestSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testWidgetTestOwnerEmail, Role: RoleUser})

	harness.handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var storedFeedback model.Feedback
	require.NoError(testingT, harness.database.First(&storedFeedback).Error)
	require.Equal(testingT, testWidgetTestContactEmail, storedFeedback.Contact)
	require.Equal(testingT, testWidgetTestMessage, storedFeedback.Message)
}
