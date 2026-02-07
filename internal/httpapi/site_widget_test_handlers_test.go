package httpapi

import (
	"bytes"
	"crypto/tls"
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
	testWidgetSiteID                 = "widget-site-id"
	testWidgetSiteName               = "Widget Site"
	testWidgetOwnerEmail             = "widget-owner@example.com"
	testWidgetOtherEmail             = "other@example.com"
	testWidgetBaseURL                = "https://widget.example.com"
	testWidgetContactEmail           = "contact@example.com"
	testWidgetMessage                = "Test widget message"
	testWidgetScriptHost             = "widget.local"
	testWidgetMissingSiteID          = "missing"
	testWidgetFeedbackPath           = "/api/sites/" + testWidgetSiteID + "/widget-test/feedback"
	testWidgetRenderPath             = "/app/sites/" + testWidgetSiteID + "/widget-test"
	testWidgetFeedbackCreateCallback = "force_widget_feedback_create_error"
	testWidgetFeedbackCreateError    = "feedback_create_failed"
)

func buildWidgetHandlers(testingT *testing.T) (*SiteWidgetTestHandlers, *model.Site) {
	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	feedbackBroadcaster := NewFeedbackEventBroadcaster()
	testingT.Cleanup(feedbackBroadcaster.Close)

	handlers := NewSiteWidgetTestHandlers(database, zap.NewNop(), testWidgetBaseURL, feedbackBroadcaster, nil, AuthClientConfig{})
	site := &model.Site{
		ID:                         testWidgetSiteID,
		Name:                       testWidgetSiteName,
		OwnerEmail:                 testWidgetOwnerEmail,
		AllowedOrigin:              testWidgetBaseURL,
		WidgetBubbleSide:           widgetBubbleSideRight,
		WidgetBubbleBottomOffsetPx: defaultWidgetBubbleBottomOffset,
	}
	require.NoError(testingT, database.Create(site).Error)
	return handlers, site
}

func buildWidgetContext(method string, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	context.Request = request
	return context, recorder
}

func TestFormatWidgetPlacementSide(testingT *testing.T) {
	require.Equal(testingT, "Left", formatWidgetPlacementSide(widgetBubbleSideLeft))
	require.Equal(testingT, "Right", formatWidgetPlacementSide(widgetBubbleSideRight))
	require.Equal(testingT, "Right", formatWidgetPlacementSide("invalid"))
}

func TestResolveWidgetScriptURLUsesRequestHost(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	request := httptest.NewRequest(http.MethodGet, "http://"+testWidgetScriptHost+"/", nil)
	resolved := handlers.resolveWidgetScriptURL(request, testWidgetSiteID)
	require.Contains(testingT, resolved, "http://"+testWidgetScriptHost)
	require.Contains(testingT, resolved, testWidgetSiteID)
}

func TestResolveWidgetScriptURLUsesTLS(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	request := httptest.NewRequest(http.MethodGet, "http://"+testWidgetScriptHost+"/", nil)
	request.TLS = &tls.ConnectionState{}
	resolved := handlers.resolveWidgetScriptURL(request, testWidgetSiteID)
	require.Contains(testingT, resolved, "https://"+testWidgetScriptHost)
}

func TestResolveWidgetScriptURLUsesForwardedProto(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	request := httptest.NewRequest(http.MethodGet, "http://"+testWidgetScriptHost+"/", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	resolved := handlers.resolveWidgetScriptURL(request, testWidgetSiteID)
	require.Contains(testingT, resolved, "https://"+testWidgetScriptHost)
}

func TestResolveWidgetScriptURLFallsBackToBase(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	resolved := handlers.resolveWidgetScriptURL(nil, testWidgetSiteID)
	require.Contains(testingT, resolved, testWidgetBaseURL)
}

func TestResolveWidgetScriptURLReturnsEmptyForMissingSiteID(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	require.Equal(testingT, "", handlers.resolveWidgetScriptURL(nil, ""))
}

func TestRenderWidgetTestPageRequiresSiteID(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	context, recorder := buildWidgetContext(http.MethodGet, "/app/sites//widget-test", nil)

	handlers.RenderWidgetTestPage(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestRenderWidgetTestPageRequiresUser(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	context, recorder := buildWidgetContext(http.MethodGet, testWidgetRenderPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testWidgetSiteID}}

	handlers.RenderWidgetTestPage(context)
	require.Equal(testingT, http.StatusFound, recorder.Code)
}

func TestRenderWidgetTestPageRendersHTML(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	context, recorder := buildWidgetContext(http.MethodGet, testWidgetRenderPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testWidgetSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testWidgetOwnerEmail, Role: RoleUser})

	handlers.RenderWidgetTestPage(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), testWidgetSiteID)
	require.Contains(testingT, recorder.Body.String(), "widget-test-site-name")
	require.Contains(testingT, recorder.Body.String(), "Loading...")
	require.Contains(testingT, recorder.Body.String(), "window.LOOPAWARE_WIDGET_TEST_MODE = true;")
	require.Contains(testingT, recorder.Body.String(), "LOOPAWARE_WIDGET_TEST_ENDPOINT")
	require.Contains(testingT, recorder.Body.String(), "widget-test\\/feedback")
}

