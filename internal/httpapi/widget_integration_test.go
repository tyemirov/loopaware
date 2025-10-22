package httpapi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod"
	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	integrationPageRoutePath        = "/widget-integration"
	darkThemePageRoutePath          = "/widget-integration-dark"
	integrationPageContentType      = "text/html; charset=utf-8"
	integrationPageHTMLTemplate     = "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>Widget Integration</title></head><body><script defer src=\"/widget.js?site_id=%s\"></script></body></html>"
	darkThemePageHTMLTemplate       = "<!doctype html><html lang=\"en\" data-theme=\"dark\"><head><meta charset=\"utf-8\"><title>Widget Integration Dark</title><style>body{background:#0b1526;color:#f8fafc;margin:0;padding:48px;font-family:system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,Noto Sans,sans-serif;}</style></head><body><h1>LoopAware Dark Theme</h1><script defer src=\"/widget.js?site_id=%s\"></script></body></html>"
	integrationSiteName             = "Widget Integration"
	integrationSiteOwnerEmail       = "integration@example.com"
	integrationFeedbackContactValue = "integration@example.com"
	integrationFeedbackMessageValue = "Headless integration feedback"
	integrationStatusWaitTimeout    = 5 * time.Second
	integrationStatusPollInterval   = 100 * time.Millisecond
	integrationPanelAutoHideTimeout = 4 * time.Second
	widgetBubbleSelector            = "#mp-feedback-bubble"
	widgetPanelSelector             = "#mp-feedback-panel"
	widgetContactSelector           = "#mp-feedback-panel input"
	widgetMessageSelector           = "#mp-feedback-panel textarea"
	widgetSendButtonSelector        = "#mp-feedback-panel button[type='button']:not([aria-label='Close feedback panel'])"
	widgetBrandingSelector          = "#mp-feedback-branding"
	widgetBrandingLinkSelector      = "#mp-feedback-branding a"
	widgetSuccessStatusMessage      = "Thanks! Sent."
	widgetStatusResolveScript       = `(function(){
		var panel = document.getElementById("mp-feedback-panel");
		if (!panel) { return ""; }
		var nodes = panel.querySelectorAll("div");
		for (var index = 0; index < nodes.length; index++) {
			var node = nodes[index];
			if (!node || !node.style) { continue; }
			if (node.style.fontSize === "12px") {
				return (node.innerText || "").trim();
			}
		}
		return "";
	})()`
	widgetPanelHiddenDisplayValue           = "none"
	panelDisplayBlockValue                  = "block"
	widgetPanelDisplayScript                = "document.getElementById(\"mp-feedback-panel\").style.display"
	panelInputRelativeSelector              = "input"
	panelMessageRelativeSelector            = "textarea"
	panelButtonRelativeSelector             = "button"
	lightThemeExpectedBubbleBackgroundColor = "rgb(13, 110, 253)"
	lightThemeExpectedBubbleTextColor       = "rgb(255, 255, 255)"
	lightThemeExpectedPanelBackgroundColor  = "rgb(255, 255, 255)"
	lightThemeExpectedInputBackgroundColor  = "rgb(255, 255, 255)"
	lightThemeExpectedButtonBackgroundColor = "rgb(13, 110, 253)"
	darkThemeExpectedBubbleBackgroundColor  = "rgb(77, 171, 247)"
	darkThemeExpectedBubbleTextColor        = "rgb(11, 21, 38)"
	darkThemeExpectedPanelBackgroundColor   = "rgb(31, 41, 55)"
	darkThemeExpectedInputBackgroundColor   = "rgb(17, 24, 39)"
	darkThemeExpectedButtonBackgroundColor  = "rgb(37, 99, 235)"
	widgetBrandingLinkExpectedText          = "Marco Polo Research Lab"
	widgetBrandingContainerExpectedText     = "Built by Marco Polo Research Lab"
	widgetBrandingLinkExpectedHref          = "https://mprlab.com"
	widgetCloseButtonSelector               = "#mp-feedback-panel button[aria-label='Close feedback panel']"
	widgetCloseButtonExpectedText           = "×"
	widgetHeadlineSelector                  = "#mp-feedback-headline"
	widgetContactFocusScript                = `document.activeElement === document.querySelector("#mp-feedback-panel input")`
	widgetContactTabsToMessageScript        = `(function(){
		var contact = document.querySelector("#mp-feedback-panel input");
		var message = document.querySelector("#mp-feedback-panel textarea");
		if (!contact || !message) { return false; }
		contact.focus();
		var tabEvent = new KeyboardEvent("keydown", { key: "Tab", bubbles: true, cancelable: true });
		contact.dispatchEvent(tabEvent);
		return document.activeElement === message && tabEvent.defaultPrevented;
	})()`
	widgetMessageTabsToContactScript = `(function(){
                var contact = document.querySelector("#mp-feedback-panel input");
                var message = document.querySelector("#mp-feedback-panel textarea");
                if (!contact || !message) { return false; }
                message.focus();
                var tabEvent = new KeyboardEvent("keydown", { key: "Tab", bubbles: true, cancelable: true });
                message.dispatchEvent(tabEvent);
                return document.activeElement === contact && tabEvent.defaultPrevented;
        })()`
	customWidgetBubbleSide              = "left"
	customWidgetBottomOffsetPixels      = 32
	widgetHorizontalOffsetPixels        = 16
	widgetBubbleDiameterPixels          = 56
	widgetPanelVerticalSpacingPixels    = 64
	positionTolerancePixels             = 6.0
	closeButtonAlignmentTolerancePixels = 2.0
	bootstrapThemeAttributeName         = "data-bs-theme"
	bootstrapThemeLightValue            = "light"
	bootstrapThemeDarkValue             = "dark"
)

