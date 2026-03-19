package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestWidgetConfigMultipleAllowedOrigins(t *testing.T) {
	apiHarness := buildAPIHarness(t, nil, nil, nil)
	
	// Create a site with a primary origin and an extra widget allowed origin
	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Test Site",
		AllowedOrigin:              "https://ps.mprlab.com",
		OwnerEmail:                 "admin@example.com",
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 16,
		WidgetAllowedOrigins:       "https://poodlescanner.com",
	}
	require.NoError(t, apiHarness.database.Create(&site).Error)

	t.Run("Primary origin is allowed", func(t *testing.T) {
		resp := performJSONRequest(t, apiHarness.router, http.MethodGet, "/public/widget-config?site_id="+site.ID, nil, map[string]string{
			"Origin": "https://ps.mprlab.com",
		})
		require.Equal(t, http.StatusOK, resp.Code)
		
		var payload struct {
			SiteID string `json:"site_id"`
		}
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		require.Equal(t, site.ID, payload.SiteID)
	})

	t.Run("Secondary origin is allowed", func(t *testing.T) {
		resp := performJSONRequest(t, apiHarness.router, http.MethodGet, "/public/widget-config?site_id="+site.ID, nil, map[string]string{
			"Origin": "https://poodlescanner.com",
		})
		require.Equal(t, http.StatusOK, resp.Code, "Secondary origin should be allowed")
		
		var payload struct {
			SiteID string `json:"site_id"`
		}
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		require.Equal(t, site.ID, payload.SiteID)
	})

	t.Run("Secondary origin with trailing slash is allowed", func(t *testing.T) {
		resp := performJSONRequest(t, apiHarness.router, http.MethodGet, "/public/widget-config?site_id="+site.ID, nil, map[string]string{
			"Origin": "https://poodlescanner.com/",
		})
		require.Equal(t, http.StatusOK, resp.Code, "Origin with trailing slash should be allowed")
	})

	t.Run("Secondary origin with referer is allowed", func(t *testing.T) {
		resp := performJSONRequest(t, apiHarness.router, http.MethodGet, "/public/widget-config?site_id="+site.ID, nil, map[string]string{
			"Referer": "https://poodlescanner.com/some-page",
		})
		require.Equal(t, http.StatusOK, resp.Code, "Referer from secondary origin should be allowed")
	})

	t.Run("Secondary origin is allowed for CreateFeedback", func(t *testing.T) {
		resp := performJSONRequest(t, apiHarness.router, http.MethodPost, "/public/feedback", map[string]any{
			"site_id": site.ID,
			"contact": "test@example.com",
			"message": "test message",
		}, map[string]string{
			"Origin": "https://poodlescanner.com",
		})
		require.Equal(t, http.StatusOK, resp.Code, "Secondary origin should be allowed for feedback submission")
	})

	t.Run("Forbidden origin is blocked", func(t *testing.T) {
		resp := performJSONRequest(t, apiHarness.router, http.MethodGet, "/public/widget-config?site_id="+site.ID, nil, map[string]string{
			"Origin": "https://malicious.com",
		})
		require.Equal(t, http.StatusForbidden, resp.Code)
	})
}
