package api

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSecurityHeadersAddsHardeningHeaders(testingT *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/health", func(context *gin.Context) {
		context.String(http.StatusOK, "ok")
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(testingT, securityValueContentSecurityPolicy, recorder.Header().Get(securityHeaderContentSecurityPolicy))
	require.Equal(testingT, securityValuePermissionsPolicy, recorder.Header().Get(securityHeaderPermissionsPolicy))
	require.Equal(testingT, securityValueReferrerPolicy, recorder.Header().Get(securityHeaderReferrerPolicy))
	require.Equal(testingT, securityValueXContentTypeOptions, recorder.Header().Get(securityHeaderXContentTypeOptions))
	require.Equal(testingT, securityValueXFrameOptions, recorder.Header().Get(securityHeaderXFrameOptions))
	require.Equal(testingT, securityValueXPermittedCrossDomainRules, recorder.Header().Get(securityHeaderXPermittedCrossDomainRules))
	require.Empty(testingT, recorder.Header().Get(securityHeaderStrictTransportSecurity))
}

func TestSecurityHeadersAddsHSTSForForwardedHTTPS(testingT *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/health", func(context *gin.Context) {
		context.String(http.StatusOK, "ok")
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(testingT, securityValueStrictTransportSecurity, recorder.Header().Get(securityHeaderStrictTransportSecurity))
}

func TestSecurityHeadersAddsHSTSForDirectTLS(testingT *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/health", func(context *gin.Context) {
		context.String(http.StatusOK, "ok")
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.TLS = &tls.ConnectionState{}
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(testingT, securityValueStrictTransportSecurity, recorder.Header().Get(securityHeaderStrictTransportSecurity))
}
