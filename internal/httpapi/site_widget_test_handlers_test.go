package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveWidgetScriptURLUsesPrimaryForwardedProto(testingT *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://example.com/app/sites/site-123/widget-test", nil)
	request.Header.Set("X-Forwarded-Proto", "https,http")

	handlers := &SiteWidgetTestHandlers{}

	resolvedURL := handlers.resolveWidgetScriptURL(request, "site-123")

	require.Equal(testingT, "https://example.com/widget.js?site_id=site-123", resolvedURL)
}

func TestResolveWidgetScriptURLFallsBackToWidgetBaseURL(testingT *testing.T) {
	handlers := &SiteWidgetTestHandlers{widgetBaseURL: "https://widgets.loopaware.test"}

	resolvedURL := handlers.resolveWidgetScriptURL(nil, "site-456")

	require.Equal(testingT, "https://widgets.loopaware.test/widget.js?site_id=site-456", resolvedURL)
}
