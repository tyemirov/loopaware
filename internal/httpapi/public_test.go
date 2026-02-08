package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

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

const (
	testSubscribeSiteName              = "Subscribe Site"
	testSubscribeSiteOrigin            = "http://subscribe.example"
	testSubscribeOwnerAddress          = "owner@example.com"
	testSubscribeHeaderName            = "X-Site-Id"
	testSubscribeDemoMode              = "inline"
	testSubscribeDemoAccent            = "blue"
	testSubscribeDemoSuccessMsg        = "Thanks"
	testSubscribeDemoErrorMsg          = "Oops"
	testSubscribeMissingSiteID         = "missing"
	testSubscriptionUpdateCallbackName = "force_subscription_update_error"
	testSubscriptionUpdateErrorMessage = "subscription_update_failed"
	testSubscriptionUpdateTableName    = "subscribers"
	testSubscriptionUpdateEmail        = "subscriber@example.com"
	testSubscriptionUpdateName         = "Subscriber"
)

type apiHarness struct {
	router             *gin.Engine
	database           *gorm.DB
	events             *httpapi.FeedbackEventBroadcaster
	subscriptionEvents *httpapi.SubscriptionTestEventBroadcaster
}

func buildAPIHarness(testingT *testing.T, notifier httpapi.FeedbackNotifier, subscriptionNotifier httpapi.SubscriptionNotifier, emailSender httpapi.EmailSender) apiHarness {
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
	subscriptionEvents := httpapi.NewSubscriptionTestEventBroadcaster()
	publicHandlers := httpapi.NewPublicHandlers(database, logger, feedbackBroadcaster, subscriptionEvents, notifier, subscriptionNotifier, true, "http://loopaware.test", "unit-test-session-secret", emailSender, testLandingAuthConfig)
	router.POST("/api/feedback", publicHandlers.CreateFeedback)
	router.POST("/api/subscriptions", publicHandlers.CreateSubscription)
	router.POST("/api/subscriptions/confirm", publicHandlers.ConfirmSubscription)
	router.POST("/api/subscriptions/unsubscribe", publicHandlers.Unsubscribe)
	router.GET("/api/widget-config", publicHandlers.WidgetConfig)
	router.GET("/api/subscriptions/confirm-link", publicHandlers.ConfirmSubscriptionLinkJSON)
	router.GET("/api/subscriptions/unsubscribe-link", publicHandlers.UnsubscribeSubscriptionLinkJSON)
	publicJavaScriptHandlers := httpapi.NewPublicJavaScriptHandlers()
	router.GET("/widget.js", publicJavaScriptHandlers.WidgetJS)
	router.GET("/pixel.js", publicJavaScriptHandlers.PixelJS)
	router.GET("/subscribe.js", publicJavaScriptHandlers.SubscribeJS)
	subscribeDemoHandlers := httpapi.NewSubscribeDemoPageHandlers(logger)
	router.GET("/subscribe-demo", subscribeDemoHandlers.RenderSubscribeDemo)
	subscriptionLinkHandlers := httpapi.NewSubscriptionLinkPageHandlers(logger, testLandingAuthConfig, "")
	router.GET("/subscriptions/confirm", subscriptionLinkHandlers.RenderConfirmSubscriptionLink)
	router.GET("/subscriptions/unsubscribe", subscriptionLinkHandlers.RenderUnsubscribeSubscriptionLink)
	router.GET("/subscribe-target-test", func(context *gin.Context) {
		siteID := strings.TrimSpace(context.Query("site_id"))
		if siteID == "" {
			context.String(http.StatusBadRequest, "missing site_id")
			return
		}
		targetID := strings.TrimSpace(context.Query("target"))
		if targetID == "" {
			targetID = "subscribe-target"
		}
		useDataTarget := context.Query("data_target") == "true"
		scriptURL := "/subscribe.js?site_id=" + url.QueryEscape(siteID)
		if !useDataTarget {
			scriptURL += "&target=" + url.QueryEscape(targetID)
		}
		dataTargetAttribute := ""
		if useDataTarget {
			dataTargetAttribute = fmt.Sprintf(` data-target="%s"`, targetID)
		}
		page := fmt.Sprintf(`<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Subscribe Target Test</title></head><body><div id="%s"></div><script defer src="%s"%s></script></body></html>`, targetID, scriptURL, dataTargetAttribute)
		context.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
	})
	router.GET("/api/visits", publicHandlers.CollectVisit)

	testingT.Cleanup(feedbackBroadcaster.Close)
	testingT.Cleanup(subscriptionEvents.Close)

	return apiHarness{
		router:             router,
		database:           database,
		events:             feedbackBroadcaster,
		subscriptionEvents: subscriptionEvents,
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

func TestFeedbackFlow(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Moving Maps", "http://example.com", "admin@example.com")

	widgetResp := performJSONRequest(testingT, api.router, http.MethodGet, "/widget.js", nil, nil)
	require.Equal(testingT, http.StatusOK, widgetResp.Code)
	require.Contains(testingT, widgetResp.Header().Get("Content-Type"), "application/javascript")
	widgetBody := widgetResp.Body.String()
	require.Contains(testingT, widgetBody, `panel.style.width = "320px"`)
	require.Contains(testingT, widgetBody, "/api/widget-config?site_id=")
	require.Contains(testingT, widgetBody, "/api/feedback")
	require.Contains(testingT, widgetBody, `document.readyState === "loading"`)
	require.Contains(testingT, widgetBody, "scheduleWhenBodyReady")
	require.NotContains(testingT, widgetBody, "%!(")

	widgetConfigResp := performJSONRequest(testingT, api.router, http.MethodGet, "/api/widget-config?site_id="+site.ID, nil, map[string]string{
		"Origin": site.AllowedOrigin,
	})
	require.Equal(testingT, http.StatusOK, widgetConfigResp.Code)
	var widgetConfigPayload struct {
		SiteID                   string `json:"site_id"`
		WidgetBubbleSide         string `json:"widget_bubble_side"`
		WidgetBubbleBottomOffset int    `json:"widget_bubble_bottom_offset"`
	}
	require.NoError(testingT, json.Unmarshal(widgetConfigResp.Body.Bytes(), &widgetConfigPayload))
	require.Equal(testingT, site.ID, widgetConfigPayload.SiteID)
	require.Equal(testingT, "right", widgetConfigPayload.WidgetBubbleSide)
	require.Equal(testingT, 16, widgetConfigPayload.WidgetBubbleBottomOffset)

	okFeedback := performJSONRequest(testingT, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "user@example.com",
		"message": "Hello from tests",
	}, map[string]string{"Origin": "http://example.com"})
	require.Equal(testingT, http.StatusOK, okFeedback.Code)

	badOrigin := performJSONRequest(testingT, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "user@example.com",
		"message": "attack",
	}, map[string]string{"Origin": "http://malicious.example"})
	require.Equal(testingT, http.StatusForbidden, badOrigin.Code)
}

