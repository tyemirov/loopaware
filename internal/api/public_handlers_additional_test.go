package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

func TestConfirmSubscriptionLinkJSONReturnsBadRequestWhenSubscriberMissing(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	require.NoError(testingT, storage.AutoMigrate(database))

	handlers := NewPublicHandlers(database, zap.NewNop(), nil, nil, nil, nil, false, "http://example.com", testTokenSecretValue, nil)

	token := buildSubscriptionToken(testingT, subscriptionConfirmationTokenPayload{
		SubscriberID: testTokenSubscriberID,
		SiteID:       testTokenSiteID,
		Email:        testTokenEmail,
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
	}, testTokenSecretValue)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/public/subscriptions/confirm-link?token="+url.QueryEscape(token), nil)
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = request

	handlers.ConfirmSubscriptionLinkJSON(ginContext)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestUpdateSubscriptionStatusRateLimited(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/public/subscriptions/confirm", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	ginContext, _ := gin.CreateTestContext(responseRecorder)
	ginContext.Request = request

	handlers := &PublicHandlers{
		rateWindow:                time.Minute,
		maxRequestsPerIPPerWindow: 0,
		rateCountersByIP:          make(map[string]int),
	}

	handlers.updateSubscriptionStatus(ginContext, model.SubscriberStatusConfirmed)
	require.Equal(testingT, http.StatusTooManyRequests, responseRecorder.Code)
}
