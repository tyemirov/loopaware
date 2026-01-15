package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	texttemplate "text/template"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

func failingTextTemplate(name string) *texttemplate.Template {
	return texttemplate.Must(texttemplate.New(name).Funcs(texttemplate.FuncMap{
		"fail": func() (string, error) {
			return "", errors.New("template failure")
		},
	}).Parse("{{fail}}"))
}

func TestRenderWidgetTemplateReportsExecuteError(testingT *testing.T) {
	originalTemplate := widgetJavaScriptTemplate
	widgetJavaScriptTemplate = failingTextTemplate("widget-error")
	testingT.Cleanup(func() {
		widgetJavaScriptTemplate = originalTemplate
	})

	site := model.Site{ID: "widget-site", WidgetBubbleSide: widgetBubbleSideRight, WidgetBubbleBottomOffsetPx: 16}
	script, renderErr := renderWidgetTemplate(site)
	require.Error(testingT, renderErr)
	require.Empty(testingT, script)
}

func TestRenderSubscribeTemplateReportsExecuteError(testingT *testing.T) {
	originalTemplate := subscribeJavaScriptTemplate
	subscribeJavaScriptTemplate = failingTextTemplate("subscribe-error")
	testingT.Cleanup(func() {
		subscribeJavaScriptTemplate = originalTemplate
	})

	site := model.Site{ID: "subscribe-site"}
	script, renderErr := renderSubscribeTemplate(site)
	require.Error(testingT, renderErr)
	require.Empty(testingT, script)
}

func TestRenderPixelTemplateReportsExecuteError(testingT *testing.T) {
	originalTemplate := pixelJavaScriptTemplate
	pixelJavaScriptTemplate = failingTextTemplate("pixel-error")
	testingT.Cleanup(func() {
		pixelJavaScriptTemplate = originalTemplate
	})

	site := model.Site{ID: "pixel-site"}
	script, renderErr := renderPixelTemplate(site)
	require.Error(testingT, renderErr)
	require.Empty(testingT, script)
}

func TestWidgetJSReportsTemplateError(testingT *testing.T) {
	originalTemplate := widgetJavaScriptTemplate
	widgetJavaScriptTemplate = failingTextTemplate("widget-handler-error")
	testingT.Cleanup(func() {
		widgetJavaScriptTemplate = originalTemplate
	})

	database := openFaviconManagerDatabase(testingT)
	site := createTestSite(testingT, database)
	feedbackBroadcaster := NewFeedbackEventBroadcaster()
	subscriptionEvents := NewSubscriptionTestEventBroadcaster()
	testingT.Cleanup(feedbackBroadcaster.Close)
	testingT.Cleanup(subscriptionEvents.Close)
	handlers := NewPublicHandlers(database, zap.NewNop(), feedbackBroadcaster, subscriptionEvents, nil, nil, true, "http://loopaware.test", "secret", nil, AuthClientConfig{})

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/widget.js?site_id="+site.ID, nil)

	handlers.WidgetJS(ginContext)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), "render error")
}

func TestSubscribeJSReportsTemplateError(testingT *testing.T) {
	originalTemplate := subscribeJavaScriptTemplate
	subscribeJavaScriptTemplate = failingTextTemplate("subscribe-handler-error")
	testingT.Cleanup(func() {
		subscribeJavaScriptTemplate = originalTemplate
	})

	database := openFaviconManagerDatabase(testingT)
	site := createTestSite(testingT, database)
	feedbackBroadcaster := NewFeedbackEventBroadcaster()
	subscriptionEvents := NewSubscriptionTestEventBroadcaster()
	testingT.Cleanup(feedbackBroadcaster.Close)
	testingT.Cleanup(subscriptionEvents.Close)
	handlers := NewPublicHandlers(database, zap.NewNop(), feedbackBroadcaster, subscriptionEvents, nil, nil, true, "http://loopaware.test", "secret", nil, AuthClientConfig{})

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/subscribe.js?site_id="+site.ID, nil)

	handlers.SubscribeJS(ginContext)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), "render error")
}

func TestPixelJSReportsTemplateError(testingT *testing.T) {
	originalTemplate := pixelJavaScriptTemplate
	pixelJavaScriptTemplate = failingTextTemplate("pixel-handler-error")
	testingT.Cleanup(func() {
		pixelJavaScriptTemplate = originalTemplate
	})

	database := openFaviconManagerDatabase(testingT)
	site := createTestSite(testingT, database)
	feedbackBroadcaster := NewFeedbackEventBroadcaster()
	subscriptionEvents := NewSubscriptionTestEventBroadcaster()
	testingT.Cleanup(feedbackBroadcaster.Close)
	testingT.Cleanup(subscriptionEvents.Close)
	handlers := NewPublicHandlers(database, zap.NewNop(), feedbackBroadcaster, subscriptionEvents, nil, nil, true, "http://loopaware.test", "secret", nil, AuthClientConfig{})

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/pixel.js?site_id="+site.ID, nil)

	handlers.PixelJS(ginContext)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), "render error")
}