func TestRateLimitingReturnsTooManyRequests(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Burst Site", "http://burst.example", "admin@example.com")

	headers := map[string]string{"Origin": "http://burst.example"}
	payload := map[string]any{"site_id": site.ID, "contact": "u@example.com", "message": "m"}

	tooMany := 0
	for attemptIndex := 0; attemptIndex < 12; attemptIndex++ {
		resp := performJSONRequest(testingT, api.router, http.MethodPost, "/api/feedback", payload, headers)
		if resp.Code == http.StatusTooManyRequests {
			tooMany++
			break
		}
	}
	require.GreaterOrEqual(testingT, tooMany, 1)
}

func TestConfirmSubscriptionReportsUpdateError(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, testSubscribeSiteName, testSubscribeSiteOrigin, testSubscribeOwnerAddress)

	subscriber, subscriberErr := model.NewSubscriber(model.SubscriberInput{
		SiteID:    site.ID,
		Email:     testSubscriptionUpdateEmail,
		Name:      testSubscriptionUpdateName,
		Status:    model.SubscriberStatusPending,
		ConsentAt: time.Now().UTC(),
	})
	require.NoError(testingT, subscriberErr)
	require.NoError(testingT, api.database.Create(&subscriber).Error)

	callbackName := testSubscriptionUpdateCallbackName
	api.database.Callback().Update().Before("gorm:update").Register(callbackName, func(database *gorm.DB) {
		if database.Statement != nil && database.Statement.Table == testSubscriptionUpdateTableName {
			database.AddError(errors.New(testSubscriptionUpdateErrorMessage))
		}
	})
	testingT.Cleanup(func() {
		api.database.Callback().Update().Remove(callbackName)
	})

	response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   subscriber.Email,
	}, map[string]string{"Origin": site.AllowedOrigin})
	require.Equal(testingT, http.StatusInternalServerError, response.Code)
}

