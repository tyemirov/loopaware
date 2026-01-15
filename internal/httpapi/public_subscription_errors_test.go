package httpapi_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
)

const (
	testSubscriptionSiteName    = "Subscription Errors"
	testSubscriptionSiteOrigin  = "http://subscribe-errors.example"
	testSubscriptionOwnerEmail  = "owner@subscribe-errors.example"
	testSubscriptionEmail       = "subscriber@subscribe-errors.example"
	testSubscriptionInvalidJSON = "{"
)

func TestCreateSubscriptionUpdatesUnsubscribedSubscriber(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, testSubscriptionSiteName, testSubscriptionSiteOrigin, testSubscriptionOwnerEmail)

	subscriber, subscriberErr := model.NewSubscriber(model.SubscriberInput{
		SiteID:         site.ID,
		Email:          testSubscriptionEmail,
		Status:         model.SubscriberStatusUnsubscribed,
		UnsubscribedAt: time.Now().UTC(),
	})
	require.NoError(testingT, subscriberErr)
	require.NoError(testingT, api.database.Create(&subscriber).Error)

	response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   testSubscriptionEmail,
		"name":    "Updated Name",
	}, map[string]string{"Origin": site.AllowedOrigin})
	require.Equal(testingT, http.StatusOK, response.Code)

	var updated model.Subscriber
	require.NoError(testingT, api.database.First(&updated, "id = ?", subscriber.ID).Error)
	require.Equal(testingT, model.SubscriberStatusPending, updated.Status)
	require.True(testingT, updated.UnsubscribedAt.IsZero())
	require.Equal(testingT, "Updated Name", updated.Name)
}

func TestCreateSubscriptionRejectsInvalidJSON(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)

	request := httptest.NewRequest(http.MethodPost, "/api/subscriptions", strings.NewReader(testSubscriptionInvalidJSON))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	api.router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestCreateSubscriptionRejectsUnknownSite(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)

	response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": storage.NewID(),
		"email":   testSubscriptionEmail,
	}, map[string]string{"Origin": testSubscriptionSiteOrigin})
	require.Equal(testingT, http.StatusNotFound, response.Code)
}

func TestCreateSubscriptionRateLimited(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, testSubscriptionSiteName, testSubscriptionSiteOrigin, testSubscriptionOwnerEmail)

	headers := map[string]string{"Origin": site.AllowedOrigin}
	payload := map[string]any{"site_id": site.ID, "email": "rate@example.com"}

	tooMany := 0
	for attemptIndex := 0; attemptIndex < 12; attemptIndex++ {
		response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", payload, headers)
		if response.Code == http.StatusTooManyRequests {
			tooMany++
			break
		}
	}
	require.GreaterOrEqual(testingT, tooMany, 1)
}

func TestUpdateSubscriptionStatusRejectsInvalidJSON(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)

	request := httptest.NewRequest(http.MethodPost, "/api/subscriptions/confirm", strings.NewReader(testSubscriptionInvalidJSON))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	api.router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestUpdateSubscriptionStatusRejectsMissingFields(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)

	response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": "",
		"email":   "",
	}, map[string]string{"Origin": testSubscriptionSiteOrigin})
	require.Equal(testingT, http.StatusBadRequest, response.Code)
}

func TestUpdateSubscriptionStatusRejectsUnknownSite(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)

	response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": storage.NewID(),
		"email":   testSubscriptionEmail,
	}, map[string]string{"Origin": testSubscriptionSiteOrigin})
	require.Equal(testingT, http.StatusNotFound, response.Code)
}

func TestUpdateSubscriptionStatusRejectsOriginForbidden(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, testSubscriptionSiteName, testSubscriptionSiteOrigin, testSubscriptionOwnerEmail)

	response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   testSubscriptionEmail,
	}, map[string]string{"Origin": "http://blocked.example"})
	require.Equal(testingT, http.StatusForbidden, response.Code)
}

func TestUpdateSubscriptionStatusReturnsUnknownSubscription(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, testSubscriptionSiteName, testSubscriptionSiteOrigin, testSubscriptionOwnerEmail)

	response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   testSubscriptionEmail,
	}, map[string]string{"Origin": site.AllowedOrigin})
	require.Equal(testingT, http.StatusNotFound, response.Code)
}

func TestUpdateSubscriptionStatusReturnsOkForSameStatus(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, testSubscriptionSiteName, testSubscriptionSiteOrigin, testSubscriptionOwnerEmail)

	subscriber, subscriberErr := model.NewSubscriber(model.SubscriberInput{
		SiteID: site.ID,
		Email:  testSubscriptionEmail,
		Status: model.SubscriberStatusConfirmed,
	})
	require.NoError(testingT, subscriberErr)
	require.NoError(testingT, api.database.Create(&subscriber).Error)

	response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions/confirm", map[string]any{
		"site_id": site.ID,
		"email":   testSubscriptionEmail,
	}, map[string]string{"Origin": site.AllowedOrigin})
	require.Equal(testingT, http.StatusOK, response.Code)
}

func TestCreateSubscriptionReportsSaveError(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, testSubscriptionSiteName, testSubscriptionSiteOrigin, testSubscriptionOwnerEmail)

	registerErr := api.database.Callback().Create().Before("gorm:create").Register("force_subscriber_error", func(callbackDatabase *gorm.DB) {
		callbackDatabase.AddError(errors.New("forced create error"))
	})
	require.NoError(testingT, registerErr)
	testingT.Cleanup(func() {
		_ = api.database.Callback().Create().Remove("force_subscriber_error")
	})

	response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   testSubscriptionEmail,
	}, map[string]string{"Origin": site.AllowedOrigin})
	require.Equal(testingT, http.StatusInternalServerError, response.Code)
}
