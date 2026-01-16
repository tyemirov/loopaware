package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	testSubscribeQueryCallbackName  = "force_subscribe_query_error"
	testSubscribeQueryErrorMessage  = "subscription query failed"
	testSubscribeUpdateCallbackName = "force_subscribe_update_error"
	testSubscribeUpdateErrorMessage = "subscription update failed"
	testSubscribeCreateCallbackName = "force_subscribe_create_error"
	testSubscribeCreateErrorMessage = "subscription create failed"
	testSubscribeTableName          = "subscribers"
	testSubscribeSourceURL          = "https://source.example"
	testSubscribeSiteIDParamKey     = "id"
	testSubscribeNamePadding        = "n"
	testSubscribeNameLength         = 201
	testSubscribeStreamDataPrefix   = "data: "
	testSubscribeOtherSiteID        = "subscribe-other-site"
	testSubscribeSubscriberID       = "subscribe-subscriber"
)

func buildSubscribePayload(testingT *testing.T) []byte {
	payload := createSubscriptionRequest{
		Email:     testSubscriberEmail,
		Name:      testSubscriberName,
		SourceURL: testSubscribeSourceURL,
	}
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)
	return body
}

func assertSubscribeErrorResponse(testingT *testing.T, recorder *httptest.ResponseRecorder, expectedError string) {
	responsePayload := map[string]string{}
	unmarshalErr := json.Unmarshal(recorder.Body.Bytes(), &responsePayload)
	require.NoError(testingT, unmarshalErr)
	require.Equal(testingT, expectedError, responsePayload[jsonKeyError])
}

func TestSiteSubscribeTestHandlersSendSubscriptionConfirmationHandlesNil(testingT *testing.T) {
	var handlers *SiteSubscribeTestHandlers
	handlers.sendSubscriptionConfirmation(context.Background(), model.Site{}, model.Subscriber{})
}

func TestSiteSubscribeTestHandlersSendSubscriptionConfirmationUsesSender(testingT *testing.T) {
	emailSender := &stubEmailSender{}
	handlers := &SiteSubscribeTestHandlers{
		logger:                  zap.NewNop(),
		publicBaseURL:           testConfirmationBaseURL,
		subscriptionTokenSecret: testConfirmationTokenSecret,
		subscriptionTokenTTL:    time.Hour,
		confirmationEmailSender: emailSender,
	}

	site := model.Site{ID: testConfirmationSiteID, Name: testConfirmationSiteName}
	subscriber := model.Subscriber{
		ID:     testConfirmationSubscriber,
		SiteID: testConfirmationSiteID,
		Email:  testConfirmationEmail,
		Status: model.SubscriberStatusPending,
	}

	handlers.sendSubscriptionConfirmation(context.Background(), site, subscriber)

	require.Equal(testingT, testConfirmationEmail, emailSender.recipient)
	require.NotEmpty(testingT, emailSender.subjectLine)
	require.NotEmpty(testingT, emailSender.messageBody)
}

func TestSiteSubscribeTestHandlersRecordSubscriptionTestEventSkipsNil(testingT *testing.T) {
	var handlers *SiteSubscribeTestHandlers
	handlers.recordSubscriptionTestEvent(model.Site{}, model.Subscriber{}, "", "", "")
}

