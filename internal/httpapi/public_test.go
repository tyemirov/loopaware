package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/httpapi"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/testutil"
)

type apiHarness struct {
	router           *gin.Engine
	adminBearerToken string
}

func buildAPIHarness(testingT *testing.T) apiHarness {
	testingT.Helper()

	dsn := testutil.DSN()
	require.NotEmpty(testingT, dsn)

	gin.SetMode(gin.TestMode)
	logger, loggerErr := zap.NewDevelopment()
	require.NoError(testingT, loggerErr)

	db, openErr := storage.OpenPostgres(dsn)
	require.NoError(testingT, openErr)
	require.NoError(testingT, storage.AutoMigrate(db))

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.Default())
	router.Use(httpapi.RequestLogger(logger))

	publicHandlers := httpapi.NewPublicHandlers(db, logger)
	adminBearerToken := "test-admin-token"
	adminHandlers := httpapi.NewAdminHandlers(db, logger, adminBearerToken)
	adminWebHandlers := httpapi.NewAdminWebHandlers(logger)

	router.POST("/api/feedback", publicHandlers.CreateFeedback)
	router.GET("/widget.js", publicHandlers.WidgetJS)
	router.GET("/admin", adminWebHandlers.RenderAdminInterface)

	admin := router.Group("/api/admin")
	admin.Use(httpapi.AdminAuthMiddleware(adminBearerToken))
	admin.POST("/sites", adminHandlers.CreateSite)
	admin.GET("/sites/:id/messages", adminHandlers.ListMessagesBySite)

	return apiHarness{
		router:           router,
		adminBearerToken: adminBearerToken,
	}
}

func performJSONRequest(testingT *testing.T, router *gin.Engine, method string, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var requestBody io.Reader
	if body != nil {
		encoded, encodeErr := json.Marshal(body)
		require.NoError(testingT, encodeErr)
		requestBody = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, requestBody)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestAdminCreateSiteAndWidgetAndFeedbackFlow(t *testing.T) {
	api := buildAPIHarness(t)

	createSitePayload := map[string]string{
		"name":           "Moving Maps",
		"allowed_origin": "http://example.com",
	}
	adminHeaders := map[string]string{"Authorization": "Bearer " + api.adminBearerToken}

	createSiteResp := performJSONRequest(t, api.router, http.MethodPost, "/api/admin/sites", createSitePayload, adminHeaders)
	require.Equal(t, http.StatusOK, createSiteResp.Code)

	var createdSite map[string]string
	require.NoError(t, json.Unmarshal(createSiteResp.Body.Bytes(), &createdSite))
	siteID := createdSite["id"]
	require.NotEmpty(t, siteID)

	widgetResp := performJSONRequest(t, api.router, http.MethodGet, "/widget.js?site_id="+siteID, nil, nil)
	require.Equal(t, http.StatusOK, widgetResp.Code)
	require.Contains(t, widgetResp.Header().Get("Content-Type"), "application/javascript")

	okFeedback := performJSONRequest(t, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": siteID,
		"contact": "user@example.com",
		"message": "Hello from tests",
	}, map[string]string{"Origin": "http://example.com"})
	require.Equal(t, http.StatusOK, okFeedback.Code)

	badOrigin := performJSONRequest(t, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": siteID,
		"contact": "user@example.com",
		"message": "attack",
	}, map[string]string{"Origin": "http://malicious.example"})
	require.Equal(t, http.StatusForbidden, badOrigin.Code)

	listResp := performJSONRequest(t, api.router, http.MethodGet, "/api/admin/sites/"+siteID+"/messages", nil, adminHeaders)
	require.Equal(t, http.StatusOK, listResp.Code)

	var listing struct {
		SiteID   string `json:"site_id"`
		Messages []any  `json:"messages"`
	}
	require.NoError(t, json.Unmarshal(listResp.Body.Bytes(), &listing))
	require.Equal(t, siteID, listing.SiteID)
	require.GreaterOrEqual(t, len(listing.Messages), 1)
}

func TestRateLimitingReturnsTooManyRequests(t *testing.T) {
	api := buildAPIHarness(t)

	adminHeaders := map[string]string{"Authorization": "Bearer " + api.adminBearerToken}
	createResp := performJSONRequest(t, api.router, http.MethodPost, "/api/admin/sites", map[string]string{
		"name":           "Burst Site",
		"allowed_origin": "http://burst.example",
	}, adminHeaders)
	require.Equal(t, http.StatusOK, createResp.Code)

	var created map[string]string
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	siteID := created["id"]

	headers := map[string]string{"Origin": "http://burst.example"}
	payload := map[string]any{"site_id": siteID, "contact": "u@example.com", "message": "m"}

	tooMany := 0
	for attemptIndex := 0; attemptIndex < 12; attemptIndex++ {
		resp := performJSONRequest(t, api.router, http.MethodPost, "/api/feedback", payload, headers)
		if resp.Code == http.StatusTooManyRequests {
			tooMany++
			break
		}
	}
	require.GreaterOrEqual(t, tooMany, 1)
}

func TestAdminMiddlewareBlocksWithoutBearer(t *testing.T) {
	api := buildAPIHarness(t)

	resp := performJSONRequest(t, api.router, http.MethodPost, "/api/admin/sites", map[string]string{
		"name":           "NoAuth",
		"allowed_origin": "http://x.example",
	}, nil)
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestWidgetRequiresValidSiteId(t *testing.T) {
	api := buildAPIHarness(t)

	resp := performJSONRequest(t, api.router, http.MethodGet, "/widget.js?site_id=", nil, nil)
	require.Equal(t, http.StatusBadRequest, resp.Code)

	respUnknown := performJSONRequest(t, api.router, http.MethodGet, "/widget.js?site_id=does-not-exist", nil, nil)
	require.Equal(t, http.StatusNotFound, respUnknown.Code)
}

func TestCreateFeedbackValidatesPayload(t *testing.T) {
	api := buildAPIHarness(t)

	adminHeaders := map[string]string{"Authorization": "Bearer " + api.adminBearerToken}
	createSiteResp := performJSONRequest(t, api.router, http.MethodPost, "/api/admin/sites", map[string]string{
		"name":           "Validation",
		"allowed_origin": "http://valid.example",
	}, adminHeaders)
	require.Equal(t, http.StatusOK, createSiteResp.Code)

	var site map[string]string
	require.NoError(t, json.Unmarshal(createSiteResp.Body.Bytes(), &site))
	siteID := site["id"]

	respMissing := performJSONRequest(t, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": siteID,
		"contact": "",
		"message": "",
	}, map[string]string{"Origin": "http://valid.example"})
	require.Equal(t, http.StatusBadRequest, respMissing.Code)

	// malformed JSON
	bad := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/feedback", bytes.NewBufferString("{"))
	req.Header.Set("Origin", "http://valid.example")
	req.Header.Set("Content-Type", "application/json")
	api.router.ServeHTTP(bad, req)
	require.Equal(t, http.StatusBadRequest, bad.Code)
}
