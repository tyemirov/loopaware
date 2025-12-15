package httpapi_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	subscribeEmailInputID    = "mp-subscribe-email"
	subscribeNameInputID     = "mp-subscribe-name"
	subscribeSubmitButtonID  = "mp-subscribe-submit"
	subscribeStatusElementID = "mp-subscribe-status"
)

func TestSubscribeWidgetSubmitsSubscription(t *testing.T) {
	gin.SetMode(gin.TestMode)

	subscriptionNotifier := &recordingSubscriptionNotifier{t: t}
	emailSender := &recordingEmailSender{t: t}
	api := buildAPIHarness(t, nil, subscriptionNotifier, emailSender)

	server := newHTTPTestServer(t, api.router)

	page := buildHeadlessPage(t)
	screenshotsDirectory := createScreenshotsDirectory(t)

	site := insertSite(t, api.database, "Subscribe Integration", server.URL, "owner@example.com")

	demoURL := fmt.Sprintf("%s/subscribe-demo?site_id=%s", server.URL, site.ID)
	navigateToPage(t, page, demoURL)
	waitForVisibleElement(t, page, "#"+subscribeEmailInputID)

	setInputValue(t, page, "#"+subscribeEmailInputID, "subscriber@example.com")
	setInputValue(t, page, "#"+subscribeNameInputID, "Newsletter User")

	clickSelector(t, page, "#"+subscribeSubmitButtonID)

	var statusText string
	require.Eventually(t, func() bool {
		statusText = evaluateScriptString(t, page, `document.getElementById("`+subscribeStatusElementID+`").innerText || ""`)
		return strings.Contains(statusText, "You're on the list")
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)
	require.Contains(t, statusText, "You're on the list")

	_ = captureAndStoreScreenshot(t, page, screenshotsDirectory, "subscribe-inline")

	var stored model.Subscriber
	require.NoError(t, api.database.First(&stored).Error)
	require.Equal(t, site.ID, stored.SiteID)
	require.Equal(t, "subscriber@example.com", stored.Email)
	require.Equal(t, "Newsletter User", stored.Name)
	require.Equal(t, demoURL, stored.SourceURL)
	require.Equal(t, model.SubscriberStatusPending, stored.Status)
	require.False(t, stored.ConsentAt.IsZero())

	require.Equal(t, 1, emailSender.CallCount())
	require.Equal(t, 0, subscriptionNotifier.CallCount())

	lastEmail := emailSender.LastCall()
	require.Equal(t, stored.Email, lastEmail.Recipient)
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

	response, requestErr := server.Client().Get(server.URL + parsedURL.RequestURI())
	require.NoError(t, requestErr)
	require.NoError(t, response.Body.Close())

	var confirmed model.Subscriber
	require.NoError(t, api.database.First(&confirmed, "id = ?", stored.ID).Error)
	require.Equal(t, model.SubscriberStatusConfirmed, confirmed.Status)

	require.Equal(t, 1, subscriptionNotifier.CallCount())
	notification := subscriptionNotifier.LastCall()
	require.Equal(t, site.ID, notification.Site.ID)
	require.Equal(t, "owner@example.com", notification.Site.OwnerEmail)
	require.Equal(t, stored.ID, notification.Subscriber.ID)
	require.Equal(t, stored.Email, notification.Subscriber.Email)
	require.Equal(t, model.SubscriberStatusConfirmed, notification.Subscriber.Status)
}
