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
	subscribeEmailInputID      = "mp-subscribe-email"
	subscribeNameInputID       = "mp-subscribe-name"
	subscribeSubmitButtonID    = "mp-subscribe-submit"
	subscribeStatusElementID   = "mp-subscribe-status"
	subscribeFormContainerID   = "mp-subscribe-form"
	subscribeTargetContainerID = "subscribe-target"
)

func TestSubscribeWidgetSubmitsSubscription(testingT *testing.T) {
	gin.SetMode(gin.TestMode)

	subscriptionNotifier := &recordingSubscriptionNotifier{testingT: testingT}
	emailSender := &recordingEmailSender{testingT: testingT}
	api := buildAPIHarness(testingT, nil, subscriptionNotifier, emailSender)

	server := newHTTPTestServer(testingT, api.router)

	page := buildHeadlessPage(testingT)
	screenshotsDirectory := createScreenshotsDirectory(testingT)

	site := insertSite(testingT, api.database, "Subscribe Integration", server.URL, "owner@example.com")

	demoURL := fmt.Sprintf("%s/subscribe-demo?site_id=%s", server.URL, site.ID)
	navigateToPage(testingT, page, demoURL)
	waitForVisibleElement(testingT, page, "#"+subscribeEmailInputID)

	setInputValue(testingT, page, "#"+subscribeEmailInputID, "subscriber@example.com")
	setInputValue(testingT, page, "#"+subscribeNameInputID, "Newsletter User")

	clickSelector(testingT, page, "#"+subscribeSubmitButtonID)

	var statusText string
	require.Eventually(testingT, func() bool {
		statusText = evaluateScriptString(testingT, page, `document.getElementById("`+subscribeStatusElementID+`").innerText || ""`)
		return strings.Contains(statusText, "You're on the list")
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)
	require.Contains(testingT, statusText, "You're on the list")

	_ = captureAndStoreScreenshot(testingT, page, screenshotsDirectory, "subscribe-inline")

	var stored model.Subscriber
	require.NoError(testingT, api.database.First(&stored).Error)
	require.Equal(testingT, site.ID, stored.SiteID)
	require.Equal(testingT, "subscriber@example.com", stored.Email)
	require.Equal(testingT, "Newsletter User", stored.Name)
	require.Equal(testingT, demoURL, stored.SourceURL)
	require.Equal(testingT, model.SubscriberStatusPending, stored.Status)
	require.False(testingT, stored.ConsentAt.IsZero())

	require.Equal(testingT, 1, emailSender.CallCount())
	require.Equal(testingT, 0, subscriptionNotifier.CallCount())

	lastEmail := emailSender.LastCall()
	require.Equal(testingT, stored.Email, lastEmail.Recipient)
	require.Contains(testingT, lastEmail.Subject, "Confirm your subscription")

	var confirmationLink string
	for _, line := range strings.Split(lastEmail.Message, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "/subscriptions/confirm?token=") {
			confirmationLink = line
			break
		}
	}
	require.NotEmpty(testingT, confirmationLink)

	parsedURL, parseErr := url.Parse(confirmationLink)
	require.NoError(testingT, parseErr)

	response, requestErr := server.Client().Get(server.URL + parsedURL.RequestURI())
	require.NoError(testingT, requestErr)
	require.NoError(testingT, response.Body.Close())

	var confirmed model.Subscriber
	require.NoError(testingT, api.database.First(&confirmed, "id = ?", stored.ID).Error)
	require.Equal(testingT, model.SubscriberStatusConfirmed, confirmed.Status)

	require.Equal(testingT, 1, subscriptionNotifier.CallCount())
	notification := subscriptionNotifier.LastCall()
	require.Equal(testingT, site.ID, notification.Site.ID)
	require.Equal(testingT, "owner@example.com", notification.Site.OwnerEmail)
	require.Equal(testingT, stored.ID, notification.Subscriber.ID)
	require.Equal(testingT, stored.Email, notification.Subscriber.Email)
	require.Equal(testingT, model.SubscriberStatusConfirmed, notification.Subscriber.Status)
}

func TestSubscribeWidgetRendersIntoTargetContainer(testingT *testing.T) {
	gin.SetMode(gin.TestMode)

	subscriptionNotifier := &recordingSubscriptionNotifier{testingT: testingT}
	emailSender := &recordingEmailSender{testingT: testingT}
	api := buildAPIHarness(testingT, nil, subscriptionNotifier, emailSender)

	server := newHTTPTestServer(testingT, api.router)

	site := insertSite(testingT, api.database, "Subscribe Target", server.URL, "owner@example.com")

	testCases := []struct {
		name          string
		targetTestURL string
	}{
		{
			name:          "query_param_target",
			targetTestURL: fmt.Sprintf("%s/subscribe-target-test?site_id=%s&target=%s", server.URL, site.ID, subscribeTargetContainerID),
		},
		{
			name:          "data_attribute_target",
			targetTestURL: fmt.Sprintf("%s/subscribe-target-test?site_id=%s&target=%s&data_target=true", server.URL, site.ID, subscribeTargetContainerID),
		},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(subTest *testing.T) {
			page := buildHeadlessPage(subTest)
			navigateToPage(subTest, page, testCase.targetTestURL)

			require.Eventually(subTest, func() bool {
				lookupScript := fmt.Sprintf(`(function(){
					var target = document.getElementById(%q);
					if (!target) { return false; }
					return !!target.querySelector(%q);
				}())`, subscribeTargetContainerID, "#"+subscribeFormContainerID)
				return evaluateScriptBoolean(subTest, page, lookupScript)
			}, integrationStatusWaitTimeout, integrationStatusPollInterval)
		})
	}
}

func TestSubscribeWidgetOmitsNameFieldWhenDisabled(testingT *testing.T) {
	gin.SetMode(gin.TestMode)

	subscriptionNotifier := &recordingSubscriptionNotifier{testingT: testingT}
	emailSender := &recordingEmailSender{testingT: testingT}
	api := buildAPIHarness(testingT, nil, subscriptionNotifier, emailSender)

	server := newHTTPTestServer(testingT, api.router)

	page := buildHeadlessPage(testingT)

	site := insertSite(testingT, api.database, "Subscribe No Name", server.URL, "owner@example.com")

	demoURL := fmt.Sprintf("%s/subscribe-demo?site_id=%s&name_field=false", server.URL, site.ID)
	navigateToPage(testingT, page, demoURL)
	waitForVisibleElement(testingT, page, "#"+subscribeEmailInputID)

	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, page, `document.getElementById("`+subscribeNameInputID+`") === null`)
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)
}
