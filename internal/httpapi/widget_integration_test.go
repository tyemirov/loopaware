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
	"sync"
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
	integrationPageHTMLTemplate            = "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>Widget Integration</title></head><body><script defer src=\"/widget.js?site_id=%s\"></script></body></html>"
	darkThemePageHTMLTemplate              = "<!doctype html><html lang=\"en\" data-theme=\"dark\"><head><meta charset=\"utf-8\"><title>Widget Integration Dark</title><style>body{background:#0b1526;color:#f8fafc;margin:0;padding:48px;font-family:system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,Noto Sans,sans-serif;}</style></head><body><h1>LoopAware Dark Theme</h1><script defer src=\"/widget.js?site_id=%s\"></script></body></html>"
	integrationSiteName                    = "Widget Integration"
	integrationSiteOwnerEmail              = "integration@example.com"
	integrationFeedbackContactValue        = "integration@example.com"
	integrationFeedbackMessageValue        = "Headless integration feedback"
	integrationTestTimeout                 = 20 * time.Second
	integrationStatusWaitTimeout           = 5 * time.Second
	integrationStatusPollInterval          = 100 * time.Millisecond
	integrationPanelAutoHideTimeout        = 8 * time.Second
	browserStartupTimeout                  = 5 * time.Second
	headlessBrowserSkipReason              = "chromedp headless browser not available"
	headlessBrowserLocateErrorMessage      = "locate headless browser executable"
	headlessBrowserSkipMessageFormat       = "%s: %v"
	headlessBrowserEnvironmentChromedp     = "CHROMEDP_BROWSER"
	headlessBrowserEnvironmentChromePath   = "CHROME_PATH"
	headlessBrowserDisabledFeatures        = "TranslateUI,BlinkGenPropertyTrees,AutomationControlled"
	headlessBrowserPasswordStore           = "basic"
	widgetBubbleSelector                   = "#mp-feedback-bubble"
	widgetPanelSelector                    = "#mp-feedback-panel"
	widgetContactSelector                  = "#mp-feedback-panel input"
	widgetMessageSelector                  = "#mp-feedback-panel textarea"
	widgetSendButtonSelector               = "#mp-feedback-panel button"
	widgetStatusSelector                   = "#mp-feedback-panel div:last-child"
	widgetBrandingSelector                 = "#mp-feedback-branding"
	widgetBrandingLinkSelector             = "#mp-feedback-branding a"
	widgetSuccessStatusMessage             = "Thanks! Sent."
	widgetPanelHiddenDisplayValue          = "none"
	widgetPanelDisplayScript               = "document.getElementById(\"mp-feedback-panel\").style.display"
	panelInputRelativeSelector             = "input"
	panelMessageRelativeSelector           = "textarea"
	panelButtonRelativeSelector            = "button"
	darkThemeExpectedBubbleBackgroundColor = "rgb(77, 171, 247)"
	darkThemeExpectedBubbleTextColor       = "rgb(11, 21, 38)"
	darkThemeExpectedPanelBackgroundColor  = "rgb(31, 41, 55)"
	darkThemeExpectedInputBackgroundColor  = "rgb(17, 24, 39)"
	darkThemeExpectedButtonBackgroundColor = "rgb(37, 99, 235)"
	widgetBrandingLinkExpectedText         = "Marco Polo Research Lab"
	widgetBrandingContainerExpectedText    = "Built by Marco Polo Research Lab"
	widgetBrandingLinkExpectedHref         = "https://mprlab.com"
)

var headlessBrowserExecutableNames = []string{
	"chromium",
	"chromium-browser",
	"google-chrome",
	"google-chrome-stable",
	"chrome",
	"headless-shell",
}

