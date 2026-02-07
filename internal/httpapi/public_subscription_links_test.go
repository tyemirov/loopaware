package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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
	testConfirmLinkSiteName                = "Confirm Links"
	testConfirmLinkSiteOrigin              = "http://links.example"
	testConfirmLinkOwnerEmail              = "owner@links.example"
	testConfirmLinkSubscriber              = "subscriber@links.example"
	testConfirmLinkAlternateEmail          = "alternate@links.example"
	testSubscriptionLinkUpdateHookName     = "gorm:update"
	testSubscriptionLinkUpdateCallbackName = "force_subscription_link_update_error"
	testSubscriptionLinkUpdateErrorMessage = "subscription_link_update_failed"
	testSubscriptionLinkUpdateTableName    = "subscribers"
)

type subscriptionLinkPayload struct {
	Heading        string `json:"heading"`
	Message        string `json:"message"`
	OpenURL        string `json:"open_url"`
	OpenLabel      string `json:"open_label"`
	UnsubscribeURL string `json:"unsubscribe_url"`
}

func decodeSubscriptionLinkPayload(testingT *testing.T, response *httptest.ResponseRecorder) subscriptionLinkPayload {
	testingT.Helper()
	var payload subscriptionLinkPayload
	require.NoError(testingT, json.Unmarshal(response.Body.Bytes(), &payload))
	return payload
}

func TestConfirmSubscriptionLinkRequiresToken(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	response := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/confirm-link", nil, nil)
	require.Equal(testingT, http.StatusBadRequest, response.Code)
	payload := decodeSubscriptionLinkPayload(testingT, response)
	require.Equal(testingT, "Subscription confirmation", payload.Heading)
	require.Equal(testingT, "Missing confirmation token.", payload.Message)
}

func TestConfirmSubscriptionLinkRejectsInvalidToken(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	response := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/confirm-link?token=bad", nil, nil)
	require.Equal(testingT, http.StatusBadRequest, response.Code)
	payload := decodeSubscriptionLinkPayload(testingT, response)
	require.Equal(testingT, "Subscription confirmation", payload.Heading)
	require.Equal(testingT, "Invalid or expired token.", payload.Message)
}

