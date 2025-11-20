package httpapi_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

	"github.com/MarkoPoloResearchLab/loopaware/internal/httpapi"
	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
	"github.com/MarkoPoloResearchLab/loopaware/pkg/favicon"
)

const (
	testAdminEmailAddress               = "admin@example.com"
	testUserEmailAddress                = "user@example.com"
	testCreatorEmail                    = "creator@example.com"
	testAlternateOwnerEmailAddress      = "owner@example.com"
	testSessionContextKey               = "httpapi_current_user"
	testWidgetBaseURL                   = "https://gravity.mprlab.com/"
	jsonErrorKey                        = "error"
	errorCodeSiteExists                 = "site_exists"
	errorCodeInvalidWidgetSide          = "invalid_widget_side"
	errorCodeInvalidWidgetOffset        = "invalid_widget_offset"
	defaultWidgetTestBubbleSide         = "right"
	defaultWidgetTestBottomOffsetPixels = 16
	customWidgetTestBubbleSide          = "left"
	customWidgetTestBottomOffsetPixels  = 48
)

type siteTestHarness struct {
	handlers *httpapi.SiteHandlers
	database *gorm.DB
}

func newSiteTestHarness(testingT *testing.T) siteTestHarness {
	testingT.Helper()

	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	handlers := httpapi.NewSiteHandlers(database, zap.NewNop(), testWidgetBaseURL, nil, nil, nil)

	return siteTestHarness{handlers: handlers, database: database}
}