func TestWidgetConfigHonorsCustomPlacement(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Custom Placement", "http://placement.example", "owner@example.com")
	require.NoError(testingT, api.database.Model(&model.Site{}).
		Where("id = ?", site.ID).
		Updates(map[string]any{
			"widget_bubble_side":             "left",
			"widget_bubble_bottom_offset_px": 48,
		}).Error)

	widgetConfigResp := performJSONRequest(testingT, api.router, http.MethodGet, "/api/widget-config?site_id="+site.ID, nil, map[string]string{
		"Origin": site.AllowedOrigin,
	})
	require.Equal(testingT, http.StatusOK, widgetConfigResp.Code)
	var widgetConfigPayload struct {
		SiteID                   string `json:"site_id"`
		WidgetBubbleSide         string `json:"widget_bubble_side"`
		WidgetBubbleBottomOffset int    `json:"widget_bubble_bottom_offset"`
	}
	require.NoError(testingT, json.Unmarshal(widgetConfigResp.Body.Bytes(), &widgetConfigPayload))
	require.Equal(testingT, site.ID, widgetConfigPayload.SiteID)
	require.Equal(testingT, "left", widgetConfigPayload.WidgetBubbleSide)
	require.Equal(testingT, 48, widgetConfigPayload.WidgetBubbleBottomOffset)
}

func TestWidgetConfigRequiresValidSiteId(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)

	resp := performJSONRequest(testingT, api.router, http.MethodGet, "/api/widget-config", nil, nil)
	require.Equal(testingT, http.StatusBadRequest, resp.Code)

	respUnknown := performJSONRequest(testingT, api.router, http.MethodGet, "/api/widget-config?site_id=does-not-exist", nil, nil)
	require.Equal(testingT, http.StatusNotFound, respUnknown.Code)

	site := insertSite(testingT, api.database, "Widget Config Origin", "http://widget-config.example", "owner@example.com")
	respForbidden := performJSONRequest(testingT, api.router, http.MethodGet, "/api/widget-config?site_id="+site.ID, nil, map[string]string{
		"Origin": "http://evil.example",
	})
	require.Equal(testingT, http.StatusForbidden, respForbidden.Code)
}

func TestCreateFeedbackValidatesPayload(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Validation", "http://valid.example", "owner@example.com")

	respMissing := performJSONRequest(testingT, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "",
		"message": "",
	}, map[string]string{"Origin": "http://valid.example"})
	require.Equal(testingT, http.StatusBadRequest, respMissing.Code)

	bad := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/feedback", bytes.NewBufferString("{"))
	req.Header.Set("Origin", "http://valid.example")
	req.Header.Set("Content-Type", "application/json")
	api.router.ServeHTTP(bad, req)
	require.Equal(testingT, http.StatusBadRequest, bad.Code)
}

func TestCreateFeedbackAcceptsWidgetAllowedOrigins(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Widget Origins", "http://origin.example", "owner@example.com")
	site.WidgetAllowedOrigins = "http://widget.example"
	require.NoError(testingT, api.database.Save(&site).Error)

	ok := performJSONRequest(testingT, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "person@example.com",
		"message": "Hello",
	}, map[string]string{"Origin": "http://widget.example"})
	require.Equal(testingT, http.StatusOK, ok.Code)

	badOrigin := performJSONRequest(testingT, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "person@example.com",
		"message": "Hello",
	}, map[string]string{"Origin": "http://evil.example"})
	require.Equal(testingT, http.StatusForbidden, badOrigin.Code)
}

func TestCreateFeedbackRateLimited(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Rate Limit", "http://rate-limit.example", "owner@example.com")

	headers := map[string]string{
		"Origin":          site.AllowedOrigin,
		"X-Forwarded-For": "127.0.0.1",
	}
	payload := map[string]any{
		"site_id": site.ID,
		"contact": "person@example.com",
		"message": "Hello",
	}

	rateLimited := false
	for attemptIndex := 0; attemptIndex < 12; attemptIndex++ {
		response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/feedback", payload, headers)
		if response.Code == http.StatusTooManyRequests {
			rateLimited = true
			break
		}
	}
	require.True(testingT, rateLimited)
}

func TestCreateFeedbackReportsSaveError(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Save Error", "http://save-error.example", "owner@example.com")

	registerErr := api.database.Callback().Create().Before("gorm:create").Register("force_feedback_error", func(callbackDatabase *gorm.DB) {
		callbackDatabase.AddError(errors.New("forced feedback error"))
	})
	require.NoError(testingT, registerErr)
	testingT.Cleanup(func() {
		_ = api.database.Callback().Create().Remove("force_feedback_error")
	})

	response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "person@example.com",
		"message": "Hello",
	}, map[string]string{"Origin": site.AllowedOrigin})
	require.Equal(testingT, http.StatusInternalServerError, response.Code)
}

