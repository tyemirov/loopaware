package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
	"github.com/MarkoPoloResearchLab/loopaware/pkg/favicon"
)

const (
	testStreamSiteID                  = "stream-site"
	testStreamSiteName                = "Stream Site"
	testStreamAllowedOrigin           = "https://stream.example"
	testStreamOwnerEmail              = "stream-owner@example.com"
	testStreamUnauthorizedEmail       = "unauthorized@example.com"
	testStreamFaviconURL              = "https://stream.example/favicon.ico"
	testStreamFeedbackID              = "feedback-stream"
	testStreamSubscriberID            = "subscriber-stream"
	testStreamSubscriberEmail         = "subscriber@stream.example"
	testStreamPublicBaseURL           = "https://stream.example"
	testStreamSubscriptionSecret      = "stream-secret"
	testStreamTimeout                 = 2 * time.Second
	testStreamPollInterval            = 10 * time.Millisecond
	testStreamWriteErrorMessage       = "stream write error"
	testStreamFaviconEventsPath       = "/api/sites/favicons/events"
	testStreamFeedbackEventsPath      = "/api/sites/feedback/events"
	testStreamFaviconShutdownMessage  = "timeout waiting for favicon stream shutdown"
	testStreamFeedbackShutdownMessage = "timeout waiting for feedback stream shutdown"
	testStreamFeedbackWriteMessage    = "timeout waiting for feedback stream write"
)

type staticResolver struct{}

func (resolver *staticResolver) Resolve(context.Context, string) (string, error) {
	return "", nil
}

func (resolver *staticResolver) ResolveAsset(context.Context, string) (*favicon.Asset, error) {
	return nil, nil
}

type notifyingRecorder struct {
	*httptest.ResponseRecorder
	writeNotification chan struct{}
	mutex             sync.Mutex
}

type erroringRecorder struct {
	header            http.Header
	statusCode        int
	writeNotification chan struct{}
	mutex             sync.Mutex
}

func newNotifyingRecorder() *notifyingRecorder {
	return &notifyingRecorder{
		ResponseRecorder:  httptest.NewRecorder(),
		writeNotification: make(chan struct{}, 1),
	}
}

func newErroringRecorder() *erroringRecorder {
	return &erroringRecorder{
		header:            make(http.Header),
		writeNotification: make(chan struct{}, 1),
	}
}

func (recorder *notifyingRecorder) WriteHeader(statusCode int) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	recorder.ResponseRecorder.WriteHeader(statusCode)
}

func (recorder *notifyingRecorder) Write(data []byte) (int, error) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	written, writeErr := recorder.ResponseRecorder.Write(data)
	select {
	case recorder.writeNotification <- struct{}{}:
	default:
	}
	return written, writeErr
}

func (recorder *notifyingRecorder) Flush() {}

func (recorder *notifyingRecorder) BodyString() string {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	return recorder.Body.String()
}

func (recorder *erroringRecorder) Header() http.Header {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	return recorder.header
}

func (recorder *erroringRecorder) WriteHeader(statusCode int) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	recorder.statusCode = statusCode
}

func (recorder *erroringRecorder) Write(data []byte) (int, error) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	select {
	case recorder.writeNotification <- struct{}{}:
	default:
	}
	return 0, errors.New(testStreamWriteErrorMessage)
}

func (recorder *erroringRecorder) Flush() {}

func openStreamDatabase(testingT *testing.T) *gorm.DB {
	testingT.Helper()
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))
	return database
}

func createStreamSite(testingT *testing.T, database *gorm.DB) model.Site {
	testingT.Helper()
	site := model.Site{
		ID:            testStreamSiteID,
		Name:          testStreamSiteName,
		AllowedOrigin: testStreamAllowedOrigin,
		OwnerEmail:    testStreamOwnerEmail,
		CreatorEmail:  testStreamOwnerEmail,
	}
	require.NoError(testingT, database.Create(&site).Error)
	return site
}