func TestCurrentUserReturnsAvatarPayload(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	recorder, context := newJSONContext(http.MethodGet, "/api/me", nil)
	expectedAvatarURL := "/api/me/avatar?v=123"
	context.Set(testSessionContextKey, &httpapi.CurrentUser{
		Email:      testUserEmailAddress,
		Name:       "Display Name",
		Role:       httpapi.RoleUser,
		PictureURL: expectedAvatarURL,
	})

	harness.handlers.CurrentUser(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var payload struct {
		Email  string `json:"email"`
		Name   string `json:"name"`
		Role   string `json:"role"`
		Avatar struct {
			URL string `json:"url"`
		} `json:"avatar"`
	}
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(testingT, testUserEmailAddress, payload.Email)
	require.Equal(testingT, "Display Name", payload.Name)
	require.Equal(testingT, string(httpapi.RoleUser), payload.Role)
	require.Equal(testingT, expectedAvatarURL, payload.Avatar.URL)
}

func TestListMessagesBySiteReturnsOrderedUnixTimestamps(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "List Messages Site",
		AllowedOrigin: "http://list.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	firstFeedback := model.Feedback{
		ID:        storage.NewID(),
		SiteID:    site.ID,
		Contact:   "first@example.com",
		Message:   "First",
		CreatedAt: time.Now().Add(-time.Minute),
	}
	secondFeedback := model.Feedback{
		ID:        storage.NewID(),
		SiteID:    site.ID,
		Contact:   "second@example.com",
		Message:   "Second",
		CreatedAt: time.Now(),
	}
	require.NoError(testingT, harness.database.Create(&firstFeedback).Error)
	require.NoError(testingT, harness.database.Create(&secondFeedback).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/sites/"+site.ID+"/messages", nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

	harness.handlers.ListMessagesBySite(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody struct {
		SiteID   string `json:"site_id"`
		Messages []struct {
			Identifier string `json:"id"`
			CreatedAt  int64  `json:"created_at"`
		} `json:"messages"`
	}
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, site.ID, responseBody.SiteID)
	require.Len(testingT, responseBody.Messages, 2)
	require.GreaterOrEqual(testingT, responseBody.Messages[0].CreatedAt, responseBody.Messages[1].CreatedAt)
}

func TestListMessagesBySiteIncludesDelivery(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Delivery Site",
		AllowedOrigin: "http://delivery.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	feedback := model.Feedback{
		ID:        storage.NewID(),
		SiteID:    site.ID,
		Contact:   "submitter@example.com",
		Message:   "Delivery expectations",
		Delivery:  model.FeedbackDeliveryTexted,
		CreatedAt: time.Now(),
	}
	require.NoError(testingT, harness.database.Create(&feedback).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/sites/"+site.ID+"/messages", nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

	harness.handlers.ListMessagesBySite(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody struct {
		Messages []struct {
			Delivery string `json:"delivery"`
		} `json:"messages"`
	}
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Len(testingT, responseBody.Messages, 1)
	require.Equal(testingT, model.FeedbackDeliveryTexted, responseBody.Messages[0].Delivery)
}

func TestListSitesUsesPublicBaseURLForWidget(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Widget Site",
		AllowedOrigin:              "https://client.example",
		OwnerEmail:                 testAdminEmailAddress,
		WidgetBubbleSide:           defaultWidgetTestBubbleSide,
		WidgetBubbleBottomOffsetPx: defaultWidgetTestBottomOffsetPixels,
		FaviconData:                []byte{0x01, 0x02, 0x03},
		FaviconContentType:         "image/png",
		FaviconOrigin:              "https://client.example",
		FaviconFetchedAt:           time.Now(),
	}
	require.NoError(testingT, harness.database.Create(&site).Error)
	for index := 0; index < 5; index++ {
		feedback := model.Feedback{
			ID:      storage.NewID(),
			SiteID:  site.ID,
			Contact: fmt.Sprintf("contact-%d@example.com", index),
			Message: fmt.Sprintf("Message %d", index),
		}
		require.NoError(testingT, harness.database.Create(&feedback).Error)
	}
	recorder, context := newJSONContext(http.MethodGet, "/api/sites", nil)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

	harness.handlers.ListSites(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody struct {
		Sites []struct {
			Identifier               string `json:"id"`
			Widget                   string `json:"widget"`
			FaviconURL               string `json:"favicon_url"`
			FeedbackCount            int64  `json:"feedback_count"`
			WidgetBubbleSide         string `json:"widget_bubble_side"`
			WidgetBubbleBottomOffset int    `json:"widget_bubble_bottom_offset"`
		} `json:"sites"`
	}
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Len(testingT, responseBody.Sites, 1)

	expectedBaseURL := strings.TrimRight(testWidgetBaseURL, "/")
	expectedWidget := fmt.Sprintf("<script defer src=\"%s/widget.js?site_id=%s\"></script>", expectedBaseURL, site.ID)
	require.Equal(testingT, expectedWidget, responseBody.Sites[0].Widget)
	expectedFavicon := fmt.Sprintf("/api/sites/%s/favicon?ts=%d", site.ID, site.FaviconFetchedAt.UTC().Unix())
	require.Equal(testingT, expectedFavicon, responseBody.Sites[0].FaviconURL)
	require.Equal(testingT, int64(5), responseBody.Sites[0].FeedbackCount)
	require.Equal(testingT, defaultWidgetTestBubbleSide, responseBody.Sites[0].WidgetBubbleSide)
	require.Equal(testingT, defaultWidgetTestBottomOffsetPixels, responseBody.Sites[0].WidgetBubbleBottomOffset)
}

func TestListSitesReturnsOwnerSitesForNonAdminRegardlessOfEmailCase(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	ownerEmail := "Owner.MixedCase@Example.com"
	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Owner Filter Site",
		AllowedOrigin: "https://filters.example",
		OwnerEmail:    ownerEmail,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/sites", nil)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: strings.ToLower(ownerEmail), Role: httpapi.RoleUser})

	harness.handlers.ListSites(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody struct {
		Sites []struct {
			Identifier string `json:"id"`
		} `json:"sites"`
	}
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Len(testingT, responseBody.Sites, 1)
	require.Equal(testingT, site.ID, responseBody.Sites[0].Identifier)
}

func TestListSitesReturnsCreatorSitesForNonAdmin(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Creator Visibility Site",
		AllowedOrigin: "https://creator-visibility.example",
		OwnerEmail:    testAdminEmailAddress,
		CreatorEmail:  testCreatorEmail,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/sites", nil)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testCreatorEmail, Role: httpapi.RoleUser})

	harness.handlers.ListSites(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody struct {
		Sites []struct {
			Identifier string `json:"id"`
		} `json:"sites"`
	}
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Len(testingT, responseBody.Sites, 1)
	require.Equal(testingT, site.ID, responseBody.Sites[0].Identifier)
}

func TestListSitesIncludesAllSitesForAdmin(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	adminSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Admin Visibility Anchor",
		AllowedOrigin: "https://admin-visibility.example",
		OwnerEmail:    testAdminEmailAddress,
		CreatorEmail:  testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&adminSite).Error)

	userManagedSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Admin Should See",
		AllowedOrigin: "https://admin-sees-user.example",
		OwnerEmail:    testAlternateOwnerEmailAddress,
		CreatorEmail:  testUserEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&userManagedSite).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/sites", nil)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

	harness.handlers.ListSites(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody struct {
		Sites []struct {
			Identifier string `json:"id"`
		} `json:"sites"`
	}
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))

	returnedSites := make(map[string]struct{}, len(responseBody.Sites))
	for _, site := range responseBody.Sites {
		returnedSites[site.Identifier] = struct{}{}
	}
	_, adminSiteFound := returnedSites[adminSite.ID]
	_, userManagedSiteFound := returnedSites[userManagedSite.ID]
	require.True(testingT, adminSiteFound)
	require.True(testingT, userManagedSiteFound)
}

