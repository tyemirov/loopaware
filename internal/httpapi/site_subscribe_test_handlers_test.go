package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

const (
	testSubscribeSiteID      = "subscribe-site-id"
	testSubscribeSiteName    = "Subscribe Site"
	testSubscribeOwnerEmail  = "subscribe-owner@example.com"
	testSubscribeBaseURL     = "https://subscribe.example.com"
	testSubscribeTokenSecret = "subscribe-secret"
	testSubscriberEmail      = "subscriber@example.com"
	testSubscriberName       = "Subscriber Name"
	testOtherEmail           = "other@example.com"
	testSubscribeRenderPath  = "/app/sites/" + testSubscribeSiteID + "/subscribe-test"
	testSubscribeCreatePath  = "/app/sites/" + testSubscribeSiteID + "/subscribe-test/subscriptions"
	testSubscribeEventsPath  = "/app/sites/" + testSubscribeSiteID + "/subscribe-test/events"
)

func buildSubscribeHandlers(testingT *testing.T) *SiteSubscribeTestHandlers {
	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	broadcaster := NewSubscriptionTestEventBroadcaster()
	testingT.Cleanup(broadcaster.Close)

	handlers := NewSiteSubscribeTestHandlers(database, zap.NewNop(), broadcaster, nil, false, testSubscribeBaseURL, testSubscribeTokenSecret, nil, AuthClientConfig{})
	return handlers
}

func buildSubscribeContext(method string, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	context.Request = request
	return context, recorder
}

func insertSubscribeSite(testingT *testing.T, handlers *SiteSubscribeTestHandlers) {
	site := model.Site{
		ID:            testSubscribeSiteID,
		Name:          testSubscribeSiteName,
		OwnerEmail:    testSubscribeOwnerEmail,
		AllowedOrigin: testSubscribeBaseURL,
	}
	require.NoError(testingT, handlers.database.Create(&site).Error)
}

func TestRenderSubscribeTestPageRequiresSiteID(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	context, recorder := buildSubscribeContext(http.MethodGet, "/app/sites//subscribe-test", nil)

	handlers.RenderSubscribeTestPage(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestRenderSubscribeTestPageRequiresUser(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	context, recorder := buildSubscribeContext(http.MethodGet, testSubscribeRenderPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}

	handlers.RenderSubscribeTestPage(context)
	require.Equal(testingT, http.StatusFound, recorder.Code)
}

func TestRenderSubscribeTestPageReturnsNotFound(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	context, recorder := buildSubscribeContext(http.MethodGet, testSubscribeRenderPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.RenderSubscribeTestPage(context)
	require.Equal(testingT, http.StatusNotFound, recorder.Code)
}

func TestRenderSubscribeTestPageRejectsForbidden(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)

	context, recorder := buildSubscribeContext(http.MethodGet, testSubscribeRenderPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testOtherEmail, Role: RoleUser})

	handlers.RenderSubscribeTestPage(context)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)
}

func TestRenderSubscribeTestPageRendersHTML(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)

	context, recorder := buildSubscribeContext(http.MethodGet, testSubscribeRenderPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.RenderSubscribeTestPage(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), testSubscribeSiteName)
}

func TestStreamSubscriptionTestEventsRequiresSiteID(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	context, recorder := buildSubscribeContext(http.MethodGet, "/app/sites//subscribe-test/events", nil)

	handlers.StreamSubscriptionTestEvents(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestStreamSubscriptionTestEventsRequiresUser(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	context, recorder := buildSubscribeContext(http.MethodGet, testSubscribeEventsPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}

	handlers.StreamSubscriptionTestEvents(context)
	require.Equal(testingT, http.StatusUnauthorized, recorder.Code)
}

func TestStreamSubscriptionTestEventsReturnsNotFound(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	context, recorder := buildSubscribeContext(http.MethodGet, testSubscribeEventsPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.StreamSubscriptionTestEvents(context)
	require.Equal(testingT, http.StatusNotFound, recorder.Code)
}

func TestStreamSubscriptionTestEventsRejectsForbidden(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)

	context, recorder := buildSubscribeContext(http.MethodGet, testSubscribeEventsPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testOtherEmail, Role: RoleUser})

	handlers.StreamSubscriptionTestEvents(context)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)
}

func TestStreamSubscriptionTestEventsRequiresBroadcaster(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)
	handlers.eventBroadcaster = nil

	context, recorder := buildSubscribeContext(http.MethodGet, testSubscribeEventsPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.StreamSubscriptionTestEvents(context)
	require.Equal(testingT, http.StatusNoContent, recorder.Code)
}

