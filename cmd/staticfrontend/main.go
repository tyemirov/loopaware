package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/httpapi"
)

type anonymousUserProvider struct{}

func (anonymousUserProvider) CurrentUser(*gin.Context) (*httpapi.CurrentUser, bool) {
	return nil, false
}

type renderTarget struct {
	method     string
	path       string
	handler    gin.HandlerFunc
	outputPath string
	postRender func([]byte) []byte
}

func parseDotenvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		rawValue = strings.TrimSpace(rawValue)
		if key == "" {
			continue
		}
		rawValue = strings.Trim(rawValue, "\"'")
		values[key] = rawValue
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func renderHTML(handler gin.HandlerFunc, method string, path string) (int, []byte) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, path, nil)
	handler(context)
	return recorder.Code, recorder.Body.Bytes()
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func rewriteDashboardTestPaths(payload []byte) []byte {
	replacements := map[string]string{
		`"widget_test_prefix":"/app/sites/"`:        `"widget_test_prefix":"/app/widget-test?site_id="`,
		`"widget_test_suffix":"/widget-test"`:       `"widget_test_suffix":""`,
		`"subscribe_test_prefix":"/app/sites/"`:     `"subscribe_test_prefix":"/app/subscribe-test?site_id="`,
		`"subscribe_test_suffix":"/subscribe-test"`: `"subscribe_test_suffix":""`,
		`"traffic_test_prefix":"/app/sites/"`:       `"traffic_test_prefix":"/app/traffic-test?site_id="`,
		`"traffic_test_suffix":"/traffic-test"`:     `"traffic_test_suffix":""`,
	}

	for from, to := range replacements {
		payload = bytes.ReplaceAll(payload, []byte(from), []byte(to))
	}
	return payload
}

func main() {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	var envFilePath string
	var outputDir string
	flag.StringVar(&envFilePath, "env-file", "configs/.env.loopaware.computercat", "path to a loopaware env file")
	flag.StringVar(&outputDir, "out", "public", "directory to write static assets into")
	flag.Parse()

	envValues, envErr := parseDotenvFile(envFilePath)
	if envErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "read %s: %v\n", envFilePath, envErr)
		os.Exit(1)
	}

	googleClientID := strings.TrimSpace(envValues["GOOGLE_CLIENT_ID"])
	tauthTenantID := strings.TrimSpace(envValues["TAUTH_TENANT_ID"])
	tauthBaseURL := strings.TrimSpace(envValues["TAUTH_BASE_URL"])
	apiBaseURL := strings.TrimSpace(envValues["API_BASE_URL"])
	if googleClientID == "" {
		_, _ = fmt.Fprintln(os.Stderr, "missing GOOGLE_CLIENT_ID in env file")
		os.Exit(1)
	}
	if tauthTenantID == "" {
		_, _ = fmt.Fprintln(os.Stderr, "missing TAUTH_TENANT_ID in env file")
		os.Exit(1)
	}

	authConfig := httpapi.NewAuthClientConfig(googleClientID, tauthBaseURL, tauthTenantID)

	userProvider := anonymousUserProvider{}
	landingHandlers := httpapi.NewLandingPageHandlers(logger, userProvider, authConfig, apiBaseURL)
	privacyHandlers := httpapi.NewPrivacyPageHandlers(userProvider, authConfig)
	dashboardHandlers := httpapi.NewDashboardWebHandlers(logger, httpapi.LandingPagePath, authConfig, apiBaseURL)
	subscriptionLinkHandlers := httpapi.NewSubscriptionLinkPageHandlers(logger, authConfig, apiBaseURL)
	publicJavaScriptHandlers := httpapi.NewPublicJavaScriptHandlers()

	targets := []renderTarget{
		{
			method:     http.MethodGet,
			path:       httpapi.LandingPagePath,
			handler:    landingHandlers.RenderLandingPage,
			outputPath: filepath.Join(outputDir, "login/index.html"),
		},
		{
			method:     http.MethodGet,
			path:       httpapi.PrivacyPagePath,
			handler:    privacyHandlers.RenderPrivacyPage,
			outputPath: filepath.Join(outputDir, "privacy/index.html"),
		},
		{
			method:     http.MethodGet,
			path:       "/app",
			handler:    dashboardHandlers.RenderDashboard,
			outputPath: filepath.Join(outputDir, "app/index.html"),
			postRender: rewriteDashboardTestPaths,
		},
		{
			method:     http.MethodGet,
			path:       "/subscriptions/confirm",
			handler:    subscriptionLinkHandlers.RenderConfirmSubscriptionLink,
			outputPath: filepath.Join(outputDir, "subscriptions/confirm/index.html"),
		},
		{
			method:     http.MethodGet,
			path:       "/subscriptions/unsubscribe",
			handler:    subscriptionLinkHandlers.RenderUnsubscribeSubscriptionLink,
			outputPath: filepath.Join(outputDir, "subscriptions/unsubscribe/index.html"),
		},
		{
			method:     http.MethodGet,
			path:       "/widget.js",
			handler:    publicJavaScriptHandlers.WidgetJS,
			outputPath: filepath.Join(outputDir, "widget.js"),
		},
		{
			method:     http.MethodGet,
			path:       "/subscribe.js",
			handler:    publicJavaScriptHandlers.SubscribeJS,
			outputPath: filepath.Join(outputDir, "subscribe.js"),
		},
		{
			method:     http.MethodGet,
			path:       "/pixel.js",
			handler:    publicJavaScriptHandlers.PixelJS,
			outputPath: filepath.Join(outputDir, "pixel.js"),
		},
	}

	for _, target := range targets {
		status, payload := renderHTML(target.handler, target.method, target.path)
		if status < 200 || status >= 300 {
			_, _ = fmt.Fprintf(os.Stderr, "render %s returned %d\n", target.path, status)
			os.Exit(1)
		}
		payload = bytes.ReplaceAll(payload, []byte("\r\n"), []byte("\n"))
		if target.postRender != nil {
			payload = target.postRender(payload)
		}
		if err := writeFile(target.outputPath, payload); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "write %s: %v\n", target.outputPath, err)
			os.Exit(1)
		}
	}

	fmt.Println("static frontend generated in", outputDir)
}