func TestListSitesExcludesForeignSitesForNonAdmin(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	managedSite := model.Site{
		ID:            storage.NewID(),
		Name:          "User Managed Site",
		AllowedOrigin: "https://user-managed.example",
		OwnerEmail:    testAlternateOwnerEmailAddress,
		CreatorEmail:  testUserEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&managedSite).Error)

	foreignSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Foreign Visibility Site",
		AllowedOrigin: "https://foreign-visibility.example",
		OwnerEmail:    testAdminEmailAddress,
		CreatorEmail:  testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&foreignSite).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/sites", nil)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, Role: httpapi.RoleUser})

	harness.handlers.ListSites(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody struct {
		Sites []struct {
			Identifier string `json:"id"`
		} `json:"sites"`
	}
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))

	require.Len(testingT, responseBody.Sites, 1)
	require.Equal(testingT, managedSite.ID, responseBody.Sites[0].Identifier)
}

func TestSiteFaviconReturnsStoredIcon(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:                 storage.NewID(),
		Name:               "Icon Site",
		AllowedOrigin:      "https://icon.example",
		OwnerEmail:         testAdminEmailAddress,
		FaviconData:        []byte{0x10, 0x20, 0x30},
		FaviconContentType: "image/png",
		FaviconOrigin:      "https://icon.example",
		FaviconFetchedAt:   time.Now(),
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/sites/"+site.ID+"/favicon", nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

	harness.handlers.SiteFavicon(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Equal(testingT, "image/png", recorder.Header().Get("Content-Type"))
	require.Equal(testingT, []byte{0x10, 0x20, 0x30}, recorder.Body.Bytes())
}

type streamStubFaviconResolver struct {
	asset *favicon.Asset
	mu    sync.Mutex
	calls int
}

func (resolver *streamStubFaviconResolver) Resolve(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (resolver *streamStubFaviconResolver) ResolveAsset(_ context.Context, _ string) (*favicon.Asset, error) {
	resolver.mu.Lock()
	resolver.calls++
	resolver.mu.Unlock()
	return resolver.asset, nil
}

func (resolver *streamStubFaviconResolver) callCount() int {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	return resolver.calls
}

func TestStreamFaviconUpdatesEmitsEvents(testingT *testing.T) {
	testingT.Helper()

	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Streamed Site",
		AllowedOrigin: "https://stream.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, database.Create(&site).Error)

	resolver := &streamStubFaviconResolver{asset: &favicon.Asset{ContentType: "image/png", Data: []byte{0x05}}}
	service := favicon.NewService(resolver)
	faviconManager := httpapi.NewSiteFaviconManager(
		database,
		service,
		zap.NewNop(),
		httpapi.WithFaviconIntervals(5*time.Millisecond, 5*time.Millisecond),
	)
	faviconManager.Start(context.Background())
	testingT.Cleanup(faviconManager.Stop)

	handlers := httpapi.NewSiteHandlers(database, zap.NewNop(), testWidgetBaseURL, faviconManager, nil, nil)

	engine := gin.New()
	engine.GET("/stream", func(context *gin.Context) {
		context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})
		handlers.StreamFaviconUpdates(context)
	})

	server := httptest.NewServer(engine)
	testingT.Cleanup(server.Close)

	client := server.Client()
	request, err := http.NewRequest(http.MethodGet, server.URL+"/stream", nil)
	require.NoError(testingT, err)
	response, err := client.Do(request)
	require.NoError(testingT, err)
	testingT.Cleanup(func() {
		_ = response.Body.Close()
	})
	require.Equal(testingT, "text/event-stream", response.Header.Get("Content-Type"))

	events := make(chan struct {
		name string
		data string
	}, 1)
	go func() {
		reader := bufio.NewReader(response.Body)
		var eventName string
		var dataPayload string
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				close(events)
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(line, "event: ") {
				eventName = strings.TrimPrefix(line, "event: ")
				continue
			}
			if strings.HasPrefix(line, "data: ") {
				dataPayload = strings.TrimPrefix(line, "data: ")
				continue
			}
			if line == "" && eventName != "" && dataPayload != "" {
				events <- struct {
					name string
					data string
				}{name: eventName, data: dataPayload}
				return
			}
		}
	}()

	faviconManager.ScheduleFetch(site)

	require.Eventually(testingT, func() bool {
		return resolver.callCount() > 0
	}, 2*time.Second, 10*time.Millisecond)

	select {
	case payload, ok := <-events:
		require.True(testingT, ok)
		require.Equal(testingT, "favicon_updated", payload.name)
		require.Contains(testingT, payload.data, site.ID)
		require.Contains(testingT, payload.data, "favicon_url")
	case <-time.After(5 * time.Second):
		testingT.Fatal("timed out waiting for SSE event")
	}
}

