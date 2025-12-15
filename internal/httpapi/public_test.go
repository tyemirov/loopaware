package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
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
	publicHandlers := httpapi.NewPublicHandlers(database, logger, feedbackBroadcaster, subscriptionEvents, notifier, subscriptionNotifier, true, "http://loopaware.test", "unit-test-session-secret", emailSender)
	router.POST("/api/feedback", publicHandlers.CreateFeedback)
	router.POST("/api/subscriptions", publicHandlers.CreateSubscription)
	router.POST("/api/subscriptions/confirm", publicHandlers.ConfirmSubscription)
	router.POST("/api/subscriptions/unsubscribe", publicHandlers.Unsubscribe)
	router.GET("/subscriptions/confirm", publicHandlers.ConfirmSubscriptionLink)
	router.GET("/widget.js", publicHandlers.WidgetJS)
	router.GET("/subscribe.js", publicHandlers.SubscribeJS)
	router.GET("/subscribe-demo", publicHandlers.SubscribeDemo)
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

func TestFeedbackFlow(t *testing.T) {
	api := buildAPIHarness(t, nil, nil, nil)
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
	api := buildAPIHarness(t, nil, nil, nil)
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
	api := buildAPIHarness(t, nil, nil, nil)
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
	api := buildAPIHarness(t, nil, nil, nil)

	resp := performJSONRequest(t, api.router, http.MethodGet, "/widget.js?site_id=", nil, nil)
	require.Equal(t, http.StatusBadRequest, resp.Code)

	respUnknown := performJSONRequest(t, api.router, http.MethodGet, "/widget.js?site_id=does-not-exist", nil, nil)
	require.Equal(t, http.StatusNotFound, respUnknown.Code)
}

func TestCreateFeedbackValidatesPayload(t *testing.T) {
	api := buildAPIHarness(t, nil, nil, nil)
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

func TestCreateSubscriptionStoresSubscriber(t *testing.T) {
	api := buildAPIHarness(t, nil, nil, nil)
	site := insertSite(t, api.database, "Newsletter", "http://newsletter.example", "owner@example.com")

	resp := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "Subscriber@example.com",
		"name":    "Subscriber",
	}, map[string]string{"Origin": "http://newsletter.example"})
	require.Equal(t, http.StatusOK, resp.Code)

	var stored model.Subscriber
	require.NoError(t, api.database.First(&stored).Error)
	require.Equal(t, site.ID, stored.SiteID)
	require.Equal(t, "subscriber@example.com", stored.Email)
	require.Equal(t, model.SubscriberStatusPending, stored.Status)
}

func TestCreateSubscriptionValidatesInput(t *testing.T) {
	api := buildAPIHarness(t, nil, nil, nil)
	site := insertSite(t, api.database, "Validation Subscription", "http://sub.example", "owner@example.com")

	respMissing := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": "",
		"email":   "",
	}, map[string]string{"Origin": "http://sub.example"})
	require.Equal(t, http.StatusBadRequest, respMissing.Code)

	respInvalidEmail := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "not-an-email",
	}, map[string]string{"Origin": "http://sub.example"})
	require.Equal(t, http.StatusBadRequest, respInvalidEmail.Code)
}

func TestCreateSubscriptionBlocksOriginAndDuplicates(t *testing.T) {
	api := buildAPIHarness(t, nil, nil, nil)
	site := insertSite(t, api.database, "Origins", "http://origin.example", "owner@example.com")

	badOrigin := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "user@example.com",
	}, map[string]string{"Origin": "http://evil.example"})
	require.Equal(t, http.StatusForbidden, badOrigin.Code)

	ok := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "user@example.com",
	}, map[string]string{"Origin": "http://origin.example"})
	require.Equal(t, http.StatusOK, ok.Code)

	duplicate := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "user@example.com",
	}, map[string]string{"Origin": "http://origin.example"})
	require.Equal(t, http.StatusConflict, duplicate.Code)
}

func TestCreateSubscriptionSupportsMultipleAllowedOrigins(t *testing.T) {
	api := buildAPIHarness(t, nil, nil, nil)
	site := insertSite(t, api.database, "Multi Origins", "https://mprlab.com http://localhost:8080", "owner@example.com")

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
		t.Run(testCase.name, func(testingT *testing.T) {
			subscriberEmailValue := storage.NewID() + "@example.com"
			response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
				"site_id": site.ID,
				"email":   subscriberEmailValue,
			}, map[string]string{"Origin": testCase.originHeader})
			require.Equal(testingT, testCase.expectedStatus, response.Code)
		})
	}
}

