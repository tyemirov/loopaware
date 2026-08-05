package api

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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
	require.NoError(testingT, router.SetTrustedProxies([]string{"198.51.100.10/32"}))
	router.RemoteIPHeaders = []string{"X-Forwarded-For"}
	router.Use(TrustedProxyHeaders([]netip.Prefix{netip.MustParsePrefix("198.51.100.10/32")}, nil))
	router.Use(SecurityHeaders())
	router.GET("/health", func(context *gin.Context) {
		context.String(http.StatusOK, "ok")
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.RemoteAddr = "198.51.100.10:443"
	request.Header.Set("X-Forwarded-Proto", "https")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(testingT, securityValueStrictTransportSecurity, recorder.Header().Get(securityHeaderStrictTransportSecurity))
}

func TestSecurityHeadersAddsHSTSForForwardedHeader(testingT *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	require.NoError(testingT, router.SetTrustedProxies([]string{"198.51.100.10/32"}))
	router.RemoteIPHeaders = []string{"X-Forwarded-For"}
	router.Use(TrustedProxyHeaders([]netip.Prefix{netip.MustParsePrefix("198.51.100.10/32")}, nil))
	router.Use(SecurityHeaders())
	router.GET("/health", func(context *gin.Context) {
		context.String(http.StatusOK, "ok")
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.RemoteAddr = "198.51.100.10:443"
	request.Header.Set(headerForwarded, "for=203.0.113.43;proto=https;by=203.0.113.44")
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

func TestTrustedProxyHeadersRejectsDirectClientSpoofing(testingT *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	require.NoError(testingT, router.SetTrustedProxies([]string{"198.51.100.10/32"}))
	router.RemoteIPHeaders = []string{"X-Forwarded-For"}
	router.Use(TrustedProxyHeaders(
		[]netip.Prefix{netip.MustParsePrefix("198.51.100.10/32")},
		[]netip.Prefix{netip.MustParsePrefix("198.51.100.10/32")},
	))
	var clientIP string
	var forwardedProto string
	var geoCountry string
	router.GET("/identity", func(context *gin.Context) {
		clientIP = context.ClientIP()
		forwardedProto = context.GetHeader("X-Forwarded-Proto")
		geoCountry = context.GetHeader("CF-IPCountry")
		context.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/identity", nil)
	request.RemoteAddr = "203.0.113.7:4321"
	request.Header.Set("X-Forwarded-For", "192.0.2.44")
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("CF-IPCountry", "US")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusNoContent, recorder.Code)
	require.Equal(testingT, "203.0.113.7", clientIP)
	require.Empty(testingT, forwardedProto)
	require.Empty(testingT, geoCountry)
}

func TestTrustedProxyHeadersAcceptsOwnedProxyMetadata(testingT *testing.T) {
	gin.SetMode(gin.TestMode)

	trustedProxy := netip.MustParsePrefix("198.51.100.10/32")
	router := gin.New()
	require.NoError(testingT, router.SetTrustedProxies([]string{trustedProxy.String()}))
	router.RemoteIPHeaders = []string{"X-Forwarded-For"}
	router.Use(TrustedProxyHeaders([]netip.Prefix{trustedProxy}, []netip.Prefix{trustedProxy}))
	var clientIP string
	var geoCountry string
	router.GET("/identity", func(context *gin.Context) {
		clientIP = context.ClientIP()
		geoCountry = context.GetHeader("CF-IPCountry")
		context.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/identity", nil)
	request.RemoteAddr = "198.51.100.10:443"
	request.Header.Set("X-Forwarded-For", "203.0.113.8")
	request.Header.Set("CF-IPCountry", "US")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusNoContent, recorder.Code)
	require.Equal(testingT, "203.0.113.8", clientIP)
	require.Equal(testingT, "US", geoCountry)
}

func TestRequestLoggerRemovesLineBreaksAndBoundsRequestFields(testingT *testing.T) {
	gin.SetMode(gin.TestMode)

	observedCore, observedLogs := observer.New(zap.InfoLevel)
	router := gin.New()
	router.Use(RequestLogger(zap.New(observedCore)))
	router.Any("/*path", func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/safe%0Aforged", nil)
	request.RemoteAddr = "203.0.113.7:4321"
	request.Header.Set("User-Agent", strings.Repeat("browser", 700)+"\r\nforged")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusNoContent, recorder.Code)
	entries := observedLogs.All()
	require.Len(testingT, entries, 1)
	fields := entries[0].ContextMap()
	for _, name := range []string{"method", "path", "ip", "ua"} {
		value, valueOK := fields[name].(string)
		require.True(testingT, valueOK, "expected string log field %s", name)
		require.NotContains(testingT, value, "\r")
		require.NotContains(testingT, value, "\n")
		require.LessOrEqual(testingT, len(value), requestLogValueMaximumBytes)
	}
}