func TestStreamFeedbackUpdatesReceivesCreateEvents(testingT *testing.T) {
	testingT.Helper()

	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Feedback Stream Site",
		AllowedOrigin: "",
		OwnerEmail:    testAdminEmailAddress,
		CreatorEmail:  testAdminEmailAddress,
	}
	require.NoError(testingT, database.Create(&site).Error)

	feedbackBroadcaster := httpapi.NewFeedbackEventBroadcaster()
	testingT.Cleanup(feedbackBroadcaster.Close)
	siteHandlers := httpapi.NewSiteHandlers(database, zap.NewNop(), testWidgetBaseURL, nil, nil, feedbackBroadcaster)
	publicHandlers := httpapi.NewPublicHandlers(database, zap.NewNop(), feedbackBroadcaster, nil, nil, true)

	engine := gin.New()
	engine.GET("/stream", func(context *gin.Context) {
		context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})
		siteHandlers.StreamFeedbackUpdates(context)
	})
	engine.POST("/api/feedback", func(context *gin.Context) {
		publicHandlers.CreateFeedback(context)
	})

	server := httptest.NewServer(engine)
	testingT.Cleanup(server.Close)

	client := server.Client()

	request, err := http.NewRequest(http.MethodGet, server.URL+"/stream", nil)
	require.NoError(testingT, err)

	response, err := client.Do(request)
	require.NoError(testingT, err)
	testingT.Cleanup(func() {
		_ = response.Body.Close()
	})
	require.Equal(testingT, "text/event-stream", response.Header.Get("Content-Type"))

	type eventPayload struct {
		name string
		data string
	}
	events := make(chan eventPayload, 1)
	go func() {
		reader := bufio.NewReader(response.Body)
		var eventName string
		var dataPayload string
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				close(events)
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(line, "event: ") {
				eventName = strings.TrimPrefix(line, "event: ")
				continue
			}
			if strings.HasPrefix(line, "data: ") {
				dataPayload = strings.TrimPrefix(line, "data: ")
				continue
			}
			if line == "" && eventName != "" && dataPayload != "" {
				events <- eventPayload{name: eventName, data: dataPayload}
				return
			}
		}
	}()

	feedbackRequestBody := bytes.NewBufferString(fmt.Sprintf(`{"site_id":"%s","contact":"person@example.com","message":"Hello"}`, site.ID))
	createRequest, err := http.NewRequest(http.MethodPost, server.URL+"/api/feedback", feedbackRequestBody)
	require.NoError(testingT, err)
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse, err := client.Do(createRequest)
	require.NoError(testingT, err)
	require.Equal(testingT, http.StatusOK, createResponse.StatusCode)
	testingT.Cleanup(func() {
		_ = createResponse.Body.Close()
	})

	select {
	case payload, ok := <-events:
		require.True(testingT, ok)
		require.Equal(testingT, "feedback_created", payload.name)
		var decoded map[string]any
		require.NoError(testingT, json.Unmarshal([]byte(payload.data), &decoded))
		require.Equal(testingT, site.ID, decoded["site_id"])
		require.Greater(testingT, decoded["created_at"], float64(0))
		require.Equal(testingT, float64(1), decoded["feedback_count"])
	case <-time.After(5 * time.Second):
		testingT.Fatal("timed out waiting for feedback SSE event")
	}
}

func TestNonAdminCannotAccessForeignSite(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Foreign Site",
		AllowedOrigin: "http://foreign.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/sites/"+site.ID+"/messages", nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, Role: httpapi.RoleUser})

	harness.handlers.ListMessagesBySite(context)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)
}

