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
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
)

func createTestSite(testingT *testing.T, database *gorm.DB) model.Site {
	testingT.Helper()
	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Test Site",
		AllowedOrigin: testFaviconOrigin,
		OwnerEmail:    "owner@example.com",
		CreatorEmail:  "owner@example.com",
	}
	require.NoError(testingT, database.Create(&site).Error)
	return site
}

func newTestPageContext(testingT *testing.T, path string, site model.Site) (*httptest.ResponseRecorder, *gin.Context) {
	testingT.Helper()
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, path, nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: site.OwnerEmail, Role: RoleAdmin})
	return recorder, context
}

func TestRenderWidgetTestPageHandlesFooterAndAuthErrors(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	site := createTestSite(testingT, database)
	recorder, context := newTestPageContext(testingT, "/app/sites/"+site.ID+"/widget-test", site)

	originalOverrides := footerVariantOverridesByKey
	footerVariantOverridesByKey = map[footerVariant]footerVariantOverrides{}
	testingT.Cleanup(func() {
		footerVariantOverridesByKey = originalOverrides
	})

	originalAuthTemplate := publicAuthScriptTemplate
	publicAuthScriptTemplate = texttemplate.Must(texttemplate.New("broken-widget-auth").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		publicAuthScriptTemplate = originalAuthTemplate
	})

	handlers := NewSiteWidgetTestHandlers(database, zap.NewNop(), "https://widgets.example", nil, nil, AuthClientConfig{})
	handlers.RenderWidgetTestPage(context)

	require.Equal(testingT, http.StatusOK, recorder.Code)
}

func TestRenderWidgetTestPageReportsTemplateError(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	site := createTestSite(testingT, database)
	recorder, context := newTestPageContext(testingT, "/app/sites/"+site.ID+"/widget-test", site)

	handlers := NewSiteWidgetTestHandlers(database, zap.NewNop(), "https://widgets.example", nil, nil, AuthClientConfig{})
	handlers.template = template.Must(template.New("broken-widget-template").Parse("{{.MissingField}}"))
	handlers.RenderWidgetTestPage(context)

	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
}

func TestRenderSubscribeTestPageHandlesFooterAndAuthErrors(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	site := createTestSite(testingT, database)
	recorder, context := newTestPageContext(testingT, "/app/sites/"+site.ID+"/subscribe-test", site)

	originalOverrides := footerVariantOverridesByKey
	footerVariantOverridesByKey = map[footerVariant]footerVariantOverrides{}
	testingT.Cleanup(func() {
		footerVariantOverridesByKey = originalOverrides
	})

	originalAuthTemplate := publicAuthScriptTemplate
	publicAuthScriptTemplate = texttemplate.Must(texttemplate.New("broken-subscribe-auth").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		publicAuthScriptTemplate = originalAuthTemplate
	})

	handlers := NewSiteSubscribeTestHandlers(database, zap.NewNop(), nil, nil, true, "http://loopaware.test", "secret", nil, AuthClientConfig{})
	handlers.RenderSubscribeTestPage(context)

	require.Equal(testingT, http.StatusOK, recorder.Code)
}

func TestRenderSubscribeTestPageReportsTemplateError(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	site := createTestSite(testingT, database)
	recorder, context := newTestPageContext(testingT, "/app/sites/"+site.ID+"/subscribe-test", site)

	handlers := NewSiteSubscribeTestHandlers(database, zap.NewNop(), nil, nil, true, "http://loopaware.test", "secret", nil, AuthClientConfig{})
	handlers.template = template.Must(template.New("broken-subscribe-template").Parse("{{.MissingField}}"))
	handlers.RenderSubscribeTestPage(context)

	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
}

func TestRenderTrafficTestPageHandlesFooterAndAuthErrors(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	site := createTestSite(testingT, database)
	recorder, context := newTestPageContext(testingT, "/app/sites/"+site.ID+"/traffic-test", site)

	originalOverrides := footerVariantOverridesByKey
	footerVariantOverridesByKey = map[footerVariant]footerVariantOverrides{}
	testingT.Cleanup(func() {
		footerVariantOverridesByKey = originalOverrides
	})

	originalAuthTemplate := publicAuthScriptTemplate
	publicAuthScriptTemplate = texttemplate.Must(texttemplate.New("broken-traffic-auth").Parse("{{.MissingField}}"))
	testingT.Cleanup(func() {
		publicAuthScriptTemplate = originalAuthTemplate
	})

	handlers := NewSiteTrafficTestHandlers(database, zap.NewNop(), AuthClientConfig{})
	handlers.RenderTrafficTestPage(context)

	require.Equal(testingT, http.StatusOK, recorder.Code)
}

func TestRenderTrafficTestPageReportsTemplateError(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	site := createTestSite(testingT, database)
	recorder, context := newTestPageContext(testingT, "/app/sites/"+site.ID+"/traffic-test", site)

	handlers := NewSiteTrafficTestHandlers(database, zap.NewNop(), AuthClientConfig{})
	handlers.template = template.Must(template.New("broken-traffic-template").Parse("{{.MissingField}}"))
	handlers.RenderTrafficTestPage(context)

	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
}