func TestCreateSubscriptionStoresSubscriber(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Newsletter", "http://newsletter.example", "owner@example.com")

	resp := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "Subscriber@example.com",
		"name":    "Subscriber",
	}, map[string]string{"Origin": "http://newsletter.example"})
	require.Equal(testingT, http.StatusOK, resp.Code)

	var stored model.Subscriber
	require.NoError(testingT, api.database.First(&stored).Error)
	require.Equal(testingT, site.ID, stored.SiteID)
	require.Equal(testingT, "subscriber@example.com", stored.Email)
	require.Equal(testingT, model.SubscriberStatusPending, stored.Status)
}

func TestCreateSubscriptionValidatesInput(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Validation Subscription", "http://sub.example", "owner@example.com")

	respMissing := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": "",
		"email":   "",
	}, map[string]string{"Origin": "http://sub.example"})
	require.Equal(testingT, http.StatusBadRequest, respMissing.Code)

	respInvalidEmail := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "not-an-email",
	}, map[string]string{"Origin": "http://sub.example"})
	require.Equal(testingT, http.StatusBadRequest, respInvalidEmail.Code)
}

func TestCreateSubscriptionBlocksOriginAndDuplicates(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Origins", "http://origin.example", "owner@example.com")

	badOrigin := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "user@example.com",
	}, map[string]string{"Origin": "http://evil.example"})
	require.Equal(testingT, http.StatusForbidden, badOrigin.Code)

	ok := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "user@example.com",
	}, map[string]string{"Origin": "http://origin.example"})
	require.Equal(testingT, http.StatusOK, ok.Code)

	duplicate := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "user@example.com",
	}, map[string]string{"Origin": "http://origin.example"})
	require.Equal(testingT, http.StatusConflict, duplicate.Code)
}

func TestCreateSubscriptionSupportsMultipleAllowedOrigins(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Multi Origins", "https://mprlab.com http://localhost:8080", "owner@example.com")

	testCases := []struct {
		name           string
		originHeader   string
		expectedStatus int
	}{
		{
			name:           "primary origin accepted",
			originHeader:   "https://mprlab.com",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "secondary origin accepted",
			originHeader:   "http://localhost:8080",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unknown origin rejected",
			originHeader:   "https://evil.example",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			subscriberEmailValue := storage.NewID() + "@example.com"
			response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
				"site_id": site.ID,
				"email":   subscriberEmailValue,
			}, map[string]string{"Origin": testCase.originHeader})
			require.Equal(testingT, testCase.expectedStatus, response.Code)
		})
	}
}

func TestCreateSubscriptionAcceptsSubscribeAllowedOrigins(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Subscribe Origins", "http://origin.example", "owner@example.com")
	site.SubscribeAllowedOrigins = "http://newsletter.example"
	require.NoError(testingT, api.database.Save(&site).Error)

	ok := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   storage.NewID() + "@example.com",
	}, map[string]string{"Origin": "http://newsletter.example"})
	require.Equal(testingT, http.StatusOK, ok.Code)

	badOrigin := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   storage.NewID() + "@example.com",
	}, map[string]string{"Origin": "http://evil.example"})
	require.Equal(testingT, http.StatusForbidden, badOrigin.Code)
}

func TestConfirmAndUnsubscribeSubscription(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Confirmations", "http://confirm.example", "owner@example.com")

	createResp := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "confirm@example.com",
	}, map[string]string{"Origin": "http://confirm.example"})
	require.Equal(testingT, http.StatusOK, createResp.Code)

	confirm := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   "confirm@example.com",
	}, map[string]string{"Origin": "http://confirm.example"})
	require.Equal(testingT, http.StatusOK, confirm.Code)

	var confirmed model.Subscriber
	require.NoError(testingT, api.database.First(&confirmed).Error)
	require.Equal(testingT, model.SubscriberStatusConfirmed, confirmed.Status)
	require.False(testingT, confirmed.ConfirmedAt.IsZero())

	unsubscribe := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions/unsubscribe", map[string]any{
		"site_id": site.ID,
		"email":   "confirm@example.com",
	}, map[string]string{"Origin": "http://confirm.example"})
	require.Equal(testingT, http.StatusOK, unsubscribe.Code)

	var unsubscribed model.Subscriber
	require.NoError(testingT, api.database.First(&unsubscribed).Error)
	require.Equal(testingT, model.SubscriberStatusUnsubscribed, unsubscribed.Status)
	require.False(testingT, unsubscribed.UnsubscribedAt.IsZero())

	reconfirm := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   "confirm@example.com",
	}, map[string]string{"Origin": "http://confirm.example"})
	require.Equal(testingT, http.StatusConflict, reconfirm.Code)

	missing := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   "absent@example.com",
	}, map[string]string{"Origin": "http://confirm.example"})
	require.Equal(testingT, http.StatusNotFound, missing.Code)
}

