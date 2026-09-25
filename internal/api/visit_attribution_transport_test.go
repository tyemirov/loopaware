package api_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/stretchr/testify/require"
)

func TestCollectVisitPreservesPageReferral(testingT *testing.T) {
	for _, scenario := range []struct{ name, referrer string }{
		{name: "external", referrer: "https://www.linkedin.com/feed/"},
		{name: "direct", referrer: ""},
	} {
		testingT.Run(scenario.name, func(testingT *testing.T) {
			harness := buildAPIHarness(testingT, nil, nil, nil)
			site := insertSite(testingT, harness.database, "Marketing attribution", testVisitOrigin, "owner@example.com")
			server := httptest.NewServer(harness.router)
			defer server.Close()
			landingURL := testVisitOriginPage + "?utm_source=mprlab&utm_medium=social&utm_campaign=ps-t01"
			query := url.Values{"site_id": {site.ID}, "url": {landingURL}, "referrer": {scenario.referrer}}
			request, err := http.NewRequest(http.MethodGet, server.URL+testVisitPath+"?"+query.Encode(), nil)
			require.NoError(testingT, err)
			request.Header.Set("Referer", testVisitOrigin+"/")
			response, err := server.Client().Do(request)
			require.NoError(testingT, err)
			defer response.Body.Close()
			_, err = io.Copy(io.Discard, response.Body)
			require.NoError(testingT, err)
			require.Equal(testingT, http.StatusOK, response.StatusCode)
			var stored model.SiteVisit
			require.NoError(testingT, harness.database.Where("site_id = ?", site.ID).First(&stored).Error)
			require.Equal(testingT, scenario.referrer, stored.Referrer)
			require.Equal(testingT, landingURL, stored.URL)
		})
	}
}