func setBootstrapThemeAttribute(testingT *testing.T, page *rod.Page, themeValue string) {
	testingT.Helper()

	themeScript := fmt.Sprintf(`(function(){
                var desiredThemeValue = %q;
                var htmlElement = document.documentElement;
                if (htmlElement) {
                        htmlElement.removeAttribute("data-theme");
                        htmlElement.setAttribute("%s", desiredThemeValue);
                }
                if (document.body) {
                        document.body.setAttribute("%s", desiredThemeValue);
                }
                return true;
        })()`, themeValue, bootstrapThemeAttributeName, bootstrapThemeAttributeName)

	require.True(testingT, evaluateScriptBoolean(testingT, page, themeScript))
}

func TestWidgetIntegrationSubmitsFeedback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	page := buildHeadlessPage(t)
	screenshotsDirectory := createScreenshotsDirectory(t)

	apiHarness := buildAPIHarness(t)

	server := httptest.NewServer(apiHarness.router)
	t.Cleanup(server.Close)

	site := insertSite(t, apiHarness.database, integrationSiteName, server.URL, integrationSiteOwnerEmail)
	require.NoError(t, apiHarness.database.Model(&model.Site{}).
		Where("id = ?", site.ID).
		Updates(map[string]interface{}{
			"widget_bubble_side":             customWidgetBubbleSide,
			"widget_bubble_bottom_offset_px": customWidgetBottomOffsetPixels,
		}).Error)

	integrationPageHTML := fmt.Sprintf(integrationPageHTMLTemplate, site.ID)
	apiHarness.router.GET(integrationPageRoutePath, func(ginContext *gin.Context) {
		ginContext.Data(http.StatusOK, integrationPageContentType, []byte(integrationPageHTML))
	})

	integrationPageURL := server.URL + integrationPageRoutePath

	navigateToPage(t, page, integrationPageURL)
	waitForVisibleElement(t, page, widgetBubbleSelector)

	bubbleBounds := resolveViewportBounds(t, page, widgetBubbleSelector)
	expectedBubbleLeft := float64(widgetHorizontalOffsetPixels)
	require.InDelta(t, expectedBubbleLeft, bubbleBounds.Left, positionTolerancePixels)
	expectedBubbleTop := float64(headlessViewportHeight - widgetBubbleDiameterPixels - customWidgetBottomOffsetPixels)
	require.InDelta(t, expectedBubbleTop, bubbleBounds.Top, positionTolerancePixels)
	bubbleScreenshot := captureAndStoreScreenshot(t, page, screenshotsDirectory, "widget-light-bubble")
	require.FileExists(t, filepath.Join(screenshotsDirectory, "widget-light-bubble.png"))
	analyzeScreenshotRegion(t, bubbleScreenshot, bubbleBounds, screenshotExpectation{
		MinimumVariance: screenshotMinimumVariance,
	}, float64(headlessViewportWidth), float64(headlessViewportHeight))

	clickSelector(t, page, widgetBubbleSelector)
	waitForVisibleElement(t, page, widgetPanelSelector)
	require.True(t, evaluateScriptBoolean(t, page, widgetContactFocusScript))
	require.True(t, evaluateScriptBoolean(t, page, widgetContactTabsToMessageScript))
	require.True(t, evaluateScriptBoolean(t, page, widgetMessageTabsToContactScript))

	panelBounds := resolveViewportBounds(t, page, widgetPanelSelector)
	require.InDelta(t, expectedBubbleLeft, panelBounds.Left, positionTolerancePixels)
	panelScreenshot := captureAndStoreScreenshot(t, page, screenshotsDirectory, "widget-light-panel")
	require.FileExists(t, filepath.Join(screenshotsDirectory, "widget-light-panel.png"))
	analyzeScreenshotRegion(t, panelScreenshot, panelBounds, screenshotExpectation{
		MinimumVariance: screenshotMinimumVariance,
	}, float64(headlessViewportWidth), float64(headlessViewportHeight))

	setInputValue(t, page, widgetContactSelector, integrationFeedbackContactValue)
	setInputValue(t, page, widgetMessageSelector, integrationFeedbackMessageValue)

	contactFitsWithinPanel := evaluateFormElementFits(t, page, panelInputRelativeSelector)
	messageFitsWithinPanel := evaluateFormElementFits(t, page, panelMessageRelativeSelector)
	sendButtonFitsWithinPanel := evaluateFormElementFits(t, page, panelButtonRelativeSelector)

	require.True(t, contactFitsWithinPanel)
	require.True(t, messageFitsWithinPanel)
	require.True(t, sendButtonFitsWithinPanel)

	clickSelector(t, page, widgetSendButtonSelector)

	var widgetStatusText string
	require.Eventually(t, func() bool {
		widgetStatusText = evaluateScriptString(t, page, widgetStatusResolveScript)
		t.Logf("widget status debug: %q", widgetStatusText)
		return strings.Contains(widgetStatusText, widgetSuccessStatusMessage)
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)
	require.Contains(t, widgetStatusText, widgetSuccessStatusMessage)

	brandingContainer := waitForVisibleElement(t, page, widgetBrandingSelector)
	brandingContainerText, brandingContainerErr := brandingContainer.Text()
	require.NoError(t, brandingContainerErr)

	brandingLinkElement, brandingLinkErr := page.Element(widgetBrandingLinkSelector)
	require.NoError(t, brandingLinkErr)
	require.NoError(t, brandingLinkElement.WaitVisible())
	brandingLinkText, brandingLinkTextErr := brandingLinkElement.Text()
	require.NoError(t, brandingLinkTextErr)
	brandingLinkHref, brandingLinkHrefErr := brandingLinkElement.Attribute("href")
	require.NoError(t, brandingLinkHrefErr)
	require.NotNil(t, brandingLinkHref)

	require.Equal(t, widgetBrandingContainerExpectedText, strings.TrimSpace(brandingContainerText))
	require.Equal(t, widgetBrandingLinkExpectedText, strings.TrimSpace(brandingLinkText))
	require.Equal(t, widgetBrandingLinkExpectedHref, strings.TrimSpace(*brandingLinkHref))

	var storedFeedbackRecords []model.Feedback
	queryErr := apiHarness.database.Find(&storedFeedbackRecords).Error
	require.NoError(t, queryErr)
	require.Len(t, storedFeedbackRecords, 1)

	storedFeedbackRecord := storedFeedbackRecords[0]
	require.Equal(t, site.ID, storedFeedbackRecord.SiteID)
	require.Equal(t, integrationFeedbackContactValue, storedFeedbackRecord.Contact)
	require.Equal(t, integrationFeedbackMessageValue, storedFeedbackRecord.Message)

	var panelDisplayState string
	require.Eventually(t, func() bool {
		panelDisplayState = evaluateScriptString(t, page, widgetPanelDisplayScript)
		t.Logf("panel display state: %q", panelDisplayState)
		return panelDisplayState == widgetPanelHiddenDisplayValue
	}, integrationPanelAutoHideTimeout, integrationStatusPollInterval)
	require.Equal(t, widgetPanelHiddenDisplayValue, panelDisplayState)
}