func TestConfirmAndUnsubscribeSubscription(t *testing.T) {
	api := buildAPIHarness(t, nil, nil, nil)
	site := insertSite(t, api.database, "Confirmations", "http://confirm.example", "owner@example.com")

	createResp := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "confirm@example.com",
	}, map[string]string{"Origin": "http://confirm.example"})
	require.Equal(t, http.StatusOK, createResp.Code)

	confirm := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   "confirm@example.com",
	}, map[string]string{"Origin": "http://confirm.example"})
	require.Equal(t, http.StatusOK, confirm.Code)

	var confirmed model.Subscriber
	require.NoError(t, api.database.First(&confirmed).Error)
	require.Equal(t, model.SubscriberStatusConfirmed, confirmed.Status)
	require.False(t, confirmed.ConfirmedAt.IsZero())

	unsubscribe := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions/unsubscribe", map[string]any{
		"site_id": site.ID,
		"email":   "confirm@example.com",
	}, map[string]string{"Origin": "http://confirm.example"})
	require.Equal(t, http.StatusOK, unsubscribe.Code)

	var unsubscribed model.Subscriber
	require.NoError(t, api.database.First(&unsubscribed).Error)
	require.Equal(t, model.SubscriberStatusUnsubscribed, unsubscribed.Status)
	require.False(t, unsubscribed.UnsubscribedAt.IsZero())

	reconfirm := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   "confirm@example.com",
	}, map[string]string{"Origin": "http://confirm.example"})
	require.Equal(t, http.StatusConflict, reconfirm.Code)

	missing := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   "absent@example.com",
	}, map[string]string{"Origin": "http://confirm.example"})
	require.Equal(t, http.StatusNotFound, missing.Code)
}

func TestSubscriptionConfirmationEmailConfirmsViaLink(t *testing.T) {
	emailSender := &recordingEmailSender{t: t}
	subscriptionNotifier := &recordingSubscriptionNotifier{t: t}
	api := buildAPIHarness(t, nil, subscriptionNotifier, emailSender)
	site := insertSite(t, api.database, "Confirmation Email", "http://confirm.example", "owner@example.com")

	createResp := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "confirm@example.com",
	}, map[string]string{"Origin": "http://confirm.example"})
	require.Equal(t, http.StatusOK, createResp.Code)
	require.Equal(t, 1, emailSender.CallCount())
	require.Equal(t, 0, subscriptionNotifier.CallCount())

	lastEmail := emailSender.LastCall()
	require.Equal(t, "confirm@example.com", lastEmail.Recipient)
	require.Contains(t, lastEmail.Subject, "Confirm your subscription")

	var confirmationLink string
	for _, line := range strings.Split(lastEmail.Message, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "/subscriptions/confirm?token=") {
			confirmationLink = line
			break
		}
	}
	require.NotEmpty(t, confirmationLink)

	parsedURL, parseErr := url.Parse(confirmationLink)
	require.NoError(t, parseErr)

	confirmResponse := performJSONRequest(t, api.router, http.MethodGet, parsedURL.RequestURI(), nil, nil)
	require.Equal(t, http.StatusOK, confirmResponse.Code)
	confirmBody := confirmResponse.Body.String()
	require.Contains(t, confirmBody, "Subscription confirmed")
	require.Contains(t, confirmBody, "Open Confirmation Email")
	require.Contains(t, confirmBody, `href="http://confirm.example"`)

	var stored model.Subscriber
	require.NoError(t, api.database.First(&stored, "site_id = ? AND email = ?", site.ID, "confirm@example.com").Error)
	require.Equal(t, model.SubscriberStatusConfirmed, stored.Status)
	require.False(t, stored.ConfirmedAt.IsZero())

	require.Equal(t, 1, subscriptionNotifier.CallCount())
	notification := subscriptionNotifier.LastCall()
	require.Equal(t, site.ID, notification.Site.ID)
	require.Equal(t, "confirm@example.com", notification.Subscriber.Email)
}

func TestCreateSubscriptionDoesNotNotifyUntilConfirmed(t *testing.T) {
	subscriptionNotifier := &recordingSubscriptionNotifier{t: t}
	api := buildAPIHarness(t, nil, subscriptionNotifier, nil)
	site := insertSite(t, api.database, "Notify", "http://notify.example", "owner@example.com")

	resp := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "notify@example.com",
	}, map[string]string{"Origin": "http://notify.example"})
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 0, subscriptionNotifier.CallCount())

	confirm := performJSONRequest(t, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   "notify@example.com",
	}, map[string]string{"Origin": "http://notify.example"})
	require.Equal(t, http.StatusOK, confirm.Code)
	require.Equal(t, 1, subscriptionNotifier.CallCount())
}