func waitForFaviconSubscriber(testingT *testing.T, manager *SiteFaviconManager) {
	testingT.Helper()
	require.Eventually(testingT, func() bool {
		manager.subscribersMutex.RLock()
		count := len(manager.subscribers)
		manager.subscribersMutex.RUnlock()
		return count > 0
	}, testStreamTimeout, testStreamPollInterval)
}

func waitForFeedbackSubscriber(testingT *testing.T, broadcaster *FeedbackEventBroadcaster) {
	testingT.Helper()
	require.Eventually(testingT, func() bool {
		broadcaster.mutex.Lock()
		count := len(broadcaster.subscribers)
		broadcaster.mutex.Unlock()
		return count > 0
	}, testStreamTimeout, testStreamPollInterval)
}

func waitForSubscriptionSubscriber(testingT *testing.T, broadcaster *SubscriptionTestEventBroadcaster) {
	testingT.Helper()
	require.Eventually(testingT, func() bool {
		broadcaster.mutex.Lock()
		count := len(broadcaster.subscribers)
		broadcaster.mutex.Unlock()
		return count > 0
	}, testStreamTimeout, testStreamPollInterval)
}

func TestStreamFaviconUpdatesWritesEvent(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	database := openStreamDatabase(testingT)
	site := createStreamSite(testingT, database)

	siteFaviconManager := NewSiteFaviconManager(database, favicon.NewService(&staticResolver{}), zap.NewNop())
	handlers := NewSiteHandlers(database, zap.NewNop(), testStreamPublicBaseURL, siteFaviconManager, nil, nil)

	recorder := newNotifyingRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	testingT.Cleanup(cancel)
	ginContext.Request = httptest.NewRequest(http.MethodGet, testStreamFaviconEventsPath, nil).WithContext(requestContext)
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: testStreamOwnerEmail, Role: RoleAdmin})

	streamDone := make(chan struct{})
	go func() {
		handlers.StreamFaviconUpdates(ginContext)
		close(streamDone)
	}()

	waitForFaviconSubscriber(testingT, siteFaviconManager)
	siteFaviconManager.broadcast(SiteFaviconEvent{
		SiteID:     site.ID,
		FaviconURL: testStreamFaviconURL,
		UpdatedAt:  time.Now().UTC(),
	})

	select {
	case <-recorder.writeNotification:
	case <-time.After(testStreamTimeout):
		testingT.Fatal("timeout waiting for favicon stream write")
	}

	cancel()
	select {
	case <-streamDone:
	case <-time.After(testStreamTimeout):
		testingT.Fatal(testStreamFaviconShutdownMessage)
	}

	body := recorder.BodyString()
	require.Contains(testingT, body, "favicon_updated")
	require.Contains(testingT, body, site.ID)
	require.Contains(testingT, body, testStreamFaviconURL)
}

func TestStreamFaviconUpdatesStopsOnWriteError(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	database := openStreamDatabase(testingT)
	site := createStreamSite(testingT, database)

	siteFaviconManager := NewSiteFaviconManager(database, favicon.NewService(&staticResolver{}), zap.NewNop())
	handlers := NewSiteHandlers(database, zap.NewNop(), testStreamPublicBaseURL, siteFaviconManager, nil, nil)

	recorder := newErroringRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	testingT.Cleanup(cancel)
	ginContext.Request = httptest.NewRequest(http.MethodGet, testStreamFaviconEventsPath, nil).WithContext(requestContext)
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: testStreamOwnerEmail, Role: RoleAdmin})

	streamDone := make(chan struct{})
	go func() {
		handlers.StreamFaviconUpdates(ginContext)
		close(streamDone)
	}()

	waitForFaviconSubscriber(testingT, siteFaviconManager)
	siteFaviconManager.broadcast(SiteFaviconEvent{
		SiteID:     site.ID,
		FaviconURL: testStreamFaviconURL,
		UpdatedAt:  time.Now().UTC(),
	})

	select {
	case <-recorder.writeNotification:
	case <-time.After(testStreamTimeout):
		testingT.Fatal("timeout waiting for favicon stream write error")
	}

	select {
	case <-streamDone:
	case <-time.After(testStreamTimeout):
		testingT.Fatal(testStreamFaviconShutdownMessage)
	}
}

