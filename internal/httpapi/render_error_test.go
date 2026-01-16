package httpapi

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	texttemplate "text/template"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

func TestRenderDashboardHandlesFooterAndAuthErrors(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)

	originalOverrides := footerVariantOverridesByKey
	footerVariantOverridesByKey = map[footerVariant]footerVariantOverrides{}
	testingT.Cleanup(func() {
		footerVariantOverridesByKey = originalOverrides
	})

	originalAuthTemplate := publicAuthScriptTemplate
	publicAuthScriptTemplate = texttemplate.Must(texttemplate.New("broken-auth").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		publicAuthScriptTemplate = originalAuthTemplate
	})

	handlers := NewDashboardWebHandlers(zap.NewNop(), "/", AuthClientConfig{})
	handlers.RenderDashboard(context)

	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Contains(testingT, recorder.Header().Get("Content-Type"), "text/html")
}

func TestRenderDashboardReportsTemplateError(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/app", nil)

	handlers := &DashboardWebHandlers{
		logger:      zap.NewNop(),
		template:    template.Must(template.New("broken-dashboard").Parse("{{.MissingField}}")),
		landingPath: "/",
	}

	handlers.RenderDashboard(context)

	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), "render_failed")
}

func TestRenderLandingPageHandlesAssetErrors(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	originalOverrides := footerVariantOverridesByKey
	footerVariantOverridesByKey = map[footerVariant]footerVariantOverrides{}
	testingT.Cleanup(func() {
		footerVariantOverridesByKey = originalOverrides
	})

	originalHeaderTemplate := publicHeaderTemplate
	publicHeaderTemplate = template.Must(template.New("broken-header").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		publicHeaderTemplate = originalHeaderTemplate
	})

	originalThemeTemplate := publicThemeScriptTemplate
	publicThemeScriptTemplate = template.Must(template.New("broken-theme").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		publicThemeScriptTemplate = originalThemeTemplate
	})

	originalAuthTemplate := publicAuthScriptTemplate
	publicAuthScriptTemplate = texttemplate.Must(texttemplate.New("broken-auth").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		publicAuthScriptTemplate = originalAuthTemplate
	})

	handlers := NewLandingPageHandlers(zap.NewNop(), nil, AuthClientConfig{})
	handlers.RenderLandingPage(context)

	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Contains(testingT, recorder.Header().Get("Content-Type"), "text/html")
}

func TestRenderLandingPageReportsTemplateError(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/login", nil)

	handlers := NewLandingPageHandlers(zap.NewNop(), nil, AuthClientConfig{})
	handlers.template = template.Must(template.New("broken-landing").Parse("{{.MissingField}}"))

	handlers.RenderLandingPage(context)

	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), "landing_render_failed")
}

func TestRenderPrivacyPageHandlesAssetErrors(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/privacy", nil)

	originalOverrides := footerVariantOverridesByKey
	footerVariantOverridesByKey = map[footerVariant]footerVariantOverrides{}
	testingT.Cleanup(func() {
		footerVariantOverridesByKey = originalOverrides
	})

	originalHeaderTemplate := publicHeaderTemplate
	publicHeaderTemplate = template.Must(template.New("broken-privacy-header").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		publicHeaderTemplate = originalHeaderTemplate
	})

	originalThemeTemplate := publicThemeScriptTemplate
	publicThemeScriptTemplate = template.Must(template.New("broken-privacy-theme").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		publicThemeScriptTemplate = originalThemeTemplate
	})

	originalAuthTemplate := publicAuthScriptTemplate
	publicAuthScriptTemplate = texttemplate.Must(texttemplate.New("broken-privacy-auth").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		publicAuthScriptTemplate = originalAuthTemplate
	})

	handlers := NewPrivacyPageHandlers(nil, AuthClientConfig{})
	handlers.RenderPrivacyPage(context)

	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Contains(testingT, recorder.Header().Get("Content-Type"), "text/html")
}

func TestRenderPrivacyPageReportsTemplateError(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/privacy", nil)

	handlers := NewPrivacyPageHandlers(nil, AuthClientConfig{})
	handlers.template = template.Must(template.New("broken-privacy").Parse("{{.MissingField}}"))

	handlers.RenderPrivacyPage(context)

	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), privacyRenderFailure)
}

func TestRenderSubscriptionConfirmationPageIncludesUnsubscribeLink(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/subscriptions/confirm", nil)

	handlers := &PublicHandlers{authConfig: AuthClientConfig{}}
	site := model.Site{Name: "LoopAware", AllowedOrigin: "https://example.com"}
	subscriber := model.Subscriber{
		Email:  "subscriber@example.com",
		Status: model.SubscriberStatusConfirmed,
	}

	handlers.renderSubscriptionConfirmationPage(context, http.StatusOK, "Heading", "Message", site, subscriber, "token-value")

	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), "/subscriptions/unsubscribe?token=token-value")
	require.Contains(testingT, recorder.Body.String(), "Open LoopAware")
}

func TestRenderSubscriptionConfirmationPageHandlesTemplateErrors(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/subscriptions/confirm", nil)

	originalOverrides := footerVariantOverridesByKey
	footerVariantOverridesByKey = map[footerVariant]footerVariantOverrides{}
	testingT.Cleanup(func() {
		footerVariantOverridesByKey = originalOverrides
	})

	originalHeaderTemplate := publicHeaderTemplate
	publicHeaderTemplate = template.Must(template.New("broken-subscription-header").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		publicHeaderTemplate = originalHeaderTemplate
	})

	originalThemeTemplate := publicThemeScriptTemplate
	publicThemeScriptTemplate = template.Must(template.New("broken-subscription-theme").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		publicThemeScriptTemplate = originalThemeTemplate
	})

	originalAuthTemplate := publicAuthScriptTemplate
	publicAuthScriptTemplate = texttemplate.Must(texttemplate.New("broken-subscription-auth").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		publicAuthScriptTemplate = originalAuthTemplate
	})

	originalConfirmationTemplate := subscriptionConfirmedTemplate
	subscriptionConfirmedTemplate = template.Must(template.New("broken-subscription-template").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		subscriptionConfirmedTemplate = originalConfirmationTemplate
	})

	handlers := &PublicHandlers{authConfig: AuthClientConfig{}, logger: zap.NewNop()}
	handlers.renderSubscriptionConfirmationPage(context, http.StatusBadRequest, "Heading", "Message", model.Site{}, model.Subscriber{}, "")

	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), "Message")
}