func TestSubscriptionConfirmationEmailConfirmsViaLink(testingT *testing.T) {
	emailSender := &recordingEmailSender{testingT: testingT}
	subscriptionNotifier := &recordingSubscriptionNotifier{testingT: testingT}
	api := buildAPIHarness(testingT, nil, subscriptionNotifier, emailSender)
	site := insertSite(testingT, api.database, "Confirmation Email", "http://confirm.example", "owner@example.com")

	createResp := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "confirm@example.com",
	}, map[string]string{"Origin": "http://confirm.example"})
	require.Equal(testingT, http.StatusOK, createResp.Code)
	require.Equal(testingT, 1, emailSender.CallCount())
	require.Equal(testingT, 0, subscriptionNotifier.CallCount())

	lastEmail := emailSender.LastCall()
	require.Equal(testingT, "confirm@example.com", lastEmail.Recipient)
	require.Contains(testingT, lastEmail.Subject, "Confirm your subscription")

	var confirmationLink string
	for _, line := range strings.Split(lastEmail.Message, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "/subscriptions/confirm?token=") {
			confirmationLink = line
			break
		}
	}
	require.NotEmpty(testingT, confirmationLink)

	parsedURL, parseErr := url.Parse(confirmationLink)
	require.NoError(testingT, parseErr)

	tokenValue := parsedURL.Query().Get("token")
	require.NotEmpty(testingT, tokenValue)

	type subscriptionLinkPayload struct {
		Heading        string `json:"heading"`
		Message        string `json:"message"`
		OpenURL        string `json:"open_url"`
		OpenLabel      string `json:"open_label"`
		UnsubscribeURL string `json:"unsubscribe_url"`
	}

	confirmResponse := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/confirm-link?token="+url.QueryEscape(tokenValue), nil, nil)
	require.Equal(testingT, http.StatusOK, confirmResponse.Code)
	var confirmPayload subscriptionLinkPayload
	require.NoError(testingT, json.Unmarshal(confirmResponse.Body.Bytes(), &confirmPayload))
	require.Equal(testingT, "Subscription confirmed", confirmPayload.Heading)
	require.Equal(testingT, site.AllowedOrigin, confirmPayload.OpenURL)
	require.Contains(testingT, confirmPayload.UnsubscribeURL, "/subscriptions/unsubscribe?token=")

	unsubscribeResponse := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/unsubscribe-link?token="+url.QueryEscape(tokenValue), nil, nil)
	require.Equal(testingT, http.StatusOK, unsubscribeResponse.Code)
	var unsubscribePayload subscriptionLinkPayload
	require.NoError(testingT, json.Unmarshal(unsubscribeResponse.Body.Bytes(), &unsubscribePayload))
	require.Equal(testingT, "Unsubscribed", unsubscribePayload.Heading)

	var stored model.Subscriber
	require.NoError(testingT, api.database.First(&stored, "site_id = ? AND email = ?", site.ID, "confirm@example.com").Error)
	require.Equal(testingT, model.SubscriberStatusUnsubscribed, stored.Status)
	require.False(testingT, stored.ConfirmedAt.IsZero())
	require.False(testingT, stored.UnsubscribedAt.IsZero())

	require.Equal(testingT, 1, subscriptionNotifier.CallCount())
	notification := subscriptionNotifier.LastCall()
	require.Equal(testingT, site.ID, notification.Site.ID)
	require.Equal(testingT, "confirm@example.com", notification.Subscriber.Email)
}

func TestCreateSubscriptionDoesNotNotifyUntilConfirmed(testingT *testing.T) {
	subscriptionNotifier := &recordingSubscriptionNotifier{testingT: testingT}
	api := buildAPIHarness(testingT, nil, subscriptionNotifier, nil)
	site := insertSite(testingT, api.database, "Notify", "http://notify.example", "owner@example.com")

	resp := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "notify@example.com",
	}, map[string]string{"Origin": "http://notify.example"})
	require.Equal(testingT, http.StatusOK, resp.Code)
	require.Equal(testingT, 0, subscriptionNotifier.CallCount())

	confirm := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   "notify@example.com",
	}, map[string]string{"Origin": "http://notify.example"})
	require.Equal(testingT, http.StatusOK, confirm.Code)
	require.Equal(testingT, 1, subscriptionNotifier.CallCount())
}

