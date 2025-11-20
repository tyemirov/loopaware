package httpapi_test

import (
	"fmt"
	"net/http/httptest"
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

	page := buildHeadlessPage(t)
	screenshotsDirectory := createScreenshotsDirectory(t)

	api := buildAPIHarness(t, nil)

	server := httptest.NewServer(api.router)
	t.Cleanup(server.Close)

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
	require.Equal(t, model.SubscriberStatusPending, stored.Status)
}