func TestWidgetAppliesDarkThemeStyles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	page := buildHeadlessPage(t)
	screenshotsDirectory := createScreenshotsDirectory(t)

	apiHarness := buildAPIHarness(t)

	server := httptest.NewServer(apiHarness.router)
	t.Cleanup(server.Close)

	site := insertSite(t, apiHarness.database, integrationSiteName, server.URL, integrationSiteOwnerEmail)

	darkThemePageHTML := fmt.Sprintf(darkThemePageHTMLTemplate, site.ID)
	apiHarness.router.GET(darkThemePageRoutePath, func(ginContext *gin.Context) {
		ginContext.Data(http.StatusOK, integrationPageContentType, []byte(darkThemePageHTML))
	})

	darkThemePageURL := server.URL + darkThemePageRoutePath

	navigateToPage(t, page, darkThemePageURL)
	waitForVisibleElement(t, page, widgetBubbleSelector)

	expectedBubbleBackgroundColor := mustParseRGBColor(t, darkThemeExpectedBubbleBackgroundColor)
	bubbleBounds := resolveViewportBounds(t, page, widgetBubbleSelector)
	darkBubbleScreenshot := captureAndStoreScreenshot(t, page, screenshotsDirectory, "widget-dark-bubble")
	require.FileExists(t, filepath.Join(screenshotsDirectory, "widget-dark-bubble.png"))
	analyzeScreenshotRegion(t, darkBubbleScreenshot, bubbleBounds, screenshotExpectation{
		MinimumVariance: screenshotMinimumVariance,
		ColorPresence: []colorPresenceExpectation{
			{
				Color:        expectedBubbleBackgroundColor,
				Tolerance:    colorChannelTolerance,
				MinimumRatio: colorPresenceMinimumRatio,
			},
		},
	}, float64(headlessViewportWidth), float64(headlessViewportHeight))

	bubbleBackgroundColor := evaluateScriptString(t, page, `window.getComputedStyle(document.getElementById("mp-feedback-bubble")).backgroundColor`)
	bubbleTextColor := evaluateScriptString(t, page, `window.getComputedStyle(document.getElementById("mp-feedback-bubble")).color`)

	clickSelector(t, page, widgetBubbleSelector)
	waitForVisibleElement(t, page, widgetPanelSelector)

	panelBounds := resolveViewportBounds(t, page, widgetPanelSelector)
	inputBounds := resolveViewportBounds(t, page, widgetContactSelector)
	buttonBounds := resolveViewportBounds(t, page, widgetSendButtonSelector)

	darkPanelScreenshot := captureAndStoreScreenshot(t, page, screenshotsDirectory, "widget-dark-panel")
	require.FileExists(t, filepath.Join(screenshotsDirectory, "widget-dark-panel.png"))

	expectedPanelBackgroundColor := mustParseRGBColor(t, darkThemeExpectedPanelBackgroundColor)
	expectedInputBackgroundColor := mustParseRGBColor(t, darkThemeExpectedInputBackgroundColor)
	expectedButtonBackgroundColor := mustParseRGBColor(t, darkThemeExpectedButtonBackgroundColor)

	analyzeScreenshotRegion(t, darkPanelScreenshot, panelBounds, screenshotExpectation{
		MinimumVariance: screenshotMinimumVariance,
		ColorPresence: []colorPresenceExpectation{
			{
				Color:        expectedPanelBackgroundColor,
				Tolerance:    colorChannelTolerance,
				MinimumRatio: colorPresenceMinimumRatio,
			},
		},
	}, float64(headlessViewportWidth), float64(headlessViewportHeight))

	analyzeScreenshotRegion(t, darkPanelScreenshot, inputBounds, screenshotExpectation{
		MinimumVariance: screenshotMinimumVariance,
		ColorPresence: []colorPresenceExpectation{
			{
				Color:        expectedInputBackgroundColor,
				Tolerance:    colorChannelTolerance,
				MinimumRatio: colorPresenceMinimumRatio,
			},
		},
	}, float64(headlessViewportWidth), float64(headlessViewportHeight))

	analyzeScreenshotRegion(t, darkPanelScreenshot, buttonBounds, screenshotExpectation{
		MinimumVariance: screenshotMinimumVariance,
		ColorPresence: []colorPresenceExpectation{
			{
				Color:        expectedButtonBackgroundColor,
				Tolerance:    colorChannelTolerance,
				MinimumRatio: colorPresenceMinimumRatio,
			},
		},
	}, float64(headlessViewportWidth), float64(headlessViewportHeight))

	panelBackgroundColor := evaluateScriptString(t, page, `window.getComputedStyle(document.getElementById("mp-feedback-panel")).backgroundColor`)
	inputBackgroundColor := evaluateScriptString(t, page, `window.getComputedStyle(document.querySelector("#mp-feedback-panel input")).backgroundColor`)
	buttonBackgroundColor := evaluateScriptString(t, page, `window.getComputedStyle(document.querySelector("#mp-feedback-panel button[type='button']:not([aria-label='Close feedback panel'])")).backgroundColor`)

	require.Equal(t, darkThemeExpectedBubbleBackgroundColor, bubbleBackgroundColor)
	require.Equal(t, darkThemeExpectedBubbleTextColor, bubbleTextColor)
	require.Equal(t, darkThemeExpectedPanelBackgroundColor, panelBackgroundColor)
	require.Equal(t, darkThemeExpectedInputBackgroundColor, inputBackgroundColor)
	require.Equal(t, darkThemeExpectedButtonBackgroundColor, buttonBackgroundColor)
}