func TestStreamFeedbackUpdatesWritesEvent(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	database := openStreamDatabase(testingT)
	site := createStreamSite(testingT, database)

	feedbackBroadcaster := NewFeedbackEventBroadcaster()
	handlers := NewSiteHandlers(database, zap.NewNop(), testStreamPublicBaseURL, nil, nil, feedbackBroadcaster)

	recorder := newNotifyingRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	testingT.Cleanup(cancel)
	ginContext.Request = httptest.NewRequest(http.MethodGet, testStreamFeedbackEventsPath, nil).WithContext(requestContext)
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: testStreamOwnerEmail, Role: RoleAdmin})

	streamDone := make(chan struct{})
	go func() {
		handlers.StreamFeedbackUpdates(ginContext)
		close(streamDone)
	}()

	waitForFeedbackSubscriber(testingT, feedbackBroadcaster)
	feedbackBroadcaster.Broadcast(FeedbackEvent{
		SiteID:        site.ID,
		FeedbackID:    testStreamFeedbackID,
		CreatedAt:     time.Now().UTC(),
		FeedbackCount: 1,
	})

	select {
	case <-recorder.writeNotification:
	case <-time.After(testStreamTimeout):
		testingT.Fatal(testStreamFeedbackWriteMessage)
	}

	cancel()
	select {
	case <-streamDone:
	case <-time.After(testStreamTimeout):
		testingT.Fatal(testStreamFeedbackShutdownMessage)
	}

	body := recorder.BodyString()
	require.Contains(testingT, body, feedbackCreatedEventName)
	require.Contains(testingT, body, site.ID)
	require.Contains(testingT, body, testStreamFeedbackID)
}

func TestStreamFeedbackUpdatesStopsOnWriteError(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	database := openStreamDatabase(testingT)
	site := createStreamSite(testingT, database)

	feedbackBroadcaster := NewFeedbackEventBroadcaster()
	handlers := NewSiteHandlers(database, zap.NewNop(), testStreamPublicBaseURL, nil, nil, feedbackBroadcaster)

	recorder := newErroringRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	testingT.Cleanup(cancel)
	ginContext.Request = httptest.NewRequest(http.MethodGet, testStreamFeedbackEventsPath, nil).WithContext(requestContext)
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: testStreamOwnerEmail, Role: RoleAdmin})

	streamDone := make(chan struct{})
	go func() {
		handlers.StreamFeedbackUpdates(ginContext)
		close(streamDone)
	}()

	waitForFeedbackSubscriber(testingT, feedbackBroadcaster)
	feedbackBroadcaster.Broadcast(FeedbackEvent{
		SiteID:        site.ID,
		FeedbackID:    testStreamFeedbackID,
		CreatedAt:     time.Now().UTC(),
		FeedbackCount: 1,
	})

	select {
	case <-recorder.writeNotification:
	case <-time.After(testStreamTimeout):
		testingT.Fatal("timeout waiting for feedback stream write error")
	}

	select {
	case <-streamDone:
	case <-time.After(testStreamTimeout):
		testingT.Fatal(testStreamFeedbackShutdownMessage)
	}
}

func TestStreamFeedbackUpdatesReturnsUnavailableWhenClosed(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	database := openStreamDatabase(testingT)

	feedbackBroadcaster := NewFeedbackEventBroadcaster()
	feedbackBroadcaster.Close()
	handlers := NewSiteHandlers(database, zap.NewNop(), testStreamPublicBaseURL, nil, nil, feedbackBroadcaster)

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, testStreamFeedbackEventsPath, nil)
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: testStreamOwnerEmail, Role: RoleAdmin})

	handlers.StreamFeedbackUpdates(ginContext)
	require.Equal(testingT, http.StatusServiceUnavailable, recorder.Code)
}

