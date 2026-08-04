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