func TestCreateSiteAllowsAdminToSpecifyOwner(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	payload := map[string]string{
		"name":           "Admin Created",
		"allowed_origin": "http://owned.example",
		"owner_email":    testUserEmailAddress,
	}

	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody map[string]any
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, "Admin Created", responseBody["name"])
	require.Equal(testingT, "http://owned.example", responseBody["allowed_origin"])
	require.Equal(testingT, testUserEmailAddress, responseBody["owner_email"])
	require.Equal(testingT, float64(0), responseBody["feedback_count"])
	require.Equal(testingT, defaultWidgetTestBubbleSide, responseBody["widget_bubble_side"])
	require.Equal(testingT, float64(defaultWidgetTestBottomOffsetPixels), responseBody["widget_bubble_bottom_offset"])

	var createdSite model.Site
	require.NoError(testingT, harness.database.First(&createdSite, "name = ?", "Admin Created").Error)
	require.Equal(testingT, testUserEmailAddress, createdSite.OwnerEmail)
	require.Equal(testingT, testAdminEmailAddress, createdSite.CreatorEmail)
	require.Equal(testingT, defaultWidgetTestBubbleSide, createdSite.WidgetBubbleSide)
	require.Equal(testingT, defaultWidgetTestBottomOffsetPixels, createdSite.WidgetBubbleBottomOffsetPx)
}

func TestCreateSiteAssignsCurrentUserAsOwner(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	payload := map[string]string{
		"name":           "Self Owned",
		"allowed_origin": "http://self.example",
	}

	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, Role: httpapi.RoleUser})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody map[string]any
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, testUserEmailAddress, responseBody["owner_email"])
	require.Equal(testingT, float64(0), responseBody["feedback_count"])
	require.Equal(testingT, defaultWidgetTestBubbleSide, responseBody["widget_bubble_side"])
	require.Equal(testingT, float64(defaultWidgetTestBottomOffsetPixels), responseBody["widget_bubble_bottom_offset"])

	var createdSite model.Site
	require.NoError(testingT, harness.database.First(&createdSite, "name = ?", "Self Owned").Error)
	require.Equal(testingT, testUserEmailAddress, createdSite.OwnerEmail)
	require.Equal(testingT, testUserEmailAddress, createdSite.CreatorEmail)
	require.Equal(testingT, defaultWidgetTestBubbleSide, createdSite.WidgetBubbleSide)
	require.Equal(testingT, defaultWidgetTestBottomOffsetPixels, createdSite.WidgetBubbleBottomOffsetPx)
}

func TestCreateSiteAllowsNonAdminToAssignAlternateOwner(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	payload := map[string]string{
		"name":           "Delegated Ownership",
		"allowed_origin": "http://delegated.example",
		"owner_email":    testAlternateOwnerEmailAddress,
	}

	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, Role: httpapi.RoleUser})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody map[string]any
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, testAlternateOwnerEmailAddress, responseBody["owner_email"])
	require.Equal(testingT, defaultWidgetTestBubbleSide, responseBody["widget_bubble_side"])
	require.Equal(testingT, float64(defaultWidgetTestBottomOffsetPixels), responseBody["widget_bubble_bottom_offset"])

	var createdSite model.Site
	require.NoError(testingT, harness.database.First(&createdSite, "name = ?", "Delegated Ownership").Error)
	require.Equal(testingT, testAlternateOwnerEmailAddress, createdSite.OwnerEmail)
	require.Equal(testingT, testUserEmailAddress, createdSite.CreatorEmail)
	require.Equal(testingT, defaultWidgetTestBubbleSide, createdSite.WidgetBubbleSide)
	require.Equal(testingT, defaultWidgetTestBottomOffsetPixels, createdSite.WidgetBubbleBottomOffsetPx)
}

func TestCreateSiteRejectsDuplicateAllowedOrigin(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	existingSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Existing Site",
		AllowedOrigin: "https://duplicate.example",
		OwnerEmail:    testAdminEmailAddress,
		CreatorEmail:  testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&existingSite).Error)

	payload := map[string]string{
		"name":           "Duplicate Site",
		"allowed_origin": "https://duplicate.example",
		"owner_email":    testAdminEmailAddress,
	}

	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusConflict, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeSiteExists, responseBody[jsonErrorKey])
}

func TestCreateSiteAcceptsWidgetPlacementOverrides(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	payload := map[string]any{
		"name":                        "Widget Placement",
		"allowed_origin":              "http://widget-placement.example",
		"widget_bubble_side":          customWidgetTestBubbleSide,
		"widget_bubble_bottom_offset": customWidgetTestBottomOffsetPixels,
	}

	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody map[string]any
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, customWidgetTestBubbleSide, responseBody["widget_bubble_side"])
	require.Equal(testingT, float64(customWidgetTestBottomOffsetPixels), responseBody["widget_bubble_bottom_offset"])

	var createdSite model.Site
	require.NoError(testingT, harness.database.First(&createdSite, "name = ?", "Widget Placement").Error)
	require.Equal(testingT, customWidgetTestBubbleSide, createdSite.WidgetBubbleSide)
	require.Equal(testingT, customWidgetTestBottomOffsetPixels, createdSite.WidgetBubbleBottomOffsetPx)
}