var headlessBrowserDeterministicAllocatorFlags = []chromedp.ExecAllocatorOption{
	chromedp.Flag("headless", true),
	chromedp.Flag("disable-gpu", true),
	chromedp.Flag("no-sandbox", true),
	chromedp.Flag("disable-dev-shm-usage", true),
	chromedp.Flag("ignore-certificate-errors", true),
	chromedp.Flag("disable-background-networking", true),
	chromedp.Flag("disable-backgrounding-occluded-windows", true),
	chromedp.Flag("disable-breakpad", true),
	chromedp.Flag("disable-component-update", true),
	chromedp.Flag("disable-default-apps", true),
	chromedp.Flag("disable-extensions", true),
	chromedp.Flag("disable-features", headlessBrowserDisabledFeatures),
	chromedp.Flag("disable-hang-monitor", true),
	chromedp.Flag("disable-ipc-flooding-protection", true),
	chromedp.Flag("disable-renderer-backgrounding", true),
	chromedp.Flag("disable-sync", true),
	chromedp.Flag("enable-automation", true),
	chromedp.Flag("metrics-recording-only", true),
	chromedp.Flag("mute-audio", true),
	chromedp.Flag("no-first-run", true),
	chromedp.Flag("password-store", headlessBrowserPasswordStore),
	chromedp.Flag("remote-debugging-port", 0),
	chromedp.Flag("safebrowsing-disable-auto-update", true),
	chromedp.Flag("use-mock-keychain", true),
}

var headlessBrowserLookupCache struct {
	once sync.Once
	path string
	err  error
}

var headlessBrowserStartupCache struct {
	once sync.Once
	err  error
}