func TestSubscriptionNotificationFailureDoesNotBlock(testingT *testing.T) {
	subscriptionNotifier := &recordingSubscriptionNotifier{testingT: testingT, callErr: errors.New("pinguin down")}

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
	router.Use(httpapi.RequestLogger(logger))

	feedbackBroadcaster := httpapi.NewFeedbackEventBroadcaster()
	testingT.Cleanup(feedbackBroadcaster.Close)
	publicHandlers := httpapi.NewPublicHandlers(database, logger, feedbackBroadcaster, nil, nil, subscriptionNotifier, true, "http://loopaware.test", "unit-test-session-secret", nil, testLandingAuthConfig)

	router.POST("/api/subscriptions", publicHandlers.CreateSubscription)
	router.POST("/api/subscriptions/confirm", publicHandlers.ConfirmSubscription)

	site := insertSite(testingT, database, "Notify Fail", "http://notifyfail.example", "owner@example.com")
	resp := performJSONRequest(testingT, router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "notify@example.com",
	}, map[string]string{"Origin": "http://notifyfail.example"})
	require.Equal(testingT, http.StatusOK, resp.Code)
	require.Equal(testingT, 0, subscriptionNotifier.CallCount())

	confirm := performJSONRequest(testingT, router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   "notify@example.com",
	}, map[string]string{"Origin": "http://notifyfail.example"})
	require.Equal(testingT, http.StatusOK, confirm.Code)
	require.Equal(testingT, 1, subscriptionNotifier.CallCount())
}

func TestSubscriptionNotificationsCanBeDisabled(testingT *testing.T) {
	subscriptionNotifier := &recordingSubscriptionNotifier{testingT: testingT}

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
	router.Use(httpapi.RequestLogger(logger))

	feedbackBroadcaster := httpapi.NewFeedbackEventBroadcaster()
	testingT.Cleanup(feedbackBroadcaster.Close)
	publicHandlers := httpapi.NewPublicHandlers(database, logger, feedbackBroadcaster, nil, nil, subscriptionNotifier, false, "http://loopaware.test", "unit-test-session-secret", nil, testLandingAuthConfig)
	router.POST("/api/subscriptions", publicHandlers.CreateSubscription)
	router.POST("/api/subscriptions/confirm", publicHandlers.ConfirmSubscription)

	site := insertSite(testingT, database, "Notify Off", "http://notifyoff.example", "owner@example.com")
	resp := performJSONRequest(testingT, router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "notify@example.com",
	}, map[string]string{"Origin": "http://notifyoff.example"})
	require.Equal(testingT, http.StatusOK, resp.Code)
	require.Equal(testingT, 0, subscriptionNotifier.CallCount())

	confirm := performJSONRequest(testingT, router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   "notify@example.com",
	}, map[string]string{"Origin": "http://notifyoff.example"})
	require.Equal(testingT, http.StatusOK, confirm.Code)
	require.Equal(testingT, 0, subscriptionNotifier.CallCount())
}

func TestCollectVisitStoresRecord(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Visits", "http://visits.example", "owner@example.com")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/visits?site_id="+site.ID+"&url=http://visits.example/page", nil)
	request.Header.Set("Origin", "http://visits.example")

	api.router.ServeHTTP(recorder, request)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Contains(testingT, recorder.Header().Get("Content-Type"), "image/gif")

	var stored model.SiteVisit
	require.NoError(testingT, api.database.First(&stored).Error)
	require.Equal(testingT, site.ID, stored.SiteID)
	require.Equal(testingT, "http://visits.example/page", stored.URL)
	require.Equal(testingT, "/page", stored.Path)
}

func TestCollectVisitValidatesInput(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Visits Invalid", "http://visits.example", "owner@example.com")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/visits?site_id="+site.ID+"&url=//bad-url", nil)
	request.Header.Set("Origin", "http://visits.example")
	api.router.ServeHTTP(recorder, request)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/visits?site_id="+site.ID+"&url=http://visits.example/page", nil)
	request.Header.Set("Origin", "http://evil.example")
	api.router.ServeHTTP(recorder, request)
	require.Equal(testingT, http.StatusOK, recorder.Code)
}

func TestCollectVisitRequiresMatchingURLOrigin(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Visits Mismatch", "http://visits.example", "owner@example.com")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/visits?site_id="+site.ID+"&url=http://other.example/page", nil)
	request.Header.Set("Referer", "http://dashboard.loopaware.test/app/sites/"+site.ID+"/traffic-test")

	api.router.ServeHTTP(recorder, request)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)
}

