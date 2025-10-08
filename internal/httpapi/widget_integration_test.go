package httpapi_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
)

const (
	integrationPageRoutePath               = "/widget-integration"
	darkThemePageRoutePath                 = "/widget-integration-dark"
	integrationPageContentType             = "text/html; charset=utf-8"
	integrationPageHTMLTemplate            = "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>Widget Integration</title></head><body><script src=\"/widget.js?site_id=%s\"></script></body></html>"
	darkThemePageHTMLTemplate              = "<!doctype html><html lang=\"en\" data-theme=\"dark\"><head><meta charset=\"utf-8\"><title>Widget Integration Dark</title><style>body{background:#0b1526;color:#f8fafc;margin:0;padding:48px;font-family:system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,Noto Sans,sans-serif;}</style></head><body><h1>LoopAware Dark Theme</h1><script src=\"/widget.js?site_id=%s\"></script></body></html>"
	integrationSiteName                    = "Widget Integration"
	integrationSiteOwnerEmail              = "integration@example.com"
	integrationFeedbackContactValue        = "integration@example.com"
	integrationFeedbackMessageValue        = "Headless integration feedback"
	integrationTestTimeout                 = 20 * time.Second
	integrationStatusWaitTimeout           = 5 * time.Second
	integrationStatusPollInterval          = 100 * time.Millisecond
	headlessBrowserSkipReason              = "chromedp headless browser not available"
	headlessBrowserLocateErrorMessage      = "locate headless browser executable"
	headlessBrowserEnvironmentChromedp     = "CHROMEDP_BROWSER"
	headlessBrowserEnvironmentChromePath   = "CHROME_PATH"
	widgetBubbleSelector                   = "#mp-feedback-bubble"
	widgetPanelSelector                    = "#mp-feedback-panel"
	widgetContactSelector                  = "#mp-feedback-panel input"
	widgetMessageSelector                  = "#mp-feedback-panel textarea"
	widgetSendButtonSelector               = "#mp-feedback-panel button"
	widgetStatusSelector                   = "#mp-feedback-panel div:last-child"
	widgetSuccessStatusMessage             = "Thanks! Sent."
	panelInputRelativeSelector             = "input"
	panelMessageRelativeSelector           = "textarea"
	panelButtonRelativeSelector            = "button"
	darkThemeExpectedBubbleBackgroundColor = "rgb(77, 171, 247)"
	darkThemeExpectedBubbleTextColor       = "rgb(11, 21, 38)"
	darkThemeExpectedPanelBackgroundColor  = "rgb(31, 41, 55)"
	darkThemeExpectedInputBackgroundColor  = "rgb(17, 24, 39)"
	darkThemeExpectedButtonBackgroundColor = "rgb(37, 99, 235)"
)

var headlessBrowserExecutableNames = []string{
	"chromium",
	"chromium-browser",
	"google-chrome",
	"google-chrome-stable",
	"chrome",
	"headless-shell",
}

var errHeadlessBrowserNotFound = errors.New("headless browser executable not found")

func TestWidgetIntegrationSubmitsFeedback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	browserContext := buildHeadlessBrowserContext(t)

	apiHarness := buildAPIHarness(t)

	server := httptest.NewServer(apiHarness.router)
	t.Cleanup(server.Close)

	site := insertSite(t, apiHarness.database, integrationSiteName, server.URL, integrationSiteOwnerEmail)

	integrationPageHTML := fmt.Sprintf(integrationPageHTMLTemplate, site.ID)
	apiHarness.router.GET(integrationPageRoutePath, func(ginContext *gin.Context) {
		ginContext.Data(http.StatusOK, integrationPageContentType, []byte(integrationPageHTML))
	})

	integrationPageURL := server.URL + integrationPageRoutePath

	var contactFitsWithinPanel bool
	var messageFitsWithinPanel bool
	var sendButtonFitsWithinPanel bool

	runErr := chromedp.Run(browserContext,
		chromedp.Navigate(integrationPageURL),
		chromedp.WaitVisible(widgetBubbleSelector, chromedp.ByQuery),
		chromedp.Click(widgetBubbleSelector, chromedp.ByQuery),
		chromedp.WaitVisible(widgetPanelSelector, chromedp.ByQuery),
		chromedp.SetValue(widgetContactSelector, integrationFeedbackContactValue, chromedp.ByQuery),
		chromedp.SetValue(widgetMessageSelector, integrationFeedbackMessageValue, chromedp.ByQuery),
		chromedp.Evaluate(formElementFitsPanelScript(panelInputRelativeSelector), &contactFitsWithinPanel),
		chromedp.Evaluate(formElementFitsPanelScript(panelMessageRelativeSelector), &messageFitsWithinPanel),
		chromedp.Evaluate(formElementFitsPanelScript(panelButtonRelativeSelector), &sendButtonFitsWithinPanel),
		chromedp.Click(widgetSendButtonSelector, chromedp.ByQuery),
	)
	require.NoError(t, runErr)
	require.True(t, contactFitsWithinPanel)
	require.True(t, messageFitsWithinPanel)
	require.True(t, sendButtonFitsWithinPanel)

	var widgetStatusText string
	require.Eventually(t, func() bool {
		statusErr := chromedp.Run(browserContext, chromedp.Text(widgetStatusSelector, &widgetStatusText, chromedp.ByQuery))
		if statusErr != nil {
			return false
		}
		return widgetStatusText == widgetSuccessStatusMessage
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)
	require.Equal(t, widgetSuccessStatusMessage, widgetStatusText)

	var storedFeedbackRecords []model.Feedback
	queryErr := apiHarness.database.Find(&storedFeedbackRecords).Error
	require.NoError(t, queryErr)
	require.Len(t, storedFeedbackRecords, 1)

	storedFeedbackRecord := storedFeedbackRecords[0]
	require.Equal(t, site.ID, storedFeedbackRecord.SiteID)
	require.Equal(t, integrationFeedbackContactValue, storedFeedbackRecord.Contact)
	require.Equal(t, integrationFeedbackMessageValue, storedFeedbackRecord.Message)
}

