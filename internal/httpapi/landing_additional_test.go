package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/httpapi"
)

func TestNewLandingPageHandlersDefaultsLogger(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, httpapi.LandingPagePath, nil)

	handlers := httpapi.NewLandingPageHandlers(nil, &stubCurrentUserProvider{}, testLandingAuthConfig, "")
	handlers.RenderLandingPage(ginContext)

	require.Equal(testingT, http.StatusOK, recorder.Code)
}