func TestCreateSiteRejectsInvalidWidgetPlacement(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	payload := map[string]any{
		"name":                        "Invalid Placement",
		"allowed_origin":              "http://invalid-placement.example",
		"widget_bubble_side":          customWidgetTestBubbleSide,
		"widget_bubble_bottom_offset": -12,
	}

	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeInvalidWidgetOffset, responseBody[jsonErrorKey])
}

func TestCreateSiteRejectsInvalidWidgetSide(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	payload := map[string]any{
		"name":                        "Invalid Side",
		"allowed_origin":              "http://invalid-side.example",
		"widget_bubble_side":          "center",
		"widget_bubble_bottom_offset": customWidgetTestBottomOffsetPixels,
	}

	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeInvalidWidgetSide, responseBody[jsonErrorKey])
}

func TestUpdateSiteRejectsDuplicateAllowedOrigin(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	targetSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Primary Site",
		AllowedOrigin: "https://primary.example",
		OwnerEmail:    testAdminEmailAddress,
		CreatorEmail:  testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&targetSite).Error)

	conflictingSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Existing Site",
		AllowedOrigin: "https://conflict.example",
		OwnerEmail:    testAdminEmailAddress,
		CreatorEmail:  testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&conflictingSite).Error)

	payload := map[string]string{
		"allowed_origin": "https://conflict.example",
	}

	recorder, context := newJSONContext(http.MethodPatch, "/api/sites/"+targetSite.ID, payload)
	context.Params = gin.Params{{Key: "id", Value: targetSite.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

	harness.handlers.UpdateSite(context)
	require.Equal(testingT, http.StatusConflict, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeSiteExists, responseBody[jsonErrorKey])
}

func TestUpdateSiteAllowsOwnerToChangeDetails(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Owner Site",
		AllowedOrigin: "http://owner.example",
		OwnerEmail:    testUserEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	payload := map[string]string{
		"name":           "Updated Name",
		"allowed_origin": "http://updated.example",
	}

	recorder, context := newJSONContext(http.MethodPatch, "/api/sites/"+site.ID, payload)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, Role: httpapi.RoleUser})

	harness.handlers.UpdateSite(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody map[string]any
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, "Updated Name", responseBody["name"])
	require.Equal(testingT, "http://updated.example", responseBody["allowed_origin"])
}

func TestUpdateSiteAllowsOwnerToReassignOwnership(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Reassignable Site",
		AllowedOrigin:              "http://reassign.example",
		OwnerEmail:                 testUserEmailAddress,
		WidgetBubbleSide:           defaultWidgetTestBubbleSide,
		WidgetBubbleBottomOffsetPx: defaultWidgetTestBottomOffsetPixels,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	newOwnerEmail := "new-owner@example.com"
	payload := map[string]string{
		"name":           "Reassignable Site",
		"allowed_origin": "http://reassign.example",
		"owner_email":    newOwnerEmail,
	}

	recorder, context := newJSONContext(http.MethodPatch, "/api/sites/"+site.ID, payload)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, Role: httpapi.RoleUser})

	harness.handlers.UpdateSite(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody map[string]any
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, newOwnerEmail, responseBody["owner_email"])

	var updatedSite model.Site
	require.NoError(testingT, harness.database.First(&updatedSite, "id = ?", site.ID).Error)
	require.Equal(testingT, newOwnerEmail, updatedSite.OwnerEmail)
}

func TestUpdateSiteAdjustsWidgetPlacement(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Placement Update",
		AllowedOrigin:              "http://placement-update.example",
		OwnerEmail:                 testUserEmailAddress,
		WidgetBubbleSide:           defaultWidgetTestBubbleSide,
		WidgetBubbleBottomOffsetPx: defaultWidgetTestBottomOffsetPixels,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	payload := map[string]any{
		"widget_bubble_side":          customWidgetTestBubbleSide,
		"widget_bubble_bottom_offset": customWidgetTestBottomOffsetPixels,
	}

	recorder, context := newJSONContext(http.MethodPatch, "/api/sites/"+site.ID, payload)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, Role: httpapi.RoleUser})

	harness.handlers.UpdateSite(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody map[string]any
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, customWidgetTestBubbleSide, responseBody["widget_bubble_side"])
	require.Equal(testingT, float64(customWidgetTestBottomOffsetPixels), responseBody["widget_bubble_bottom_offset"])

	var updatedSite model.Site
	require.NoError(testingT, harness.database.First(&updatedSite, "id = ?", site.ID).Error)
	require.Equal(testingT, customWidgetTestBubbleSide, updatedSite.WidgetBubbleSide)
	require.Equal(testingT, customWidgetTestBottomOffsetPixels, updatedSite.WidgetBubbleBottomOffsetPx)
}