func TestCollectVisitAcceptsTrafficAllowedOrigins(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Traffic Origins", "http://visits.example", "owner@example.com")
	site.TrafficAllowedOrigins = "http://pixel.example"
	require.NoError(testingT, api.database.Save(&site).Error)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/visits?site_id="+site.ID+"&url=http://pixel.example/page", nil)
	request.Header.Set("Origin", "http://pixel.example")
	api.router.ServeHTTP(recorder, request)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/visits?site_id="+site.ID+"&url=http://evil.example/page", nil)
	request.Header.Set("Origin", "http://evil.example")
	api.router.ServeHTTP(recorder, request)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)
}

type feedbackNotificationCall struct {
	Site     model.Site
	Feedback model.Feedback
	Context  context.Context
}

type recordingFeedbackNotifier struct {
	testingT  *testing.T
	mu        sync.Mutex
	calls     []feedbackNotificationCall
	delivery  string
	callError error
}

type subscriptionNotificationCall struct {
	Site       model.Site
	Subscriber model.Subscriber
	Context    context.Context
}

type recordingSubscriptionNotifier struct {
	testingT *testing.T
	mu       sync.Mutex
	calls    []subscriptionNotificationCall
	callErr  error
}

type emailSendCall struct {
	Recipient string
	Subject   string
	Message   string
	Context   context.Context
}

type recordingEmailSender struct {
	testingT *testing.T
	mu       sync.Mutex
	calls    []emailSendCall
	callErr  error
}

func (sender *recordingEmailSender) SendEmail(ctx context.Context, recipient string, subject string, message string) error {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	sender.calls = append(sender.calls, emailSendCall{
		Recipient: recipient,
		Subject:   subject,
		Message:   message,
		Context:   ctx,
	})
	return sender.callErr
}

func (sender *recordingEmailSender) CallCount() int {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	return len(sender.calls)
}

func (sender *recordingEmailSender) LastCall() emailSendCall {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	if len(sender.calls) == 0 {
		sender.testingT.Fatalf("expected at least one email sender call")
	}
	return sender.calls[len(sender.calls)-1]
}

func (notifier *recordingSubscriptionNotifier) NotifySubscription(ctx context.Context, site model.Site, subscriber model.Subscriber) error {
	notifier.mu.Lock()
	defer notifier.mu.Unlock()
	notifier.calls = append(notifier.calls, subscriptionNotificationCall{
		Site:       site,
		Subscriber: subscriber,
		Context:    ctx,
	})
	return notifier.callErr
}

func (notifier *recordingSubscriptionNotifier) CallCount() int {
	notifier.mu.Lock()
	defer notifier.mu.Unlock()
	return len(notifier.calls)
}

func (notifier *recordingSubscriptionNotifier) LastCall() subscriptionNotificationCall {
	notifier.mu.Lock()
	defer notifier.mu.Unlock()
	if len(notifier.calls) == 0 {
		notifier.testingT.Fatalf("expected at least one subscription notifier call")
	}
	return notifier.calls[len(notifier.calls)-1]
}

func (notifier *recordingFeedbackNotifier) NotifyFeedback(ctx context.Context, site model.Site, feedback model.Feedback) (string, error) {
	notifier.mu.Lock()
	defer notifier.mu.Unlock()
	notifier.calls = append(notifier.calls, feedbackNotificationCall{
		Site:     site,
		Feedback: feedback,
		Context:  ctx,
	})
	return notifier.delivery, notifier.callError
}

func (notifier *recordingFeedbackNotifier) CallCount() int {
	notifier.mu.Lock()
	defer notifier.mu.Unlock()
	return len(notifier.calls)
}

func (notifier *recordingFeedbackNotifier) LastCall() feedbackNotificationCall {
	notifier.mu.Lock()
	defer notifier.mu.Unlock()
	if len(notifier.calls) == 0 {
		notifier.testingT.Fatalf("expected at least one notifier call")
	}
	return notifier.calls[len(notifier.calls)-1]
}

func TestCreateFeedbackDispatchesNotificationToOwner(testingT *testing.T) {
	notifier := &recordingFeedbackNotifier{
		testingT: testingT,
		delivery: model.FeedbackDeliveryMailed,
	}
	api := buildAPIHarness(testingT, notifier, nil, nil)
	site := insertSite(testingT, api.database, "Dispatcher", "http://dispatch.example", "owner@example.com")
	require.NoError(testingT, api.database.Model(&model.Site{}).
		Where("id = ?", site.ID).
		Update("creator_email", "registrar@example.com").Error)

	resp := performJSONRequest(testingT, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "submitter@example.com",
		"message": "Dispatch notification",
	}, map[string]string{"Origin": "http://dispatch.example"})
	require.Equal(testingT, http.StatusOK, resp.Code)
	require.Equal(testingT, 1, notifier.CallCount())

	var stored model.Feedback
	require.NoError(testingT, api.database.First(&stored).Error)
	require.Equal(testingT, model.FeedbackDeliveryMailed, stored.Delivery)

	lastCall := notifier.LastCall()
	require.Equal(testingT, site.ID, lastCall.Site.ID)
	require.Equal(testingT, "owner@example.com", lastCall.Site.OwnerEmail)
	require.Equal(testingT, stored.ID, lastCall.Feedback.ID)
}