func TestSubmitWidgetTestFeedbackRequiresSiteID(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	context, recorder := buildWidgetContext(http.MethodPost, "/api/sites//widget-test/feedback", nil)

	handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestSubmitWidgetTestFeedbackRequiresUser(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	context, recorder := buildWidgetContext(http.MethodPost, testWidgetFeedbackPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testWidgetSiteID}}

	handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusUnauthorized, recorder.Code)
}

func TestSubmitWidgetTestFeedbackReturnsNotFound(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	context, recorder := buildWidgetContext(http.MethodPost, "/api/sites/missing/widget-test/feedback", nil)
	context.Params = gin.Params{{Key: "id", Value: testWidgetMissingSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testWidgetOwnerEmail, Role: RoleUser})

	handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusNotFound, recorder.Code)
}

func TestSubmitWidgetTestFeedbackRejectsForbidden(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	context, recorder := buildWidgetContext(http.MethodPost, testWidgetFeedbackPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testWidgetSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testWidgetOtherEmail, Role: RoleUser})

	handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)
}

func TestSubmitWidgetTestFeedbackRequiresFields(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	context, recorder := buildWidgetContext(http.MethodPost, testWidgetFeedbackPath, []byte(`{"contact":"","message":""}`))
	context.Params = gin.Params{{Key: "id", Value: testWidgetSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testWidgetOwnerEmail, Role: RoleUser})

	handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestSubmitWidgetTestFeedbackRejectsInvalidJSON(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	context, recorder := buildWidgetContext(http.MethodPost, testWidgetFeedbackPath, []byte("{"))
	context.Params = gin.Params{{Key: "id", Value: testWidgetSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testWidgetOwnerEmail, Role: RoleUser})

	handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestSubmitWidgetTestFeedbackReportsSaveError(testingT *testing.T) {
	handlers, _ := buildWidgetHandlers(testingT)
	callbackName := testWidgetFeedbackCreateCallback
	handlers.database.Callback().Create().Before("gorm:create").Register(callbackName, func(database *gorm.DB) {
		database.AddError(errors.New(testWidgetFeedbackCreateError))
	})
	testingT.Cleanup(func() {
		handlers.database.Callback().Create().Remove(callbackName)
	})

	payload := widgetTestFeedbackRequest{
		Contact: testWidgetContactEmail,
		Message: testWidgetMessage,
	}
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)

	context, recorder := buildWidgetContext(http.MethodPost, testWidgetFeedbackPath, body)
	context.Params = gin.Params{{Key: "id", Value: testWidgetSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testWidgetOwnerEmail, Role: RoleUser})

	handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
}

func TestSubmitWidgetTestFeedbackCreatesRecord(testingT *testing.T) {
	handlers, site := buildWidgetHandlers(testingT)
	payload := widgetTestFeedbackRequest{
		Contact: testWidgetContactEmail,
		Message: testWidgetMessage,
	}
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)

	context, recorder := buildWidgetContext(http.MethodPost, testWidgetFeedbackPath, body)
	context.Params = gin.Params{{Key: "id", Value: testWidgetSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testWidgetOwnerEmail, Role: RoleUser})

	handlers.SubmitWidgetTestFeedback(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var storedFeedback model.Feedback
	require.NoError(testingT, handlers.database.First(&storedFeedback, "site_id = ?", site.ID).Error)
	require.Equal(testingT, testWidgetContactEmail, storedFeedback.Contact)
	require.Equal(testingT, testWidgetMessage, storedFeedback.Message)
}