func TestUpdateSiteRejectsInvalidWidgetPlacement(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Placement Invalid Update",
		AllowedOrigin:              "http://placement-invalid.example",
		OwnerEmail:                 testUserEmailAddress,
		WidgetBubbleSide:           defaultWidgetTestBubbleSide,
		WidgetBubbleBottomOffsetPx: defaultWidgetTestBottomOffsetPixels,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	payload := map[string]any{
		"widget_bubble_side":          customWidgetTestBubbleSide,
		"widget_bubble_bottom_offset": 1000,
	}

	recorder, context := newJSONContext(http.MethodPatch, "/api/sites/"+site.ID, payload)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, Role: httpapi.RoleUser})

	harness.handlers.UpdateSite(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeInvalidWidgetOffset, responseBody[jsonErrorKey])
}

func TestUpdateSiteRejectsInvalidWidgetSide(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Placement Invalid Side",
		AllowedOrigin: "http://placement-side-invalid.example",
		OwnerEmail:    testUserEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	payload := map[string]any{
		"widget_bubble_side":          "center",
		"widget_bubble_bottom_offset": customWidgetTestBottomOffsetPixels,
	}

	recorder, context := newJSONContext(http.MethodPatch, "/api/sites/"+site.ID, payload)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, Role: httpapi.RoleUser})

	harness.handlers.UpdateSite(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeInvalidWidgetSide, responseBody[jsonErrorKey])
}

func TestDeleteSiteAllowsCreatorWithAlternateOwner(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Creator Managed Site",
		AllowedOrigin: "http://creator-managed.example",
		OwnerEmail:    testAlternateOwnerEmailAddress,
		CreatorEmail:  testUserEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	recorder, context := newJSONContext(http.MethodDelete, "/api/sites/"+site.ID, nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, Role: httpapi.RoleUser})

	harness.handlers.DeleteSite(context)
	require.Equal(testingT, http.StatusNoContent, recorder.Code)

	var remainingSite model.Site
	require.ErrorIs(testingT, harness.database.First(&remainingSite, "id = ?", site.ID).Error, gorm.ErrRecordNotFound)
}

func TestDeleteSiteRemovesSiteAndFeedback(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Deletable Site",
		AllowedOrigin: "http://delete.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	feedback := model.Feedback{
		ID:      storage.NewID(),
		SiteID:  site.ID,
		Contact: "contact@example.com",
		Message: "Message",
	}
	require.NoError(testingT, harness.database.Create(&feedback).Error)

	recorder, context := newJSONContext(http.MethodDelete, "/api/sites/"+site.ID, nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

	harness.handlers.DeleteSite(context)
	require.Equal(testingT, http.StatusNoContent, recorder.Code)

	var remainingSite model.Site
	require.ErrorIs(testingT, harness.database.First(&remainingSite, "id = ?", site.ID).Error, gorm.ErrRecordNotFound)

	var remainingFeedback model.Feedback
	require.ErrorIs(testingT, harness.database.First(&remainingFeedback, "id = ?", feedback.ID).Error, gorm.ErrRecordNotFound)
}

func TestDeleteSitePreventsUnauthorizedUser(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Protected Site",
		AllowedOrigin: "http://protected.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	recorder, context := newJSONContext(http.MethodDelete, "/api/sites/"+site.ID, nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, Role: httpapi.RoleUser})

	harness.handlers.DeleteSite(context)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)

	var persistedSite model.Site
	require.NoError(testingT, harness.database.First(&persistedSite, "id = ?", site.ID).Error)
}

func TestUserAvatarReturnsStoredImage(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	user := model.User{
		Email:             strings.ToLower(testUserEmailAddress),
		Name:              "Test User",
		AvatarContentType: "image/png",
		AvatarData:        []byte{0x01, 0x02, 0x03},
	}
	require.NoError(testingT, harness.database.Save(&user).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/me/avatar", nil)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress})

	harness.handlers.UserAvatar(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Equal(testingT, "image/png", recorder.Header().Get("Content-Type"))
	require.Equal(testingT, []byte{0x01, 0x02, 0x03}, recorder.Body.Bytes())
}

func TestUserAvatarReturnsNotFoundWhenMissing(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	recorder, context := newJSONContext(http.MethodGet, "/api/me/avatar", nil)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress})

	harness.handlers.UserAvatar(context)
	require.Equal(testingT, http.StatusNotFound, recorder.Code)
}