func TestSiteSubscribeTestHandlersRecordSubscriptionTestEventSkipsEmptyIDs(testingT *testing.T) {
	broadcaster := NewSubscriptionTestEventBroadcaster()
	testingT.Cleanup(broadcaster.Close)
	handlers := &SiteSubscribeTestHandlers{eventBroadcaster: broadcaster}
	subscription := broadcaster.Subscribe()
	testingT.Cleanup(subscription.Close)

	handlers.recordSubscriptionTestEvent(model.Site{}, model.Subscriber{ID: "subscriber"}, "", "", "")

	select {
	case <-subscription.Events():
		testingT.Fatal("unexpected subscription event")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSiteSubscribeTestHandlersRecordSubscriptionTestEventDefaults(testingT *testing.T) {
	broadcaster := NewSubscriptionTestEventBroadcaster()
	testingT.Cleanup(broadcaster.Close)
	handlers := &SiteSubscribeTestHandlers{eventBroadcaster: broadcaster}
	subscription := broadcaster.Subscribe()
	testingT.Cleanup(subscription.Close)

	site := model.Site{ID: "site-1"}
	subscriber := model.Subscriber{ID: "subscriber-1", Email: "User@Example.com"}
	handlers.recordSubscriptionTestEvent(site, subscriber, "", "", " ")

	select {
	case event := <-subscription.Events():
		require.Equal(testingT, site.ID, event.SiteID)
		require.Equal(testingT, subscriber.ID, event.SubscriberID)
		require.Equal(testingT, "user@example.com", event.Email)
		require.Equal(testingT, subscriptionEventTypeSubmission, event.EventType)
		require.Equal(testingT, subscriptionEventStatusSuccess, event.Status)
	case <-time.After(time.Second):
		testingT.Fatal("expected subscription event")
	}
}

func TestCreateSubscriptionReportsFindSubscriberError(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)

	callbackName := testSubscribeQueryCallbackName
	handlers.database.Callback().Query().Before("gorm:query").Register(callbackName, func(database *gorm.DB) {
		if database.Statement != nil && database.Statement.Table == testSubscribeTableName {
			database.AddError(errors.New(testSubscribeQueryErrorMessage))
		}
	})
	testingT.Cleanup(func() {
		handlers.database.Callback().Query().Remove(callbackName)
	})

	context, recorder := buildSubscribeContext(http.MethodPost, testSubscribeCreatePath, buildSubscribePayload(testingT))
	context.Params = gin.Params{{Key: testSubscribeSiteIDParamKey, Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
	assertSubscribeErrorResponse(testingT, recorder, errorValueSaveSubscriberFailed)
}

func TestCreateSubscriptionReportsUpdateError(testingT *testing.T) {
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

	callbackName := testSubscribeUpdateCallbackName
	handlers.database.Callback().Update().Before("gorm:update").Register(callbackName, func(database *gorm.DB) {
		if database.Statement != nil && database.Statement.Table == testSubscribeTableName {
			database.AddError(errors.New(testSubscribeUpdateErrorMessage))
		}
	})
	testingT.Cleanup(func() {
		handlers.database.Callback().Update().Remove(callbackName)
	})

	context, recorder := buildSubscribeContext(http.MethodPost, testSubscribeCreatePath, buildSubscribePayload(testingT))
	context.Params = gin.Params{{Key: testSubscribeSiteIDParamKey, Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
	assertSubscribeErrorResponse(testingT, recorder, errorValueSaveSubscriberFailed)
}

func TestCreateSubscriptionReportsCreateError(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)

	callbackName := testSubscribeCreateCallbackName
	handlers.database.Callback().Create().Before("gorm:create").Register(callbackName, func(database *gorm.DB) {
		if database.Statement != nil && database.Statement.Table == testSubscribeTableName {
			database.AddError(errors.New(testSubscribeCreateErrorMessage))
		}
	})
	testingT.Cleanup(func() {
		handlers.database.Callback().Create().Remove(callbackName)
	})

	context, recorder := buildSubscribeContext(http.MethodPost, testSubscribeCreatePath, buildSubscribePayload(testingT))
	context.Params = gin.Params{{Key: testSubscribeSiteIDParamKey, Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
	assertSubscribeErrorResponse(testingT, recorder, errorValueSaveSubscriberFailed)
}

func TestCreateSubscriptionRejectsInvalidContact(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)

	longName := strings.Repeat(testSubscribeNamePadding, testSubscribeNameLength)
	payload := createSubscriptionRequest{
		Email:     testSubscriberEmail,
		Name:      longName,
		SourceURL: testSubscribeSourceURL,
	}
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)

	context, recorder := buildSubscribeContext(http.MethodPost, testSubscribeCreatePath, body)
	context.Params = gin.Params{{Key: testSubscribeSiteIDParamKey, Value: testSubscribeSiteID}}
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	handlers.CreateSubscription(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
	assertSubscribeErrorResponse(testingT, recorder, errorValueInvalidEmail)
}

func TestStreamSubscriptionTestEventsStreamsMatchingEvents(testingT *testing.T) {
	handlers := buildSubscribeHandlers(testingT)
	insertSubscribeSite(testingT, handlers)

	recorder := newNotifyingRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	testingT.Cleanup(cancel)
	ginContext.Request = httptest.NewRequest(http.MethodGet, testSubscribeEventsPath, nil).WithContext(requestContext)
	ginContext.Params = gin.Params{{Key: testSubscribeSiteIDParamKey, Value: testSubscribeSiteID}}
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: testSubscribeOwnerEmail, Role: RoleUser})

	streamDone := make(chan struct{})
	go func() {
		handlers.StreamSubscriptionTestEvents(ginContext)
		close(streamDone)
	}()

	waitForSubscriptionSubscriber(testingT, handlers.eventBroadcaster)
	handlers.eventBroadcaster.Broadcast(SubscriptionTestEvent{
		SiteID:       testSubscribeOtherSiteID,
		SubscriberID: testSubscribeSubscriberID,
		Email:        testSubscriberEmail,
		EventType:    subscriptionEventTypeSubmission,
		Status:       subscriptionEventStatusSuccess,
	})
	handlers.eventBroadcaster.Broadcast(SubscriptionTestEvent{
		SiteID:       testSubscribeSiteID,
		SubscriberID: testSubscribeSubscriberID,
		Email:        testSubscriberEmail,
		EventType:    subscriptionEventTypeSubmission,
		Status:       subscriptionEventStatusSuccess,
	})

	select {
	case <-recorder.writeNotification:
	case <-time.After(testStreamTimeout):
		testingT.Fatal("timeout waiting for subscription stream write")
	}

	handlers.eventBroadcaster.Close()

	select {
	case <-streamDone:
	case <-time.After(testStreamTimeout):
		testingT.Fatal("timeout waiting for subscription stream shutdown")
	}

	require.Contains(testingT, recorder.BodyString(), testSubscribeStreamDataPrefix)
}