func TestSubscriptionNotificationFailureDoesNotBlock(t *testing.T) {
	subscriptionNotifier := &recordingSubscriptionNotifier{t: t, callErr: errors.New("pinguin down")}

	gin.SetMode(gin.TestMode)
	logger, loggerErr := zap.NewDevelopment()
	require.NoError(t, loggerErr)

	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, openErr)
	database = testutil.ConfigureDatabaseLogger(t, database)
	require.NoError(t, storage.AutoMigrate(database))

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpapi.RequestLogger(logger))

	feedbackBroadcaster := httpapi.NewFeedbackEventBroadcaster()
	t.Cleanup(feedbackBroadcaster.Close)
	publicHandlers := httpapi.NewPublicHandlers(database, logger, feedbackBroadcaster, nil, nil, subscriptionNotifier, true, "http://loopaware.test", "unit-test-session-secret", nil)

	router.POST("/api/subscriptions", publicHandlers.CreateSubscription)
	router.POST("/api/subscriptions/confirm", publicHandlers.ConfirmSubscription)

	site := insertSite(t, database, "Notify Fail", "http://notifyfail.example", "owner@example.com")
	resp := performJSONRequest(t, router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "notify@example.com",
	}, map[string]string{"Origin": "http://notifyfail.example"})
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 0, subscriptionNotifier.CallCount())

	confirm := performJSONRequest(t, router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   "notify@example.com",
	}, map[string]string{"Origin": "http://notifyfail.example"})
	require.Equal(t, http.StatusOK, confirm.Code)
	require.Equal(t, 1, subscriptionNotifier.CallCount())
}

func TestSubscriptionNotificationsCanBeDisabled(t *testing.T) {
	subscriptionNotifier := &recordingSubscriptionNotifier{t: t}

	gin.SetMode(gin.TestMode)
	logger, loggerErr := zap.NewDevelopment()
	require.NoError(t, loggerErr)

	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, openErr)
	database = testutil.ConfigureDatabaseLogger(t, database)
	require.NoError(t, storage.AutoMigrate(database))

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpapi.RequestLogger(logger))

	feedbackBroadcaster := httpapi.NewFeedbackEventBroadcaster()
	t.Cleanup(feedbackBroadcaster.Close)
	publicHandlers := httpapi.NewPublicHandlers(database, logger, feedbackBroadcaster, nil, nil, subscriptionNotifier, false, "http://loopaware.test", "unit-test-session-secret", nil)
	router.POST("/api/subscriptions", publicHandlers.CreateSubscription)
	router.POST("/api/subscriptions/confirm", publicHandlers.ConfirmSubscription)

	site := insertSite(t, database, "Notify Off", "http://notifyoff.example", "owner@example.com")
	resp := performJSONRequest(t, router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   "notify@example.com",
	}, map[string]string{"Origin": "http://notifyoff.example"})
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 0, subscriptionNotifier.CallCount())

	confirm := performJSONRequest(t, router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   "notify@example.com",
	}, map[string]string{"Origin": "http://notifyoff.example"})
	require.Equal(t, http.StatusOK, confirm.Code)
	require.Equal(t, 0, subscriptionNotifier.CallCount())
}

func TestCollectVisitStoresRecord(t *testing.T) {
	api := buildAPIHarness(t, nil, nil, nil)
	site := insertSite(t, api.database, "Visits", "http://visits.example", "owner@example.com")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/visits?site_id="+site.ID+"&url=http://visits.example/page", nil)
	request.Header.Set("Origin", "http://visits.example")

	api.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "image/gif")

	var stored model.SiteVisit
	require.NoError(t, api.database.First(&stored).Error)
	require.Equal(t, site.ID, stored.SiteID)
	require.Equal(t, "http://visits.example/page", stored.URL)
	require.Equal(t, "/page", stored.Path)
}