func TestConfirmSubscriptionLinkRequiresSecret(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	publicHandlers := httpapi.NewPublicHandlers(database, zap.NewNop(), nil, nil, nil, nil, true, "http://loopaware.test", "", nil, testLandingAuthConfig)
	router := gin.New()
	router.GET("/api/subscriptions/confirm-link", publicHandlers.ConfirmSubscriptionLinkJSON)

	request := httptest.NewRequest(http.MethodGet, "/api/subscriptions/confirm-link?token=token", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
	payload := decodeSubscriptionLinkPayload(testingT, recorder)
	require.Equal(testingT, "Subscription confirmation", payload.Heading)
	require.Equal(testingT, "Subscription confirmation is unavailable.", payload.Message)
}

func TestConfirmSubscriptionLinkRejectsMismatchedEmail(testingT *testing.T) {
	emailSender := &recordingEmailSender{testingT: testingT}
	api := buildAPIHarness(testingT, nil, nil, emailSender)
	site := insertSite(testingT, api.database, testConfirmLinkSiteName, testConfirmLinkSiteOrigin, testConfirmLinkOwnerEmail)

	token := createSubscriptionToken(testingT, api, site, testConfirmLinkSubscriber, emailSender)

	require.NoError(testingT, api.database.Model(&model.Subscriber{}).
		Where("site_id = ? AND email = ?", site.ID, strings.ToLower(testConfirmLinkSubscriber)).
		Update("email", testConfirmLinkAlternateEmail).Error)

	response := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/confirm-link?token="+url.QueryEscape(token), nil, nil)
	require.Equal(testingT, http.StatusBadRequest, response.Code)
	payload := decodeSubscriptionLinkPayload(testingT, response)
	require.Equal(testingT, "Subscription confirmation", payload.Heading)
	require.Equal(testingT, "Invalid or expired token.", payload.Message)
}

func TestConfirmSubscriptionLinkHandlesExistingStatuses(testingT *testing.T) {
	emailSender := &recordingEmailSender{testingT: testingT}
	api := buildAPIHarness(testingT, nil, nil, emailSender)
	site := insertSite(testingT, api.database, testConfirmLinkSiteName, testConfirmLinkSiteOrigin, testConfirmLinkOwnerEmail)

	token := createSubscriptionToken(testingT, api, site, testConfirmLinkSubscriber, emailSender)

	require.NoError(testingT, api.database.Model(&model.Subscriber{}).
		Where("site_id = ? AND email = ?", site.ID, strings.ToLower(testConfirmLinkSubscriber)).
		Update("status", model.SubscriberStatusUnsubscribed).Error)

	unsubscribed := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/confirm-link?token="+url.QueryEscape(token), nil, nil)
	require.Equal(testingT, http.StatusConflict, unsubscribed.Code)
	unsubscribedPayload := decodeSubscriptionLinkPayload(testingT, unsubscribed)
	require.Equal(testingT, "Subscription confirmation", unsubscribedPayload.Heading)
	require.Equal(testingT, "Subscription already unsubscribed.", unsubscribedPayload.Message)

	require.NoError(testingT, api.database.Model(&model.Subscriber{}).
		Where("site_id = ? AND email = ?", site.ID, strings.ToLower(testConfirmLinkSubscriber)).
		Update("status", model.SubscriberStatusConfirmed).Error)

	confirmed := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/confirm-link?token="+url.QueryEscape(token), nil, nil)
	require.Equal(testingT, http.StatusOK, confirmed.Code)
	confirmedPayload := decodeSubscriptionLinkPayload(testingT, confirmed)
	require.Equal(testingT, "Subscription confirmed", confirmedPayload.Heading)
	require.Contains(testingT, confirmedPayload.Message, "already confirmed")
}

func TestConfirmSubscriptionLinkReportsUpdateError(testingT *testing.T) {
	emailSender := &recordingEmailSender{testingT: testingT}
	api := buildAPIHarness(testingT, nil, nil, emailSender)
	site := insertSite(testingT, api.database, testConfirmLinkSiteName, testConfirmLinkSiteOrigin, testConfirmLinkOwnerEmail)

	token := createSubscriptionToken(testingT, api, site, testConfirmLinkSubscriber, emailSender)

	callbackName := testSubscriptionLinkUpdateCallbackName
	api.database.Callback().Update().Before(testSubscriptionLinkUpdateHookName).Register(callbackName, func(database *gorm.DB) {
		if database.Statement != nil && database.Statement.Table == testSubscriptionLinkUpdateTableName {
			database.AddError(errors.New(testSubscriptionLinkUpdateErrorMessage))
		}
	})
	testingT.Cleanup(func() {
		api.database.Callback().Update().Remove(callbackName)
	})

	response := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/confirm-link?token="+url.QueryEscape(token), nil, nil)
	require.Equal(testingT, http.StatusInternalServerError, response.Code)
	payload := decodeSubscriptionLinkPayload(testingT, response)
	require.Equal(testingT, "Subscription confirmation", payload.Heading)
	require.Equal(testingT, "Failed to confirm subscription.", payload.Message)
}

func TestUnsubscribeSubscriptionLinkRequiresToken(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	response := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/unsubscribe-link", nil, nil)
	require.Equal(testingT, http.StatusBadRequest, response.Code)
	payload := decodeSubscriptionLinkPayload(testingT, response)
	require.Equal(testingT, "Unsubscribe", payload.Heading)
	require.Equal(testingT, "Missing unsubscribe token.", payload.Message)
}

func TestUnsubscribeSubscriptionLinkRejectsInvalidToken(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	response := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/unsubscribe-link?token=bad", nil, nil)
	require.Equal(testingT, http.StatusBadRequest, response.Code)
	payload := decodeSubscriptionLinkPayload(testingT, response)
	require.Equal(testingT, "Unsubscribe", payload.Heading)
	require.Equal(testingT, "Invalid or expired token.", payload.Message)
}

func TestUnsubscribeSubscriptionLinkRequiresSecret(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	publicHandlers := httpapi.NewPublicHandlers(database, zap.NewNop(), nil, nil, nil, nil, true, "http://loopaware.test", "", nil, testLandingAuthConfig)
	router := gin.New()
	router.GET("/api/subscriptions/unsubscribe-link", publicHandlers.UnsubscribeSubscriptionLinkJSON)

	request := httptest.NewRequest(http.MethodGet, "/api/subscriptions/unsubscribe-link?token=token", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)
	payload := decodeSubscriptionLinkPayload(testingT, recorder)
	require.Equal(testingT, "Unsubscribe", payload.Heading)
	require.Equal(testingT, "Subscription unsubscribe is unavailable.", payload.Message)
}

func TestUnsubscribeSubscriptionLinkRejectsMismatchedEmail(testingT *testing.T) {
	emailSender := &recordingEmailSender{testingT: testingT}
	api := buildAPIHarness(testingT, nil, nil, emailSender)
	site := insertSite(testingT, api.database, testConfirmLinkSiteName, testConfirmLinkSiteOrigin, testConfirmLinkOwnerEmail)

	token := createSubscriptionToken(testingT, api, site, testConfirmLinkSubscriber, emailSender)

	require.NoError(testingT, api.database.Model(&model.Subscriber{}).
		Where("site_id = ? AND email = ?", site.ID, strings.ToLower(testConfirmLinkSubscriber)).
		Update("email", testConfirmLinkAlternateEmail).Error)

	response := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/unsubscribe-link?token="+url.QueryEscape(token), nil, nil)
	require.Equal(testingT, http.StatusBadRequest, response.Code)
	payload := decodeSubscriptionLinkPayload(testingT, response)
	require.Equal(testingT, "Unsubscribe", payload.Heading)
	require.Equal(testingT, "Invalid or expired token.", payload.Message)
}

func TestUnsubscribeSubscriptionLinkHandlesExistingStatus(testingT *testing.T) {
	emailSender := &recordingEmailSender{testingT: testingT}
	api := buildAPIHarness(testingT, nil, nil, emailSender)
	site := insertSite(testingT, api.database, testConfirmLinkSiteName, testConfirmLinkSiteOrigin, testConfirmLinkOwnerEmail)

	token := createSubscriptionToken(testingT, api, site, testConfirmLinkSubscriber, emailSender)

	require.NoError(testingT, api.database.Model(&model.Subscriber{}).
		Where("site_id = ? AND email = ?", site.ID, strings.ToLower(testConfirmLinkSubscriber)).
		Update("status", model.SubscriberStatusUnsubscribed).Error)

	response := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/unsubscribe-link?token="+url.QueryEscape(token), nil, nil)
	require.Equal(testingT, http.StatusOK, response.Code)
	payload := decodeSubscriptionLinkPayload(testingT, response)
	require.Equal(testingT, "Unsubscribed", payload.Heading)
	require.Equal(testingT, "Subscription already unsubscribed.", payload.Message)
}

func TestUnsubscribeSubscriptionLinkUpdatesSubscriber(testingT *testing.T) {
	emailSender := &recordingEmailSender{testingT: testingT}
	api := buildAPIHarness(testingT, nil, nil, emailSender)
	site := insertSite(testingT, api.database, testConfirmLinkSiteName, testConfirmLinkSiteOrigin, testConfirmLinkOwnerEmail)

	token := createSubscriptionToken(testingT, api, site, testConfirmLinkSubscriber, emailSender)

	response := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/unsubscribe-link?token="+url.QueryEscape(token), nil, nil)
	require.Equal(testingT, http.StatusOK, response.Code)
	payload := decodeSubscriptionLinkPayload(testingT, response)
	require.Equal(testingT, "Unsubscribed", payload.Heading)
	require.Equal(testingT, "You have been unsubscribed.", payload.Message)

	var updated model.Subscriber
	require.NoError(testingT, api.database.First(&updated, "site_id = ? AND email = ?", site.ID, strings.ToLower(testConfirmLinkSubscriber)).Error)
	require.Equal(testingT, model.SubscriberStatusUnsubscribed, updated.Status)
	require.False(testingT, updated.UnsubscribedAt.IsZero())
}

func TestUnsubscribeSubscriptionLinkReportsUpdateError(testingT *testing.T) {
	emailSender := &recordingEmailSender{testingT: testingT}
	api := buildAPIHarness(testingT, nil, nil, emailSender)
	site := insertSite(testingT, api.database, testConfirmLinkSiteName, testConfirmLinkSiteOrigin, testConfirmLinkOwnerEmail)

	token := createSubscriptionToken(testingT, api, site, testConfirmLinkSubscriber, emailSender)

	callbackName := testSubscriptionLinkUpdateCallbackName
	api.database.Callback().Update().Before(testSubscriptionLinkUpdateHookName).Register(callbackName, func(database *gorm.DB) {
		if database.Statement != nil && database.Statement.Table == testSubscriptionLinkUpdateTableName {
			database.AddError(errors.New(testSubscriptionLinkUpdateErrorMessage))
		}
	})
	testingT.Cleanup(func() {
		api.database.Callback().Update().Remove(callbackName)
	})

	response := performJSONRequest(testingT, api.router, http.MethodGet, "/api/subscriptions/unsubscribe-link?token="+url.QueryEscape(token), nil, nil)
	require.Equal(testingT, http.StatusInternalServerError, response.Code)
	payload := decodeSubscriptionLinkPayload(testingT, response)
	require.Equal(testingT, "Unsubscribe", payload.Heading)
	require.Equal(testingT, "Failed to unsubscribe.", payload.Message)
}
func createSubscriptionToken(testingT *testing.T, api apiHarness, site model.Site, emailAddress string, emailSender *recordingEmailSender) string {
	response := performJSONRequest(testingT, api.router, http.MethodPost, "/api/subscriptions", map[string]any{
		"site_id": site.ID,
		"email":   emailAddress,
	}, map[string]string{"Origin": site.AllowedOrigin})
	require.Equal(testingT, http.StatusOK, response.Code)
	require.Equal(testingT, 1, emailSender.CallCount())

	lastCall := emailSender.LastCall()
	token := extractConfirmationToken(testingT, lastCall.Message)
	require.NotEmpty(testingT, token)
	return token
}

func extractConfirmationToken(testingT *testing.T, message string) string {
	for _, line := range strings.Split(message, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "/subscriptions/confirm?token=") {
			parsed, parseErr := url.Parse(trimmed)
			require.NoError(testingT, parseErr)
			return parsed.Query().Get("token")
		}
	}
	testingT.Fatal("expected confirmation token")
	return ""
}