func TestWidgetRespondsToThemeToggle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	page := buildHeadlessPage(t)

	apiHarness := buildAPIHarness(t)

	server := httptest.NewServer(apiHarness.router)
	t.Cleanup(server.Close)

	site := insertSite(t, apiHarness.database, integrationSiteName, server.URL, integrationSiteOwnerEmail)

	integrationPageHTML := fmt.Sprintf(integrationPageHTMLTemplate, site.ID)
	apiHarness.router.GET(integrationPageRoutePath, func(ginContext *gin.Context) {
		ginContext.Data(http.StatusOK, integrationPageContentType, []byte(integrationPageHTML))
	})

	integrationPageURL := server.URL + integrationPageRoutePath

	navigateToPage(t, page, integrationPageURL)
	waitForVisibleElement(t, page, widgetBubbleSelector)

	setBootstrapThemeAttribute(t, page, bootstrapThemeLightValue)

	require.Eventually(t, func() bool {
		lightBubbleColor := evaluateScriptString(t, page, `window.getComputedStyle(document.getElementById("mp-feedback-bubble")).backgroundColor`)
		return lightBubbleColor == lightThemeExpectedBubbleBackgroundColor
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)

	setBootstrapThemeAttribute(t, page, bootstrapThemeDarkValue)

	require.Eventually(t, func() bool {
		darkBubbleColor := evaluateScriptString(t, page, `window.getComputedStyle(document.getElementById("mp-feedback-bubble")).backgroundColor`)
		return darkBubbleColor == darkThemeExpectedBubbleBackgroundColor
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)

	clickSelector(t, page, widgetBubbleSelector)
	waitForVisibleElement(t, page, widgetPanelSelector)

	require.Eventually(t, func() bool {
		darkPanelColor := evaluateScriptString(t, page, `window.getComputedStyle(document.getElementById("mp-feedback-panel")).backgroundColor`)
		return darkPanelColor == darkThemeExpectedPanelBackgroundColor
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)

	require.Eventually(t, func() bool {
		darkInputColor := evaluateScriptString(t, page, `window.getComputedStyle(document.querySelector("#mp-feedback-panel textarea")).backgroundColor`)
		return darkInputColor == darkThemeExpectedInputBackgroundColor
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)

	require.Eventually(t, func() bool {
		darkButtonColor := evaluateScriptString(t, page, `window.getComputedStyle(document.querySelector("#mp-feedback-panel button[type='button']:not([aria-label='Close feedback panel'])")).backgroundColor`)
		return darkButtonColor == darkThemeExpectedButtonBackgroundColor
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)

	setBootstrapThemeAttribute(t, page, bootstrapThemeLightValue)

	require.Eventually(t, func() bool {
		lightBubbleColor := evaluateScriptString(t, page, `window.getComputedStyle(document.getElementById("mp-feedback-bubble")).backgroundColor`)
		return lightBubbleColor == lightThemeExpectedBubbleBackgroundColor
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)

	require.Eventually(t, func() bool {
		lightPanelColor := evaluateScriptString(t, page, `window.getComputedStyle(document.getElementById("mp-feedback-panel")).backgroundColor`)
		return lightPanelColor == lightThemeExpectedPanelBackgroundColor
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)

	require.Eventually(t, func() bool {
		lightInputColor := evaluateScriptString(t, page, `window.getComputedStyle(document.querySelector("#mp-feedback-panel textarea")).backgroundColor`)
		return lightInputColor == lightThemeExpectedInputBackgroundColor
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)

	require.Eventually(t, func() bool {
		lightButtonColor := evaluateScriptString(t, page, `window.getComputedStyle(document.querySelector("#mp-feedback-panel button[type='button']:not([aria-label='Close feedback panel'])")).backgroundColor`)
		return lightButtonColor == lightThemeExpectedButtonBackgroundColor
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)
}

func TestWidgetCloseButtonDismissesPanel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	page := buildHeadlessPage(t)
	screenshotsDirectory := createScreenshotsDirectory(t)

	apiHarness := buildAPIHarness(t)

	server := httptest.NewServer(apiHarness.router)
	t.Cleanup(server.Close)

	site := insertSite(t, apiHarness.database, integrationSiteName, server.URL, integrationSiteOwnerEmail)

	integrationPageHTML := fmt.Sprintf(integrationPageHTMLTemplate, site.ID)
	apiHarness.router.GET(integrationPageRoutePath, func(ginContext *gin.Context) {
		ginContext.Data(http.StatusOK, integrationPageContentType, []byte(integrationPageHTML))
	})

	integrationPageURL := server.URL + integrationPageRoutePath

	navigateToPage(t, page, integrationPageURL)
	waitForVisibleElement(t, page, widgetBubbleSelector)

	clickSelector(t, page, widgetBubbleSelector)
	waitForVisibleElement(t, page, widgetPanelSelector)

	panelDisplayBeforeClose := evaluateScriptString(t, page, widgetPanelDisplayScript)
	require.Equal(t, panelDisplayBlockValue, panelDisplayBeforeClose)

	closeButtonElement := waitForVisibleElement(t, page, widgetCloseButtonSelector)
	headlineBounds := resolveViewportBounds(t, page, widgetHeadlineSelector)
	closeButtonBounds := resolveViewportBounds(t, page, widgetCloseButtonSelector)
	headlineCenter := headlineBounds.Top + (headlineBounds.Height / 2.0)
	closeButtonCenter := closeButtonBounds.Top + (closeButtonBounds.Height / 2.0)
	require.InDelta(t, headlineCenter, closeButtonCenter, closeButtonAlignmentTolerancePixels)
	closeButtonText, closeButtonTextErr := closeButtonElement.Text()
	require.NoError(t, closeButtonTextErr)
	require.Equal(t, widgetCloseButtonExpectedText, strings.TrimSpace(closeButtonText))

	closeButtonAriaLabel, closeButtonAriaLabelErr := closeButtonElement.Attribute("aria-label")
	require.NoError(t, closeButtonAriaLabelErr)
	require.NotNil(t, closeButtonAriaLabel)
	require.Equal(t, "Close feedback panel", *closeButtonAriaLabel)

	captureAndStoreScreenshot(t, page, screenshotsDirectory, "widget-panel-with-close-button")
	require.FileExists(t, filepath.Join(screenshotsDirectory, "widget-panel-with-close-button.png"))

	clickSelector(t, page, widgetCloseButtonSelector)

	var panelDisplayAfterClose string
	require.Eventually(t, func() bool {
		panelDisplayAfterClose = evaluateScriptString(t, page, widgetPanelDisplayScript)
		t.Logf("panel display after close: %q", panelDisplayAfterClose)
		return panelDisplayAfterClose == widgetPanelHiddenDisplayValue
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)
	require.Equal(t, widgetPanelHiddenDisplayValue, panelDisplayAfterClose)

	clickSelector(t, page, widgetBubbleSelector)
	waitForVisibleElement(t, page, widgetPanelSelector)

	panelDisplayAfterReopening := evaluateScriptString(t, page, widgetPanelDisplayScript)
	require.Equal(t, panelDisplayBlockValue, panelDisplayAfterReopening)
}
