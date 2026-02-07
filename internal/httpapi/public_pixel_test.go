package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPixelJSRequiresSiteID(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)

	response := performJSONRequest(testingT, api.router, http.MethodGet, "/pixel.js", nil, nil)
	require.Equal(testingT, http.StatusOK, response.Code)
	require.Contains(testingT, response.Header().Get("Content-Type"), "application/javascript")
	body := response.Body.String()
	require.Contains(testingT, body, "/api/visits")
	require.Contains(testingT, body, `params.get("site_id")`)
	require.NotContains(testingT, body, "%!(")
}

func TestPixelJSReturnsNotFoundForUnknownSite(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)

	response := performJSONRequest(testingT, api.router, http.MethodGet, "/pixel.js?site_id=unknown", nil, nil)
	require.Equal(testingT, http.StatusOK, response.Code)
}

func TestPixelJSReturnsScriptForKnownSite(testingT *testing.T) {
	api := buildAPIHarness(testingT, nil, nil, nil)
	site := insertSite(testingT, api.database, "Pixel Site", "https://example.com", "owner@example.com")

	response := performJSONRequest(testingT, api.router, http.MethodGet, "/pixel.js?site_id="+site.ID, nil, nil)
	require.Equal(testingT, http.StatusOK, response.Code)
	require.Contains(testingT, response.Header().Get("Content-Type"), "application/javascript")
	require.Contains(testingT, response.Body.String(), "/api/visits")
}