func TestWidgetAppliesDarkThemeStyles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	browserContext := buildHeadlessBrowserContext(t)

	apiHarness := buildAPIHarness(t)

	server := httptest.NewServer(apiHarness.router)
	t.Cleanup(server.Close)

	site := insertSite(t, apiHarness.database, integrationSiteName, server.URL, integrationSiteOwnerEmail)

	darkThemePageHTML := fmt.Sprintf(darkThemePageHTMLTemplate, site.ID)
	apiHarness.router.GET(darkThemePageRoutePath, func(ginContext *gin.Context) {
		ginContext.Data(http.StatusOK, integrationPageContentType, []byte(darkThemePageHTML))
	})

	darkThemePageURL := server.URL + darkThemePageRoutePath

	var bubbleBackgroundColor string
	var bubbleTextColor string
	var panelBackgroundColor string
	var inputBackgroundColor string
	var buttonBackgroundColor string

	darkThemeRunErr := chromedp.Run(browserContext,
		chromedp.Navigate(darkThemePageURL),
		chromedp.WaitVisible(widgetBubbleSelector, chromedp.ByQuery),
		chromedp.Evaluate(`window.getComputedStyle(document.getElementById("mp-feedback-bubble")).backgroundColor`, &bubbleBackgroundColor),
		chromedp.Evaluate(`window.getComputedStyle(document.getElementById("mp-feedback-bubble")).color`, &bubbleTextColor),
		chromedp.Click(widgetBubbleSelector, chromedp.ByQuery),
		chromedp.WaitVisible(widgetPanelSelector, chromedp.ByQuery),
		chromedp.Evaluate(`window.getComputedStyle(document.getElementById("mp-feedback-panel")).backgroundColor`, &panelBackgroundColor),
		chromedp.Evaluate(`window.getComputedStyle(document.querySelector("#mp-feedback-panel input")).backgroundColor`, &inputBackgroundColor),
		chromedp.Evaluate(`window.getComputedStyle(document.querySelector("#mp-feedback-panel button")).backgroundColor`, &buttonBackgroundColor),
	)
	require.NoError(t, darkThemeRunErr)

	require.Equal(t, darkThemeExpectedBubbleBackgroundColor, bubbleBackgroundColor)
	require.Equal(t, darkThemeExpectedBubbleTextColor, bubbleTextColor)
	require.Equal(t, darkThemeExpectedPanelBackgroundColor, panelBackgroundColor)
	require.Equal(t, darkThemeExpectedInputBackgroundColor, inputBackgroundColor)
	require.Equal(t, darkThemeExpectedButtonBackgroundColor, buttonBackgroundColor)
}

func locateHeadlessBrowserExecutable() (string, error) {
	environmentVariableNames := []string{
		headlessBrowserEnvironmentChromedp,
		headlessBrowserEnvironmentChromePath,
	}

	for _, environmentVariableName := range environmentVariableNames {
		environmentValue := strings.TrimSpace(os.Getenv(environmentVariableName))
		if environmentValue == "" {
			continue
		}
		return environmentValue, nil
	}

	for _, executableName := range headlessBrowserExecutableNames {
		executablePath, lookupErr := exec.LookPath(executableName)
		if lookupErr == nil {
			return executablePath, nil
		}
	}

	return "", fmt.Errorf("%s: %w", headlessBrowserLocateErrorMessage, errHeadlessBrowserNotFound)
}

func buildHeadlessBrowserContext(testingT *testing.T) context.Context {
	testingT.Helper()

	browserExecutablePath, locateBrowserErr := locateHeadlessBrowserExecutable()
	if locateBrowserErr != nil {
		testingT.Skipf("%s: %v", headlessBrowserSkipReason, locateBrowserErr)
	}

	headlessAllocatorOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(browserExecutablePath),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("ignore-certificate-errors", true),
	)

	allocatorContext, allocatorCancel := chromedp.NewExecAllocator(context.Background(), headlessAllocatorOptions...)
	testingT.Cleanup(allocatorCancel)

	browserContext, browserCancel := chromedp.NewContext(allocatorContext)
	testingT.Cleanup(browserCancel)

	contextWithTimeout, timeoutCancel := context.WithTimeout(browserContext, integrationTestTimeout)
	testingT.Cleanup(timeoutCancel)

	return contextWithTimeout
}

func formElementFitsPanelScript(cssSelector string) string {
	return fmt.Sprintf(`(function(selector){
		var panel = document.getElementById("mp-feedback-panel");
		if (!panel) { return false; }
		var element = panel.querySelector(selector);
		if (!element) { return false; }
		var panelRect = panel.getBoundingClientRect();
		var elementRect = element.getBoundingClientRect();
		return (elementRect.left >= panelRect.left - 0.5) && (elementRect.right <= panelRect.right + 0.5);
	})(%q)`, cssSelector)
}