var (
	headlessBrowserRuntimeFailureMutex sync.RWMutex
	headlessBrowserRuntimeFailure      error
)

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

	runErr := runHeadlessBrowserActions(t, browserContext,
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

	var panelDisplayState string
	require.Eventually(t, func() bool {
		displayErr := chromedp.Run(browserContext, chromedp.Evaluate(widgetPanelDisplayScript, &panelDisplayState))
		if displayErr != nil {
			return false
		}
		return panelDisplayState == widgetPanelHiddenDisplayValue
	}, integrationPanelAutoHideTimeout, integrationStatusPollInterval)
	require.Equal(t, widgetPanelHiddenDisplayValue, panelDisplayState)

	var widgetBrandingContainerText string
	var widgetBrandingDisplayedText string
	var widgetBrandingDisplayedHref string
	var widgetBrandingHrefFound bool
	brandingCheckErr := chromedp.Run(browserContext,
		chromedp.WaitVisible(widgetBrandingSelector, chromedp.ByQuery),
		chromedp.Text(widgetBrandingSelector, &widgetBrandingContainerText, chromedp.ByQuery),
		chromedp.Text(widgetBrandingLinkSelector, &widgetBrandingDisplayedText, chromedp.ByQuery),
		chromedp.AttributeValue(widgetBrandingLinkSelector, "href", &widgetBrandingDisplayedHref, &widgetBrandingHrefFound, chromedp.ByQuery),
	)
	require.NoError(t, brandingCheckErr)
	require.True(t, widgetBrandingHrefFound)
	require.Equal(t, widgetBrandingContainerExpectedText, strings.TrimSpace(widgetBrandingContainerText))
	require.Equal(t, widgetBrandingLinkExpectedText, strings.TrimSpace(widgetBrandingDisplayedText))
	require.Equal(t, widgetBrandingLinkExpectedHref, strings.TrimSpace(widgetBrandingDisplayedHref))

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

	darkThemeRunErr := runHeadlessBrowserActions(t, browserContext,
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
	headlessBrowserLookupCache.once.Do(func() {
		headlessBrowserLookupCache.path, headlessBrowserLookupCache.err = discoverHeadlessBrowserExecutable()
	})
	if headlessBrowserLookupCache.err != nil {
		return "", headlessBrowserLookupCache.err
	}
	return headlessBrowserLookupCache.path, nil
}

func discoverHeadlessBrowserExecutable() (string, error) {
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

func loadHeadlessBrowserRuntimeFailure() error {
	headlessBrowserRuntimeFailureMutex.RLock()
	defer headlessBrowserRuntimeFailureMutex.RUnlock()
	return headlessBrowserRuntimeFailure
}

func storeHeadlessBrowserRuntimeFailure(failure error) {
	if failure == nil {
		return
	}
	headlessBrowserRuntimeFailureMutex.Lock()
	if headlessBrowserRuntimeFailure == nil {
		headlessBrowserRuntimeFailure = failure
	}
	headlessBrowserRuntimeFailureMutex.Unlock()
}

func buildHeadlessAllocatorOptions(browserExecutablePath string) []chromedp.ExecAllocatorOption {
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options, chromedp.ExecPath(browserExecutablePath))
	options = append(options, headlessBrowserDeterministicAllocatorFlags...)
	return options
}

func ensureHeadlessBrowserReady(browserExecutablePath string) error {
	headlessBrowserStartupCache.once.Do(func() {
		allocatorOptions := buildHeadlessAllocatorOptions(browserExecutablePath)
		allocatorContext, allocatorCancel := chromedp.NewExecAllocator(context.Background(), allocatorOptions...)
		defer allocatorCancel()

		browserContext, browserCancel := chromedp.NewContext(allocatorContext)
		defer browserCancel()

		startupContext, startupCancel := context.WithTimeout(browserContext, browserStartupTimeout)
		defer startupCancel()

		headlessBrowserStartupCache.err = chromedp.Run(startupContext,
			chromedp.Navigate("about:blank"),
			chromedp.WaitReady("body", chromedp.ByQuery),
		)
	})
	if headlessBrowserStartupCache.err != nil {
		storeHeadlessBrowserRuntimeFailure(headlessBrowserStartupCache.err)
	}
	return headlessBrowserStartupCache.err
}

func buildHeadlessBrowserContext(testingT *testing.T) context.Context {
	testingT.Helper()

	if failure := loadHeadlessBrowserRuntimeFailure(); failure != nil {
		testingT.Skipf(headlessBrowserSkipMessageFormat, headlessBrowserSkipReason, failure)
	}

	browserExecutablePath, locateBrowserErr := locateHeadlessBrowserExecutable()
	if locateBrowserErr != nil {
		storeHeadlessBrowserRuntimeFailure(locateBrowserErr)
		testingT.Skipf(headlessBrowserSkipMessageFormat, headlessBrowserSkipReason, locateBrowserErr)
	}

	if readinessErr := ensureHeadlessBrowserReady(browserExecutablePath); readinessErr != nil {
		testingT.Skipf(headlessBrowserSkipMessageFormat, headlessBrowserSkipReason, readinessErr)
	}

	allocatorOptions := buildHeadlessAllocatorOptions(browserExecutablePath)

	allocatorContext, allocatorCancel := chromedp.NewExecAllocator(context.Background(), allocatorOptions...)
	testingT.Cleanup(allocatorCancel)

	browserContext, browserCancel := chromedp.NewContext(allocatorContext)
	testingT.Cleanup(browserCancel)

	startupContext, startupCancel := context.WithTimeout(browserContext, browserStartupTimeout)
	defer startupCancel()

	startupErr := chromedp.Run(startupContext,
		chromedp.Navigate("about:blank"),
		chromedp.WaitReady("body", chromedp.ByQuery),
	)
	if startupErr != nil {
		storeHeadlessBrowserRuntimeFailure(startupErr)
		testingT.Skipf(headlessBrowserSkipMessageFormat, headlessBrowserSkipReason, startupErr)
	}

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

func runHeadlessBrowserActions(testingT *testing.T, browserContext context.Context, actions ...chromedp.Action) error {
	testingT.Helper()

	runErr := chromedp.Run(browserContext, actions...)
	if runErr == nil {
		return nil
	}

	if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
		testingT.Skipf(headlessBrowserSkipMessageFormat, headlessBrowserSkipReason, runErr)
	}

	return runErr
}