func TestCreateFeedbackRecordsNoDeliveryOnNotifierFailure(testingT *testing.T) {
	notifier := &recordingFeedbackNotifier{
		testingT:  testingT,
		delivery:  model.FeedbackDeliveryMailed,
		callError: errors.New("send failed"),
	}
	api := buildAPIHarness(testingT, notifier, nil, nil)
	site := insertSite(testingT, api.database, "Failure Delivery", "http://failure.example", "owner@example.com")

	resp := performJSONRequest(testingT, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "submitter@example.com",
		"message": "Expect failure",
	}, map[string]string{"Origin": "http://failure.example"})
	require.Equal(testingT, http.StatusOK, resp.Code)
	require.Equal(testingT, 1, notifier.CallCount())

	var stored model.Feedback
	require.NoError(testingT, api.database.First(&stored).Error)
	require.Equal(testingT, model.FeedbackDeliveryNone, stored.Delivery)
}

func TestCreateFeedbackPersistsFailureDeliveryWhenNotifierReturnsStatusAndError(testingT *testing.T) {
	notifier := &recordingFeedbackNotifier{
		testingT:  testingT,
		delivery:  model.FeedbackDeliveryTexted,
		callError: errors.New("notifier failed"),
	}
	api := buildAPIHarness(testingT, notifier, nil, nil)
	site := insertSite(testingT, api.database, "Failure Delivery Status", "http://failure-status.example", "owner@example.com")

	resp := performJSONRequest(testingT, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "submitter@example.com",
		"message": "Expect failure status",
	}, map[string]string{"Origin": "http://failure-status.example"})
	require.Equal(testingT, http.StatusOK, resp.Code)
	require.Equal(testingT, 1, notifier.CallCount())

	var stored model.Feedback
	require.NoError(testingT, api.database.First(&stored).Error)
	require.Equal(testingT, model.FeedbackDeliveryNone, stored.Delivery)
}

func TestSubscribeJSReturnsStaticScript(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)

	response := performJSONRequest(testingT, api.router, http.MethodGet, "/subscribe.js", nil, nil)
	require.Equal(testingT, http.StatusOK, response.Code)
	require.Contains(testingT, response.Header().Get("Content-Type"), "application/javascript")
	body := response.Body.String()
	require.Contains(testingT, body, "/api/subscriptions")
	require.Contains(testingT, body, `params.get("site_id")`)
	require.NotContains(testingT, body, "%!(")
}

func TestSubscribeDemoRendersScriptURL(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, testSubscribeSiteName, testSubscribeSiteOrigin, testSubscribeOwnerAddress)

	missingResp := performJSONRequest(testingT, api.router, http.MethodGet, "/subscribe-demo", nil, nil)
	require.Equal(testingT, http.StatusBadRequest, missingResp.Code)

	unknownResp := performJSONRequest(testingT, api.router, http.MethodGet, "/subscribe-demo?site_id="+testSubscribeMissingSiteID, nil, nil)
	require.Equal(testingT, http.StatusOK, unknownResp.Code)
	require.Contains(testingT, unknownResp.Body.String(), "/subscribe.js?site_id="+testSubscribeMissingSiteID)

	demoPath := "/subscribe-demo?site_id=" + url.QueryEscape(site.ID) +
		"&mode=" + url.QueryEscape(testSubscribeDemoMode) +
		"&accent=" + url.QueryEscape(testSubscribeDemoAccent) +
		"&success=" + url.QueryEscape(testSubscribeDemoSuccessMsg) +
		"&error=" + url.QueryEscape(testSubscribeDemoErrorMsg)
	demoResp := performJSONRequest(testingT, api.router, http.MethodGet, demoPath, nil, nil)
	require.Equal(testingT, http.StatusOK, demoResp.Code)
	require.Contains(testingT, demoResp.Body.String(), "/subscribe.js?site_id="+site.ID)
	require.Contains(testingT, demoResp.Body.String(), "mode="+url.QueryEscape(testSubscribeDemoMode))
	require.Contains(testingT, demoResp.Body.String(), "accent="+url.QueryEscape(testSubscribeDemoAccent))
}