func TestListSubscribersReturnsDataForAdmin(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, err)
	require.NoError(testingT, storage.AutoMigrate(database))

	router := gin.New()
	router.Use(gin.Recovery())
	feedbackBroadcaster := httpapi.NewFeedbackEventBroadcaster()
	siteHandlers := httpapi.NewSiteHandlers(database, zap.NewNop(), testWidgetBaseURL, nil, nil, feedbackBroadcaster)
	router.GET("/api/sites/:id/subscribers", func(context *gin.Context) {
		context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})
		siteHandlers.ListSubscribers(context)
	})

	site := model.Site{ID: storage.NewID(), Name: "Subs", AllowedOrigin: "http://example.com", OwnerEmail: testAdminEmailAddress, CreatorEmail: testAdminEmailAddress}
	require.NoError(testingT, database.Create(&site).Error)
	subscriber, subErr := model.NewSubscriber(model.SubscriberInput{
		SiteID: site.ID,
		Email:  "user@example.com",
		Name:   "User",
	})
	require.NoError(testingT, subErr)
	require.NoError(testingT, database.Create(&subscriber).Error)

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/sites/%s/subscribers", site.ID), nil)
	require.NoError(testingT, err)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var response httpapi.SiteSubscribersResponse
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(testingT, response.Subscribers, 1)
	require.Equal(testingT, subscriber.Email, response.Subscribers[0].Email)
}

func TestExportSubscribersReturnsCSV(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, err)
	require.NoError(testingT, storage.AutoMigrate(database))

	router := gin.New()
	router.Use(gin.Recovery())
	feedbackBroadcaster := httpapi.NewFeedbackEventBroadcaster()
	siteHandlers := httpapi.NewSiteHandlers(database, zap.NewNop(), testWidgetBaseURL, nil, nil, feedbackBroadcaster)
	router.GET("/api/sites/:id/subscribers/export", func(context *gin.Context) {
		context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})
		siteHandlers.ExportSubscribers(context)
	})

	site := model.Site{ID: storage.NewID(), Name: "Subs", AllowedOrigin: "http://example.com", OwnerEmail: testAdminEmailAddress, CreatorEmail: testAdminEmailAddress}
	require.NoError(testingT, database.Create(&site).Error)
	subscriber, subErr := model.NewSubscriber(model.SubscriberInput{
		SiteID: site.ID,
		Email:  "csv@example.com",
	})
	require.NoError(testingT, subErr)
	require.NoError(testingT, database.Create(&subscriber).Error)

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/sites/%s/subscribers/export", site.ID), nil)
	require.NoError(testingT, err)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), "csv@example.com")
	require.Contains(testingT, recorder.Header().Get("Content-Type"), "text/csv")
}

func TestUpdateSubscriberStatus(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, err)
	require.NoError(testingT, storage.AutoMigrate(database))

	router := gin.New()
	router.Use(gin.Recovery())
	feedbackBroadcaster := httpapi.NewFeedbackEventBroadcaster()
	siteHandlers := httpapi.NewSiteHandlers(database, zap.NewNop(), testWidgetBaseURL, nil, nil, feedbackBroadcaster)
	router.PATCH("/api/sites/:id/subscribers/:subscriber_id", func(context *gin.Context) {
		context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})
		siteHandlers.UpdateSubscriberStatus(context)
	})

	site := model.Site{ID: storage.NewID(), Name: "Subs", AllowedOrigin: "http://example.com", OwnerEmail: testAdminEmailAddress, CreatorEmail: testAdminEmailAddress}
	require.NoError(testingT, database.Create(&site).Error)
	subscriber, subErr := model.NewSubscriber(model.SubscriberInput{
		SiteID: site.ID,
		Email:  "status@example.com",
	})
	require.NoError(testingT, subErr)
	require.NoError(testingT, database.Create(&subscriber).Error)

	body := bytes.NewBufferString(`{"status":"unsubscribed"}`)
	request, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/sites/%s/subscribers/%s", site.ID, subscriber.ID), body)
	require.NoError(testingT, err)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var refreshed model.Subscriber
	require.NoError(testingT, database.First(&refreshed, "id = ?", subscriber.ID).Error)
	require.Equal(testingT, model.SubscriberStatusUnsubscribed, refreshed.Status)
}

func newJSONContext(method string, path string, body any) (*httptest.ResponseRecorder, *gin.Context) {
	recorder := httptest.NewRecorder()
	var requestBody *bytes.Reader
	if body != nil {
		encoded, _ := json.Marshal(body)
		requestBody = bytes.NewReader(encoded)
	} else {
		requestBody = bytes.NewReader(nil)
	}

	request := httptest.NewRequest(method, path, requestBody)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	return recorder, context
}