func TestStreamSubscriptionTestEventsWritesEvent(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	database := openStreamDatabase(testingT)
	site := createStreamSite(testingT, database)

	eventBroadcaster := NewSubscriptionTestEventBroadcaster()
	handlers := NewSiteSubscribeTestHandlers(database, zap.NewNop(), eventBroadcaster, nil, true, testStreamPublicBaseURL, testStreamSubscriptionSecret, nil)

	recorder := newNotifyingRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	testingT.Cleanup(cancel)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/app/sites/"+site.ID+"/subscribe-test/events", nil).WithContext(requestContext)
	ginContext.Params = gin.Params{gin.Param{Key: "id", Value: site.ID}}
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: testStreamOwnerEmail, Role: RoleAdmin})

	streamDone := make(chan struct{})
	go func() {
		handlers.StreamSubscriptionTestEvents(ginContext)
		close(streamDone)
	}()

	waitForSubscriptionSubscriber(testingT, eventBroadcaster)
	eventBroadcaster.Broadcast(SubscriptionTestEvent{
		SiteID:       site.ID,
		SubscriberID: testStreamSubscriberID,
		Email:        testStreamSubscriberEmail,
		EventType:    subscriptionEventTypeSubmission,
		Status:       subscriptionEventStatusSuccess,
	})

	select {
	case <-recorder.writeNotification:
	case <-time.After(testStreamTimeout):
		testingT.Fatal("timeout waiting for subscription stream write")
	}

	cancel()
	select {
	case <-streamDone:
	case <-time.After(testStreamTimeout):
		testingT.Fatal("timeout waiting for subscription stream shutdown")
	}

	body := recorder.BodyString()
	require.Contains(testingT, body, testStreamSubscriberID)
	require.Contains(testingT, body, testStreamSubscriberEmail)
	require.Contains(testingT, body, strings.ToLower(subscriptionEventTypeSubmission))
}

func TestStreamFaviconUpdatesSkipsUnauthorizedSite(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	database := openStreamDatabase(testingT)
	site := createStreamSite(testingT, database)

	siteFaviconManager := NewSiteFaviconManager(database, favicon.NewService(&staticResolver{}), zap.NewNop())
	handlers := NewSiteHandlers(database, zap.NewNop(), testStreamPublicBaseURL, siteFaviconManager, nil, nil)

	recorder := newNotifyingRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	testingT.Cleanup(cancel)
	ginContext.Request = httptest.NewRequest(http.MethodGet, testStreamFaviconEventsPath, nil).WithContext(requestContext)
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: "unauthorized@example.com", Role: RoleUser})

	streamDone := make(chan struct{})
	go func() {
		handlers.StreamFaviconUpdates(ginContext)
		close(streamDone)
	}()

	waitForFaviconSubscriber(testingT, siteFaviconManager)
	siteFaviconManager.broadcast(SiteFaviconEvent{
		SiteID:     site.ID,
		FaviconURL: testStreamFaviconURL,
		UpdatedAt:  time.Now().UTC(),
	})

	select {
	case <-recorder.writeNotification:
		testingT.Fatal("unexpected favicon stream write")
	case <-time.After(testStreamPollInterval * 4):
	}

	cancel()
	select {
	case <-streamDone:
	case <-time.After(testStreamTimeout):
		testingT.Fatal(testStreamFaviconShutdownMessage)
	}
}

func TestStreamFeedbackUpdatesSkipsEmptySiteID(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	database := openStreamDatabase(testingT)

	feedbackBroadcaster := NewFeedbackEventBroadcaster()
	handlers := NewSiteHandlers(database, zap.NewNop(), testStreamPublicBaseURL, nil, nil, feedbackBroadcaster)

	recorder := newNotifyingRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	testingT.Cleanup(cancel)
	ginContext.Request = httptest.NewRequest(http.MethodGet, testStreamFeedbackEventsPath, nil).WithContext(requestContext)
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: testStreamOwnerEmail, Role: RoleAdmin})

	streamDone := make(chan struct{})
	go func() {
		handlers.StreamFeedbackUpdates(ginContext)
		close(streamDone)
	}()

	waitForFeedbackSubscriber(testingT, feedbackBroadcaster)
	feedbackBroadcaster.Broadcast(FeedbackEvent{
		SiteID:        "",
		FeedbackID:    testStreamFeedbackID,
		FeedbackCount: 1,
	})

	select {
	case <-recorder.writeNotification:
		testingT.Fatal("unexpected feedback stream write")
	case <-time.After(testStreamPollInterval * 4):
	}

	cancel()
	select {
	case <-streamDone:
	case <-time.After(testStreamTimeout):
		testingT.Fatal(testStreamFeedbackShutdownMessage)
	}
}

