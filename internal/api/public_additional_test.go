package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	testDemoWidgetSiteID       = "__loopaware_widget_demo__"
	testDemoWidgetBubbleSide   = "left"
	testDemoWidgetBubbleOffset = 16
	testSubscriberTableName    = "subscribers"
	testSubscriberQueryHook    = "force_subscriber_query_error"
	testSubscriberUpdateHook   = "force_subscriber_update_error"
)

func TestCreateFeedbackReturnsNotFoundWhenSiteLookupFails(testingT *testing.T) {
	apiHarness := buildAPIHarness(testingT, nil, nil, nil)
	sqlDatabase, sqlErr := apiHarness.database.DB()
	require.NoError(testingT, sqlErr)
	require.NoError(testingT, sqlDatabase.Close())

	response := performJSONRequest(testingT, apiHarness.router, http.MethodPost, "/public/feedback", map[string]any{
		"site_id": "missing-site",
		"contact": "user@example.com",
		"message": "Feedback",
	}, map[string]string{"Origin": "http://example.com"})

	require.Equal(testingT, http.StatusNotFound, response.Code)
}

func TestWidgetConfigReturnsDemoDefaults(testingT *testing.T) {
	apiHarness := buildAPIHarness(testingT, nil, nil, nil)
	response := performJSONRequest(testingT, apiHarness.router, http.MethodGet, "/public/widget-config?site_id="+testDemoWidgetSiteID, nil, nil)
	require.Equal(testingT, http.StatusOK, response.Code)

	var payload struct {
		SiteID                   string `json:"site_id"`
		WidgetBubbleSide         string `json:"widget_bubble_side"`
		WidgetBubbleBottomOffset int    `json:"widget_bubble_bottom_offset"`
	}
	require.NoError(testingT, json.Unmarshal(response.Body.Bytes(), &payload))
	require.Equal(testingT, testDemoWidgetSiteID, payload.SiteID)
	require.Equal(testingT, testDemoWidgetBubbleSide, payload.WidgetBubbleSide)
	require.Equal(testingT, testDemoWidgetBubbleOffset, payload.WidgetBubbleBottomOffset)
}

func TestCreateSubscriptionReturnsServerErrorWhenSubscriberLookupFails(testingT *testing.T) {
	apiHarness := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, apiHarness.database, testSubscribeSiteName, testSubscribeSiteOrigin, testSubscribeOwnerAddress)

	registerErr := apiHarness.database.Callback().Query().Before("gorm:query").Register(testSubscriberQueryHook, func(callbackDatabase *gorm.DB) {
		if callbackDatabase.Statement != nil && callbackDatabase.Statement.Table == testSubscriberTableName {
			callbackDatabase.AddError(errors.New("forced subscriber query error"))
		}
	})
	require.NoError(testingT, registerErr)
	testingT.Cleanup(func() {
		_ = apiHarness.database.Callback().Query().Remove(testSubscriberQueryHook)
	})

	response := performJSONRequest(testingT, apiHarness.router, http.MethodPost, "/public/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   testSubscriptionUpdateEmail,
		"name":    testSubscriptionUpdateName,
	}, map[string]string{"Origin": testSubscribeSiteOrigin})

	require.Equal(testingT, http.StatusInternalServerError, response.Code)
}

func TestCreateSubscriptionReturnsServerErrorWhenUpdateFails(testingT *testing.T) {
	apiHarness := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, apiHarness.database, testSubscribeSiteName, testSubscribeSiteOrigin, testSubscribeOwnerAddress)

	subscriber, subscriberErr := model.NewSubscriber(model.SubscriberInput{
		SiteID:    site.ID,
		Email:     testSubscriptionUpdateEmail,
		Name:      testSubscriptionUpdateName,
		Status:    model.SubscriberStatusUnsubscribed,
		ConsentAt: time.Now().UTC(),
	})
	require.NoError(testingT, subscriberErr)
	require.NoError(testingT, apiHarness.database.Create(&subscriber).Error)

	registerErr := apiHarness.database.Callback().Update().Before("gorm:update").Register(testSubscriberUpdateHook, func(callbackDatabase *gorm.DB) {
		if callbackDatabase.Statement != nil && callbackDatabase.Statement.Table == testSubscriberTableName {
			callbackDatabase.AddError(errors.New("forced subscriber update error"))
		}
	})
	require.NoError(testingT, registerErr)
	testingT.Cleanup(func() {
		_ = apiHarness.database.Callback().Update().Remove(testSubscriberUpdateHook)
	})

	response := performJSONRequest(testingT, apiHarness.router, http.MethodPost, "/public/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   subscriber.Email,
		"name":    testSubscriptionUpdateName,
	}, map[string]string{"Origin": testSubscribeSiteOrigin})

	require.Equal(testingT, http.StatusInternalServerError, response.Code)
}

func TestCreateSubscriptionRejectsInvalidContact(testingT *testing.T) {
	apiHarness := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, apiHarness.database, testSubscribeSiteName, testSubscribeSiteOrigin, testSubscribeOwnerAddress)

	longSubscriberName := strings.Repeat("a", 201)
	response := performJSONRequest(testingT, apiHarness.router, http.MethodPost, "/public/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   testSubscriptionUpdateEmail,
		"name":    longSubscriberName,
	}, map[string]string{"Origin": testSubscribeSiteOrigin})

	require.Equal(testingT, http.StatusBadRequest, response.Code)
}