func TestCollectVisitValidatesInput(t *testing.T) {
	api := buildAPIHarness(t, nil, nil, nil)
	site := insertSite(t, api.database, "Visits Invalid", "http://visits.example", "owner@example.com")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/visits?site_id="+site.ID+"&url=//bad-url", nil)
	request.Header.Set("Origin", "http://visits.example")
	api.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusBadRequest, recorder.Code)

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/visits?site_id="+site.ID+"&url=http://visits.example/page", nil)
	request.Header.Set("Origin", "http://evil.example")
	api.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestCollectVisitRequiresMatchingURLOrigin(t *testing.T) {
	api := buildAPIHarness(t, nil, nil, nil)
	site := insertSite(t, api.database, "Visits Mismatch", "http://visits.example", "owner@example.com")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/visits?site_id="+site.ID+"&url=http://other.example/page", nil)
	request.Header.Set("Referer", "http://dashboard.loopaware.test/app/sites/"+site.ID+"/traffic-test")

	api.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusForbidden, recorder.Code)
}

type feedbackNotificationCall struct {
	Site     model.Site
	Feedback model.Feedback
	Context  context.Context
}

type recordingFeedbackNotifier struct {
	t         *testing.T
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
	t       *testing.T
	mu      sync.Mutex
	calls   []subscriptionNotificationCall
	callErr error
}

type emailSendCall struct {
	Recipient string
	Subject   string
	Message   string
	Context   context.Context
}

type recordingEmailSender struct {
	t       *testing.T
	mu      sync.Mutex
	calls   []emailSendCall
	callErr error
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
		sender.t.Fatalf("expected at least one email sender call")
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
		notifier.t.Fatalf("expected at least one subscription notifier call")
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
		notifier.t.Fatalf("expected at least one notifier call")
	}
	return notifier.calls[len(notifier.calls)-1]
}

func TestCreateFeedbackDispatchesNotificationToOwner(t *testing.T) {
	notifier := &recordingFeedbackNotifier{
		t:        t,
		delivery: model.FeedbackDeliveryMailed,
	}
	api := buildAPIHarness(t, notifier, nil, nil)
	site := insertSite(t, api.database, "Dispatcher", "http://dispatch.example", "owner@example.com")
	require.NoError(t, api.database.Model(&model.Site{}).
		Where("id = ?", site.ID).
		Update("creator_email", "registrar@example.com").Error)

	resp := performJSONRequest(t, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "submitter@example.com",
		"message": "Dispatch notification",
	}, map[string]string{"Origin": "http://dispatch.example"})
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 1, notifier.CallCount())

	var stored model.Feedback
	require.NoError(t, api.database.First(&stored).Error)
	require.Equal(t, model.FeedbackDeliveryMailed, stored.Delivery)

	lastCall := notifier.LastCall()
	require.Equal(t, site.ID, lastCall.Site.ID)
	require.Equal(t, "owner@example.com", lastCall.Site.OwnerEmail)
	require.Equal(t, stored.ID, lastCall.Feedback.ID)
}

func TestCreateFeedbackRecordsNoDeliveryOnNotifierFailure(t *testing.T) {
	notifier := &recordingFeedbackNotifier{
		t:         t,
		delivery:  model.FeedbackDeliveryMailed,
		callError: errors.New("send failed"),
	}
	api := buildAPIHarness(t, notifier, nil, nil)
	site := insertSite(t, api.database, "Failure Delivery", "http://failure.example", "owner@example.com")

	resp := performJSONRequest(t, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "submitter@example.com",
		"message": "Expect failure",
	}, map[string]string{"Origin": "http://failure.example"})
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 1, notifier.CallCount())

	var stored model.Feedback
	require.NoError(t, api.database.First(&stored).Error)
	require.Equal(t, model.FeedbackDeliveryNone, stored.Delivery)
}

func TestCreateFeedbackPersistsFailureDeliveryWhenNotifierReturnsStatusAndError(t *testing.T) {
	notifier := &recordingFeedbackNotifier{
		t:         t,
		delivery:  model.FeedbackDeliveryTexted,
		callError: errors.New("notifier failed"),
	}
	api := buildAPIHarness(t, notifier, nil, nil)
	site := insertSite(t, api.database, "Failure Delivery Status", "http://failure-status.example", "owner@example.com")

	resp := performJSONRequest(t, api.router, http.MethodPost, "/api/feedback", map[string]any{
		"site_id": site.ID,
		"contact": "submitter@example.com",
		"message": "Expect failure status",
	}, map[string]string{"Origin": "http://failure-status.example"})
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 1, notifier.CallCount())

	var stored model.Feedback
	require.NoError(t, api.database.First(&stored).Error)
	require.Equal(t, model.FeedbackDeliveryNone, stored.Delivery)
}
