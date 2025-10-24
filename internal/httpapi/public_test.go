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
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/httpapi"
	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

type apiHarness struct {
	router   *gin.Engine
	database *gorm.DB
	events   *httpapi.FeedbackEventBroadcaster
}

func buildAPIHarness(testingT *testing.T) apiHarness {
	testingT.Helper()

	gin.SetMode(gin.TestMode)
	logger, loggerErr := zap.NewDevelopment()
	require.NoError(testingT, loggerErr)

	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.Default())
	router.Use(httpapi.RequestLogger(logger))

	feedbackBroadcaster := httpapi.NewFeedbackEventBroadcaster()
	publicHandlers := httpapi.NewPublicHandlers(database, logger, feedbackBroadcaster)
	router.POST("/api/feedback", publicHandlers.CreateFeedback)
	router.GET("/widget.js", publicHandlers.WidgetJS)

	testingT.Cleanup(feedbackBroadcaster.Close)

	return apiHarness{
		router:   router,
		database: database,
		events:   feedbackBroadcaster,
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

func insertSite(testingT *testing.T, database *gorm.DB, name string, origin string, owner string) model.Site {
	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       name,
		AllowedOrigin:              origin,
		OwnerEmail:                 owner,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 16,
	}
	require.NoError(testingT, database.Create(&site).Error)
	return site
}

func TestFeedbackFlow(t *testing.T) {
	api := buildAPIHarness(t)
	site := insertSite(t, api.database, "Moving Maps", "http://example.com", "admin@example.com")

	widgetResp := performJSONRequest(t, api.router, http.MethodGet, "/widget.js?site_id="+site.ID, nil, nil)
	require.Equal(t, http.StatusOK, widgetResp.Code)
	require.Contains(t, widgetResp.Header().Get("Content-Type"), "application/javascript")
	widgetBody := widgetResp.Body.String()
	require.Contains(t, widgetBody, `panel.style.width = "320px"`)
	require.Contains(t, widgetBody, `site_id: "`+site.ID+`"`)
	require.Contains(t, widgetBody, `var widgetPlacementSideValue = "right"`)
	require.Contains(t, widgetBody, `var widgetPlacementBottomOffsetValue = 16`)
	require.Contains(t, widgetBody, `document.readyState === "loading"`)
	require.Contains(t, widgetBody, "scheduleWhenBodyReady")
	require.NotContains(t, widgetBody, "%!(")

	okFeedback := performJSONRequest(t, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "user@example.com",
		"message": "Hello from tests",
	}, map[string]string{"Origin": "http://example.com"})
	require.Equal(t, http.StatusOK, okFeedback.Code)

	badOrigin := performJSONRequest(t, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "user@example.com",
		"message": "attack",
	}, map[string]string{"Origin": "http://malicious.example"})
	require.Equal(t, http.StatusForbidden, badOrigin.Code)
}

func TestRateLimitingReturnsTooManyRequests(t *testing.T) {
	api := buildAPIHarness(t)
	site := insertSite(t, api.database, "Burst Site", "http://burst.example", "admin@example.com")

	headers := map[string]string{"Origin": "http://burst.example"}
	payload := map[string]any{"site_id": site.ID, "contact": "u@example.com", "message": "m"}

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

func TestWidgetJSHonorsCustomPlacement(t *testing.T) {
	api := buildAPIHarness(t)
	site := insertSite(t, api.database, "Custom Placement", "http://placement.example", "owner@example.com")
	require.NoError(t, api.database.Model(&model.Site{}).
		Where("id = ?", site.ID).
		Updates(map[string]any{
			"widget_bubble_side":             "left",
			"widget_bubble_bottom_offset_px": 48,
		}).Error)

	widgetResp := performJSONRequest(t, api.router, http.MethodGet, "/widget.js?site_id="+site.ID, nil, nil)
	require.Equal(t, http.StatusOK, widgetResp.Code)
	widgetBody := widgetResp.Body.String()
	require.Contains(t, widgetBody, `var widgetPlacementSideValue = "left"`)
	require.Contains(t, widgetBody, `var widgetPlacementBottomOffsetValue = 48`)
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
	site := insertSite(t, api.database, "Validation", "http://valid.example", "owner@example.com")

	respMissing := performJSONRequest(t, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "",
		"message": "",
	}, map[string]string{"Origin": "http://valid.example"})
	require.Equal(t, http.StatusBadRequest, respMissing.Code)

	bad := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/feedback", bytes.NewBufferString("{"))
	req.Header.Set("Origin", "http://valid.example")
	req.Header.Set("Content-Type", "application/json")
	api.router.ServeHTTP(bad, req)
	require.Equal(t, http.StatusBadRequest, bad.Code)
}
