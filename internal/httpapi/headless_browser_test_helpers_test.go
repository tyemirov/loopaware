package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/stretchr/testify/require"
)

const (
	headlessViewportWidth                = 1280
	headlessViewportHeight               = 720
	integrationTestTimeout               = 35 * time.Second
	browserStartupTimeout                = 5 * time.Second
	headlessBrowserSkipReason            = "headless browser not available"
	headlessBrowserLocateErrorMessage    = "locate headless browser executable"
	headlessBrowserSkipMessageFormat     = "%s: %v"
	headlessBrowserEnvironmentChromedp   = "CHROMEDP_BROWSER"
	headlessBrowserEnvironmentChromePath = "CHROME_PATH"
	screenshotOutputQuality              = 90
	screenshotMinimumVariance            = 0.0005
	colorChannelTolerance                = 12.0
	colorPresenceMinimumRatio            = 0.01
	headlessBlankPageURL                 = "about:blank"
)

type viewportBounds struct {
	Left   float64 `json:"left"`
	Top    float64 `json:"top"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type screenshotExpectation struct {
	MinimumVariance float64
	ColorPresence   []colorPresenceExpectation
}

type colorPresenceExpectation struct {
	Color        rgbColor
	Tolerance    float64
	MinimumRatio float64
}

type rgbColor struct {
	Red   float64
	Green float64
	Blue  float64
}

type fetchRequestRecord struct {
	URL    string `json:"url"`
	Method string `json:"method"`
	Body   string `json:"body"`
	Status int    `json:"status"`
}

var headlessBrowserExecutableNames = []string{
	"chromium",
	"chromium-browser",
	"google-chrome",
	"google-chrome-stable",
	"chrome",
	"headless-shell",
}

var headlessBrowserLookupCache struct {
	once sync.Once
	path string
	err  error
}

var (
	headlessBrowserRuntimeFailureMutex sync.RWMutex
	headlessBrowserRuntimeFailure      error
)

var projectRootCache struct {
	once sync.Once
	path string
	err  error
}

var errHeadlessBrowserNotFound = errors.New("headless browser executable not found")

func buildHeadlessPage(testingT *testing.T) *rod.Page {
	testingT.Helper()

	if failure := loadHeadlessBrowserRuntimeFailure(); failure != nil {
		testingT.Skipf(headlessBrowserSkipMessageFormat, headlessBrowserSkipReason, failure)
	}

	browserExecutablePath, locateErr := locateHeadlessBrowserExecutable()
	if locateErr != nil {
		storeHeadlessBrowserRuntimeFailure(locateErr)
		testingT.Skipf(headlessBrowserSkipMessageFormat, headlessBrowserSkipReason, locateErr)
	}

	startupContext, startupCancel := context.WithTimeout(context.Background(), browserStartupTimeout)
	launcherInstance := launcher.New().
		Bin(browserExecutablePath).
		Context(startupContext)
	browserControlURL, launchErr := launcherInstance.Launch()
	startupCancel()
	if launchErr != nil {
		storeHeadlessBrowserRuntimeFailure(launchErr)
		testingT.Skipf(headlessBrowserSkipMessageFormat, headlessBrowserSkipReason, launchErr)
	}

	testingT.Cleanup(func() {
		launcherInstance.Cleanup()
	})

	browser := rod.New().ControlURL(browserControlURL).Timeout(integrationTestTimeout)
	connectErr := browser.Connect()
	if connectErr != nil {
		storeHeadlessBrowserRuntimeFailure(connectErr)
		testingT.Skipf(headlessBrowserSkipMessageFormat, headlessBrowserSkipReason, connectErr)
	}

	testingT.Cleanup(func() {
		closeErr := browser.Close()
		if closeErr != nil && !errors.Is(closeErr, context.Canceled) {
			storeHeadlessBrowserRuntimeFailure(closeErr)
		}
	})

	page, pageErr := browser.Page(proto.TargetCreateTarget{URL: headlessBlankPageURL})
	if pageErr != nil {
		storeHeadlessBrowserRuntimeFailure(pageErr)
		testingT.Skipf(headlessBrowserSkipMessageFormat, headlessBrowserSkipReason, pageErr)
	}

	testingT.Cleanup(func() {
		closeErr := page.Close()
		if closeErr != nil && !errors.Is(closeErr, context.Canceled) {
			storeHeadlessBrowserRuntimeFailure(closeErr)
		}
	})

	require.NoError(testingT, page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             headlessViewportWidth,
		Height:            headlessViewportHeight,
		DeviceScaleFactor: 1.0,
		Mobile:            false,
	}))
	require.NoError(testingT, page.WaitLoad())

	return page.Timeout(integrationTestTimeout)
}

func waitForVisibleElement(testingT *testing.T, page *rod.Page, selector string) *rod.Element {
	testingT.Helper()
	element, elementErr := page.Element(selector)
	require.NoError(testingT, elementErr)
	require.NoError(testingT, element.WaitVisible())
	return element
}

func navigateToPage(testingT *testing.T, page *rod.Page, targetURL string) {
	testingT.Helper()
	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	navigateErr := page.Navigate(targetURL)
	require.NoError(testingT, navigateErr)
	waitNavigation()
}

func clickSelector(testingT *testing.T, page *rod.Page, selector string) {
	testingT.Helper()
	element := waitForVisibleElement(testingT, page, selector)
	require.NoError(testingT, element.Click(proto.InputMouseButtonLeft, 1))
}

func setInputValue(testingT *testing.T, page *rod.Page, selector string, value string) {
	testingT.Helper()
	element := waitForVisibleElement(testingT, page, selector)
	require.NoError(testingT, element.SelectAllText())
	require.NoError(testingT, element.Input(""))
	require.NoError(testingT, element.Input(value))
}

func evaluateScriptInto(testingT *testing.T, page *rod.Page, script string, destination interface{}) {
	testingT.Helper()
	var wrappedScript string
	if destination == nil {
		wrappedScript = fmt.Sprintf("() => { %s; return null; }", script)
	} else {
		wrappedScript = fmt.Sprintf("() => (%s)", script)
	}
	result, evalErr := page.Eval(wrappedScript)
	require.NoError(testingT, evalErr)
	if destination == nil {
		return
	}
	valueData, marshalErr := json.Marshal(result.Value)
	require.NoError(testingT, marshalErr)
	require.NoError(testingT, json.Unmarshal(valueData, destination))
}

func evaluateScriptBoolean(testingT *testing.T, page *rod.Page, script string) bool {
	testingT.Helper()
	var resultValue bool
	evaluateScriptInto(testingT, page, script, &resultValue)
	return resultValue
}

func evaluateScriptString(testingT *testing.T, page *rod.Page, script string) string {
	testingT.Helper()
	var resultValue string
	evaluateScriptInto(testingT, page, script, &resultValue)
	return resultValue
}

func setPageCookie(testingT *testing.T, page *rod.Page, baseURL string, cookie *http.Cookie) {
	testingT.Helper()
	if cookie == nil {
		return
	}
	cookiePath := cookie.Path
	if cookiePath == "" {
		cookiePath = "/"
	}
	cookieParameters := &proto.NetworkCookieParam{
		Name:     cookie.Name,
		Value:    cookie.Value,
		URL:      baseURL,
		Path:     cookiePath,
		Secure:   cookie.Secure,
		HTTPOnly: cookie.HttpOnly,
	}
	if !cookie.Expires.IsZero() {
		cookieParameters.Expires = proto.TimeSinceEpoch(float64(cookie.Expires.UTC().Unix()))
	}
	if sameSiteMode := convertSameSiteMode(cookie.SameSite); sameSiteMode != "" {
		cookieParameters.SameSite = sameSiteMode
	}
	require.NoError(testingT, page.SetCookies([]*proto.NetworkCookieParam{cookieParameters}))
}

func convertSameSiteMode(mode http.SameSite) proto.NetworkCookieSameSite {
	switch mode {
	case http.SameSiteStrictMode:
		return proto.NetworkCookieSameSiteStrict
	case http.SameSiteLaxMode:
		return proto.NetworkCookieSameSiteLax
	case http.SameSiteNoneMode:
		return proto.NetworkCookieSameSiteNone
	default:
		return ""
	}
}

func resolveViewportBounds(testingT *testing.T, page *rod.Page, cssSelector string) viewportBounds {
	testingT.Helper()
	var bounds viewportBounds
	evaluateScriptInto(testingT, page, boundingBoxScript(cssSelector), &bounds)
	require.Greater(testingT, bounds.Width, 0.0)
	require.Greater(testingT, bounds.Height, 0.0)
	return bounds
}

func captureAndStoreScreenshot(testingT *testing.T, page *rod.Page, screenshotsDirectory string, screenshotName string) image.Image {
	testingT.Helper()
	screenshotRequest := &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
	}
	screenshotData, screenshotErr := page.Screenshot(true, screenshotRequest)
	require.NoError(testingT, screenshotErr)
	screenshotPath := filepath.Join(screenshotsDirectory, screenshotName+".png")
	require.NoError(testingT, os.WriteFile(screenshotPath, screenshotData, 0o644))
	imageReader := bytes.NewReader(screenshotData)
	decodedImage, decodeErr := png.Decode(imageReader)
	require.NoError(testingT, decodeErr)
	return decodedImage
}

func analyzeScreenshotRegion(testingT *testing.T, screenshot image.Image, bounds viewportBounds, expectation screenshotExpectation, viewportWidth float64, viewportHeight float64) {
	testingT.Helper()
	imageRectangle := convertBoundsToImageRectangle(bounds, screenshot.Bounds(), viewportWidth, viewportHeight)
	require.True(testingT, imageRectangle.Dx() > 0)
	require.True(testingT, imageRectangle.Dy() > 0)
	luminanceVariance := computeRegionLuminanceVariance(screenshot, imageRectangle)
	require.GreaterOrEqual(testingT, luminanceVariance, expectation.MinimumVariance)
	for _, presenceExpectation := range expectation.ColorPresence {
		colorRatio := computeColorPresenceRatio(screenshot, imageRectangle, presenceExpectation.Color, presenceExpectation.Tolerance)
		require.GreaterOrEqual(testingT, colorRatio, presenceExpectation.MinimumRatio)
	}
}

func evaluateFormElementFits(testingT *testing.T, page *rod.Page, cssSelector string) bool {
	testingT.Helper()
	return evaluateScriptBoolean(testingT, page, formElementFitsPanelScript(cssSelector))
}

func interceptFetchRequests(testingT *testing.T, page *rod.Page) {
	testingT.Helper()
	const script = `(function(){
  if (typeof window !== 'object' || !window) { return false; }
  if (!window.__loopawareFetchIntercept) {
    window.__loopawareFetchIntercept = { originalFetch: window.fetch, requests: [], storageKey: '__loopawareFetchRequests' };
  }
  var intercept = window.__loopawareFetchIntercept;
  intercept.requests = [];
  var storageKey = intercept.storageKey || '__loopawareFetchRequests';
  function persistRequests() {
    if (typeof sessionStorage === 'undefined') {
      return;
    }
    try {
      sessionStorage.setItem(storageKey, JSON.stringify(intercept.requests));
    } catch (error) {
      // ignore storage errors
    }
  }
  persistRequests();
  window.fetch = function(resource, init) {
    var record = { url: '', method: 'GET', body: '', status: 0 };
    if (typeof resource === 'string') {
      record.url = resource;
    } else if (resource && typeof resource.url === 'string') {
      record.url = resource.url;
      if (resource.method && typeof resource.method === 'string') {
        record.method = resource.method;
      }
    }
    if (init && typeof init.method === 'string') {
      record.method = init.method;
    }
    if (init && typeof init.body === 'string') {
      record.body = init.body;
    }
    intercept.requests.push(record);
    persistRequests();
    return intercept.originalFetch.apply(this, arguments).then(function(response) {
      if (response && typeof response.status === 'number') {
        record.status = response.status;
        persistRequests();
      }
      return response;
    }).catch(function(error) {
      record.status = 0;
      persistRequests();
      throw error;
    });
  };
  return true;
}())`
	evaluateScriptInto(testingT, page, script, nil)
}

func readCapturedFetchRequests(testingT *testing.T, page *rod.Page) []fetchRequestRecord {
	testingT.Helper()
	var records []fetchRequestRecord
	evaluateScriptInto(testingT, page, `(function(){
  var combined = [];
  if (window.__loopawareFetchIntercept && Array.isArray(window.__loopawareFetchIntercept.requests)) {
    combined = combined.concat(window.__loopawareFetchIntercept.requests);
  }
  var storageKey = '__loopawareFetchRequests';
  if (window.__loopawareFetchIntercept && typeof window.__loopawareFetchIntercept.storageKey === 'string') {
    storageKey = window.__loopawareFetchIntercept.storageKey;
  }
  if (typeof sessionStorage !== 'undefined') {
    try {
      var stored = sessionStorage.getItem(storageKey);
      if (stored) {
        var parsed = JSON.parse(stored);
        if (Array.isArray(parsed)) {
          combined = combined.concat(parsed);
        }
      }
    } catch (error) {
      // ignore parse/storage errors
    }
  }
  return combined;
}())`, &records)
	return records
}

func convertBoundsToImageRectangle(bounds viewportBounds, imageBounds image.Rectangle, viewportWidth float64, viewportHeight float64) image.Rectangle {
	scaleX := float64(imageBounds.Dx()) / viewportWidth
	scaleY := float64(imageBounds.Dy()) / viewportHeight
	minX := int(math.Floor(bounds.Left * scaleX))
	minY := int(math.Floor(bounds.Top * scaleY))
	maxX := int(math.Ceil((bounds.Left + bounds.Width) * scaleX))
	maxY := int(math.Ceil((bounds.Top + bounds.Height) * scaleY))
	if minX < imageBounds.Min.X {
		minX = imageBounds.Min.X
	}
	if minY < imageBounds.Min.Y {
		minY = imageBounds.Min.Y
	}
	if maxX > imageBounds.Max.X {
		maxX = imageBounds.Max.X
	}
	if maxY > imageBounds.Max.Y {
		maxY = imageBounds.Max.Y
	}
	if maxX <= minX {
		maxX = minX + 1
		if maxX > imageBounds.Max.X {
			maxX = imageBounds.Max.X
			minX = maxX - 1
		}
	}
	if maxY <= minY {
		maxY = minY + 1
		if maxY > imageBounds.Max.Y {
			maxY = imageBounds.Max.Y
			minY = maxY - 1
		}
	}
	return image.Rect(minX, minY, maxX, maxY)
}

func computeRegionLuminanceVariance(source image.Image, region image.Rectangle) float64 {
	pixelCount := region.Dx() * region.Dy()
	if pixelCount <= 0 {
		return 0
	}
	var luminanceSum float64
	var luminanceSquaredSum float64
	for coordinateY := region.Min.Y; coordinateY < region.Max.Y; coordinateY++ {
		for coordinateX := region.Min.X; coordinateX < region.Max.X; coordinateX++ {
			redComponent, greenComponent, blueComponent, _ := source.At(coordinateX, coordinateY).RGBA()
			luminance := computeLuminance(redComponent, greenComponent, blueComponent)
			luminanceSum += luminance
			luminanceSquaredSum += luminance * luminance
		}
	}
	meanLuminance := luminanceSum / float64(pixelCount)
	variance := (luminanceSquaredSum / float64(pixelCount)) - (meanLuminance * meanLuminance)
	if variance < 0 {
		return 0
	}
	return variance
}

func computeLuminance(red uint32, green uint32, blue uint32) float64 {
	normalizedRed := float64(red) / 65535.0
	normalizedGreen := float64(green) / 65535.0
	normalizedBlue := float64(blue) / 65535.0
	return (0.2126 * normalizedRed) + (0.7152 * normalizedGreen) + (0.0722 * normalizedBlue)
}

func computeColorPresenceRatio(source image.Image, region image.Rectangle, target rgbColor, tolerance float64) float64 {
	pixelCount := region.Dx() * region.Dy()
	if pixelCount <= 0 {
		return 0
	}
	var matchingPixels int
	for coordinateY := region.Min.Y; coordinateY < region.Max.Y; coordinateY++ {
		for coordinateX := region.Min.X; coordinateX < region.Max.X; coordinateX++ {
			actualColor := extractRGBComponents(source.At(coordinateX, coordinateY))
			if math.Abs(actualColor.Red-target.Red) <= tolerance &&
				math.Abs(actualColor.Green-target.Green) <= tolerance &&
				math.Abs(actualColor.Blue-target.Blue) <= tolerance {
				matchingPixels++
			}
		}
	}
	return float64(matchingPixels) / float64(pixelCount)
}

func extractRGBComponents(value color.Color) rgbColor {
	redComponent, greenComponent, blueComponent, _ := value.RGBA()
	return rgbColor{
		Red:   float64(redComponent) / 257.0,
		Green: float64(greenComponent) / 257.0,
		Blue:  float64(blueComponent) / 257.0,
	}
}

func boundingBoxScript(cssSelector string) string {
	return fmt.Sprintf(`(function(selector){
		var element = document.querySelector(selector);
		if (!element) { return null; }
		var rect = element.getBoundingClientRect();
		return { left: rect.left, top: rect.top, width: rect.width, height: rect.height };
	})(%q)`, cssSelector)
}

func mustParseRGBColor(testingT *testing.T, value string) rgbColor {
	testingT.Helper()
	parsedColor, parseErr := parseRGBColor(value)
	require.NoError(testingT, parseErr)
	return parsedColor
}

func parseRGBColor(value string) (rgbColor, error) {
	var red float64
	var green float64
	var blue float64
	_, scanErr := fmt.Sscanf(value, "rgb(%f, %f, %f)", &red, &green, &blue)
	if scanErr != nil {
		return rgbColor{}, scanErr
	}
	return rgbColor{
		Red:   red,
		Green: green,
		Blue:  blue,
	}, nil
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

func createScreenshotsDirectory(testingT *testing.T) string {
	testingT.Helper()
	rootDirectory, rootErr := resolveProjectRootDirectory()
	require.NoError(testingT, rootErr)
	dateSegment := time.Now().UTC().Format("2006-01-02")
	testNameSegment := sanitizeTestName(testingT.Name())
	baseDirectory := filepath.Join(rootDirectory, "tests", dateSegment, testNameSegment)
	require.NoError(testingT, os.MkdirAll(baseDirectory, 0o755))
	return baseDirectory
}

func sanitizeTestName(name string) string {
	if name == "" {
		return "unnamed"
	}
	var builder strings.Builder
	builder.Grow(len(name))
	for _, character := range name {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_' {
			builder.WriteRune(character)
			continue
		}
		builder.WriteRune('_')
	}
	return builder.String()
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

	downloadedPath, downloadErr := downloadHeadlessBrowserExecutable()
	if downloadErr == nil && downloadedPath != "" {
		return downloadedPath, nil
	}

	if downloadErr != nil {
		return "", fmt.Errorf("%s: %w (auto download failed: %v)", headlessBrowserLocateErrorMessage, errHeadlessBrowserNotFound, downloadErr)
	}

	return "", fmt.Errorf("%s: %w", headlessBrowserLocateErrorMessage, errHeadlessBrowserNotFound)
}

func downloadHeadlessBrowserExecutable() (string, error) {
	browser := launcher.NewBrowser()
	path, err := browser.Get()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("launcher returned empty browser path")
	}
	return path, nil
}

func resolveProjectRootDirectory() (string, error) {
	projectRootCache.once.Do(func() {
		if gomodPath := strings.TrimSpace(os.Getenv("GOMOD")); gomodPath != "" {
			projectRootCache.path = filepath.Dir(gomodPath)
			return
		}
		if output, err := exec.Command("go", "env", "GOMOD").Output(); err == nil {
			gomodPath := strings.TrimSpace(string(output))
			if gomodPath != "" {
				projectRootCache.path = filepath.Dir(gomodPath)
				return
			}
		}
		dir, err := os.Getwd()
		if err != nil {
			projectRootCache.err = err
			return
		}
		for {
			candidate := filepath.Join(dir, "go.mod")
			if _, statErr := os.Stat(candidate); statErr == nil {
				projectRootCache.path = dir
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				projectRootCache.err = fmt.Errorf("go.mod not found from %s", dir)
				return
			}
			dir = parent
		}
	})
	if projectRootCache.err != nil {
		return "", projectRootCache.err
	}
	if projectRootCache.path == "" {
		return "", errors.New("project root path not resolved")
	}
	return projectRootCache.path, nil
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
