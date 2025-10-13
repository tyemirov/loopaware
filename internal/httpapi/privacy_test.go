package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/httpapi"
)

const (
	privacyTitleToken          = "<title>Privacy Policy — LoopAware</title>"
	privacyHeadingToken        = "<h1 class=\"privacy-heading\">Privacy Policy — LoopAware</h1>"
	privacyEffectiveDateToken  = "<strong>Effective Date:</strong> 2025-10-11"
	privacyMetaRobotsToken     = "<meta name=\"robots\" content=\"noindex,nofollow\" />"
	privacySupportEmailToken   = "mailto:support@mprlab.com"
	privacyBodyFontStyleToken  = "body{font:16px/1.5 system-ui,Segoe UI,Roboto,Helvetica,Arial,sans-serif"
	privacyGeneratorCommentTag = "<!doctype html>"
	privacyFooterLayoutToken   = "footer-layout"
	privacyFooterBrandToken    = "footer-brand d-inline-flex align-items-center"
	privacyFooterPrivacyToken  = "footer-privacy-link text-body-secondary text-decoration-none small"
)

func TestPrivacyPageRendersPolicyMarkup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/privacy", nil)

	handlers := httpapi.NewPrivacyPageHandlers()
	handlers.RenderPrivacyPage(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/html")
	body := recorder.Body.String()
	require.Contains(t, body, privacyGeneratorCommentTag)
	require.Contains(t, body, privacyTitleToken)
	require.Contains(t, body, privacyHeadingToken)
	require.Contains(t, body, privacyEffectiveDateToken)
	require.Contains(t, body, privacyMetaRobotsToken)
	require.Contains(t, body, privacySupportEmailToken)
	require.Contains(t, body, privacyBodyFontStyleToken)
	require.Contains(t, body, privacyFooterLayoutToken)
	require.Contains(t, body, privacyFooterBrandToken)
	require.Contains(t, body, privacyFooterPrivacyToken)
	require.Contains(t, body, dashboardFooterBrandPrefix)
	require.Contains(t, body, dashboardFooterBrandName)
}