func TestStreamSubscriptionTestEventsReturnsNoContentWhenClosed(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)
	handlers.eventBroadcaster.Close()

	context, recorder := buildSubscribeContext(http.MethodGet, testSubscribeEventsPath, nil)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.StreamSubscriptionTestEvents(context)
	require.Equal(testingT, http.StatusNoContent, recorder.Code)
}

func TestCreateSubscriptionRequiresSiteID(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	context, recorder := buildSubscribeContext(http.MethodPost, "/app/sites//subscribe-test/subscriptions", nil)

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestCreateSubscriptionRequiresUser(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	context, recorder := buildSubscribeContext(http.MethodPost, testSubscribeCreatePath, nil)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusUnauthorized, recorder.Code)
}

func TestCreateSubscriptionInvalidJSON(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	context, recorder := buildSubscribeContext(http.MethodPost, testSubscribeCreatePath, []byte("{"))
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestCreateSubscriptionRequiresEmail(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	payload := createSubscriptionRequest{Email: "", Name: testSubscriberName}
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)

	context, recorder := buildSubscribeContext(http.MethodPost, testSubscribeCreatePath, body)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestCreateSubscriptionReturnsNotFound(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	payload := createSubscriptionRequest{Email: testSubscriberEmail, Name: testSubscriberName}
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)

	context, recorder := buildSubscribeContext(http.MethodPost, testSubscribeCreatePath, body)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusNotFound, recorder.Code)
}

func TestCreateSubscriptionRejectsForbidden(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)
	payload := createSubscriptionRequest{Email: testSubscriberEmail, Name: testSubscriberName}
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)

	context, recorder := buildSubscribeContext(http.MethodPost, testSubscribeCreatePath, body)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testOtherEmail, Role: RoleUser})

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)
}

func TestCreateSubscriptionRejectsInvalidEmail(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)
	payload := createSubscriptionRequest{Email: "invalid", Name: testSubscriberName}
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)

	context, recorder := buildSubscribeContext(http.MethodPost, testSubscribeCreatePath, body)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestCreateSubscriptionCreatesSubscriber(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)
	payload := createSubscriptionRequest{Email: testSubscriberEmail, Name: testSubscriberName}
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)

	context, recorder := buildSubscribeContext(http.MethodPost, testSubscribeCreatePath, body)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var subscriber model.Subscriber
	require.NoError(testingT, handlers.database.First(&subscriber, "email = ?", testSubscriberEmail).Error)
	require.Equal(testingT, model.SubscriberStatusPending, subscriber.Status)
}

func TestCreateSubscriptionReturnsConflictForExistingSubscriber(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)

	existingSubscriber, subscriberErr := model.NewSubscriber(model.SubscriberInput{
		SiteID:    testSubscribeSiteID,
		Email:     testSubscriberEmail,
		Name:      testSubscriberName,
		Status:    model.SubscriberStatusConfirmed,
		ConsentAt: time.Now().UTC(),
	})
	require.NoError(testingT, subscriberErr)
	require.NoError(testingT, handlers.database.Create(&existingSubscriber).Error)

	payload := createSubscriptionRequest{Email: testSubscriberEmail, Name: testSubscriberName}
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)

	context, recorder := buildSubscribeContext(http.MethodPost, testSubscribeCreatePath, body)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusConflict, recorder.Code)
}

func TestCreateSubscriptionUpdatesUnsubscribedSubscriber(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)

	existingSubscriber, subscriberErr := model.NewSubscriber(model.SubscriberInput{
		SiteID:         testSubscribeSiteID,
		Email:          testSubscriberEmail,
		Name:           testSubscriberName,
		Status:         model.SubscriberStatusUnsubscribed,
		ConsentAt:      time.Now().UTC(),
		UnsubscribedAt: time.Now().UTC(),
	})
	require.NoError(testingT, subscriberErr)
	require.NoError(testingT, handlers.database.Create(&existingSubscriber).Error)

	payload := createSubscriptionRequest{Email: testSubscriberEmail, Name: testSubscriberName}
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)

	context, recorder := buildSubscribeContext(http.MethodPost, testSubscribeCreatePath, body)
	context.Params = gin.Params{{Key: "id", Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var subscriber model.Subscriber
	require.NoError(testingT, handlers.database.First(&subscriber, "email = ?", testSubscriberEmail).Error)
	require.Equal(testingT, model.SubscriberStatusPending, subscriber.Status)
}