func TestStreamFeedbackUpdatesDefaultsCreatedAt(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	database := openStreamDatabase(testingT)
	site := createStreamSite(testingT, database)

	feedbackBroadcaster := NewFeedbackEventBroadcaster()
	handlers := NewSiteHandlers(database, zap.NewNop(), testStreamPublicBaseURL, nil, nil, feedbackBroadcaster)

	recorder := newNotifyingRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	testingT.Cleanup(cancel)
	ginContext.Request = httptest.NewRequest(http.MethodGet, testStreamFeedbackEventsPath, nil).WithContext(requestContext)
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: testStreamOwnerEmail, Role: RoleAdmin})

	streamDone := make(chan struct{})
	go func() {
		handlers.StreamFeedbackUpdates(ginContext)
		close(streamDone)
	}()

	waitForFeedbackSubscriber(testingT, feedbackBroadcaster)
	feedbackBroadcaster.Broadcast(FeedbackEvent{
		SiteID:        site.ID,
		FeedbackID:    testStreamFeedbackID,
		FeedbackCount: 1,
	})

	select {
	case <-recorder.writeNotification:
	case <-time.After(testStreamTimeout):
		testingT.Fatal(testStreamFeedbackWriteMessage)
	}

	cancel()
	select {
	case <-streamDone:
	case <-time.After(testStreamTimeout):
		testingT.Fatal(testStreamFeedbackShutdownMessage)
	}

	payloadStart := strings.Index(recorder.BodyString(), "data: ")
	require.NotEqual(testingT, -1, payloadStart)
	payloadLine := recorder.BodyString()[payloadStart+len("data: "):]
	payloadLine = strings.TrimSpace(strings.Split(payloadLine, "\n")[0])
	var payload struct {
		CreatedAt int64 `json:"created_at"`
	}
	require.NoError(testingT, json.Unmarshal([]byte(payloadLine), &payload))
	require.Greater(testingT, payload.CreatedAt, int64(0))
}

func TestStreamFeedbackUpdatesSkipsUnauthorizedSite(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	database := openStreamDatabase(testingT)
	site := createStreamSite(testingT, database)

	feedbackBroadcaster := NewFeedbackEventBroadcaster()
	handlers := NewSiteHandlers(database, zap.NewNop(), testStreamPublicBaseURL, nil, nil, feedbackBroadcaster)

	recorder := newNotifyingRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	testingT.Cleanup(cancel)
	ginContext.Request = httptest.NewRequest(http.MethodGet, testStreamFeedbackEventsPath, nil).WithContext(requestContext)
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: testStreamUnauthorizedEmail, Role: RoleUser})

	streamDone := make(chan struct{})
	go func() {
		handlers.StreamFeedbackUpdates(ginContext)
		close(streamDone)
	}()

	waitForFeedbackSubscriber(testingT, feedbackBroadcaster)
	feedbackBroadcaster.Broadcast(FeedbackEvent{
		SiteID:        site.ID,
		FeedbackID:    testStreamFeedbackID,
		CreatedAt:     time.Now().UTC(),
		FeedbackCount: 1,
	})

	select {
	case <-recorder.writeNotification:
		testingT.Fatal("unexpected feedback stream write")
	case <-time.After(testStreamPollInterval * 4):
	}

	cancel()
	select {
	case <-streamDone:
	case <-time.After(testStreamTimeout):
		testingT.Fatal(testStreamFeedbackShutdownMessage)
	}
}
