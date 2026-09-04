package api

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var forwardedRequestHeaders = [...]string{
	headerForwarded,
	"X-Forwarded-For",
	"X-Forwarded-Host",
	"X-Forwarded-Port",
	"X-Forwarded-Proto",
	"X-Real-IP",
}

var edgeGeoRequestHeaders = [...]string{
	"CF-IPCountry",
	"CF-Region-Code",
	"CF-Region",
	"CF-IPCity",
	"CF-IPLatitude",
	"CF-IPLongitude",
	"X-Vercel-IP-Country",
	"X-Vercel-IP-Country-Region",
	"X-Vercel-IP-City",
	"X-Vercel-IP-Latitude",
	"X-Vercel-IP-Longitude",
	"CloudFront-Viewer-Country",
}

const (
	requestBodyReadErrorValue                = "invalid_request_body"
	requestBodyTooLargeErrorValue            = "request_too_large"
	requestLogValueMaximumBytes              = 4096
	securityHeaderPermissionsPolicy          = "Permissions-Policy"
	securityHeaderReferrerPolicy             = "Referrer-Policy"
	securityHeaderStrictTransportSecurity    = "Strict-Transport-Security"
	securityHeaderXContentTypeOptions        = "X-Content-Type-Options"
	securityHeaderXFrameOptions              = "X-Frame-Options"
	securityHeaderXPermittedCrossDomainRules = "X-Permitted-Cross-Domain-Policies"
	securityValuePermissionsPolicy           = "accelerometer=(), autoplay=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()"
	securityValueReferrerPolicy              = "strict-origin-when-cross-origin"
	securityValueStrictTransportSecurity     = "max-age=31536000; includeSubDomains"
	securityValueXContentTypeOptions         = "nosniff"
	securityValueXFrameOptions               = "DENY"
	securityValueXPermittedCrossDomainRules  = "none"
)

// RequestBodyLimitResolver returns the maximum accepted body size for a request.
type RequestBodyLimitResolver func(*http.Request) int64

// RequestBodyLimit reads request bodies through a route-specific upper bound before dispatch.
func RequestBodyLimit(resolveLimit RequestBodyLimitResolver) gin.HandlerFunc {
	return func(context *gin.Context) {
		request := context.Request
		if request.Body == nil || request.Body == http.NoBody {
			context.Next()
			return
		}

		maximumBytes := resolveLimit(request)
		if request.ContentLength > maximumBytes {
			_ = request.Body.Close()
			context.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": requestBodyTooLargeErrorValue})
			return
		}

		limitedBody := http.MaxBytesReader(context.Writer, request.Body, maximumBytes)
		body, readError := io.ReadAll(limitedBody)
		closeError := limitedBody.Close()
		if readError == nil {
			readError = closeError
		}
		if readError != nil {
			var maximumBytesError *http.MaxBytesError
			if errors.As(readError, &maximumBytesError) {
				context.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": requestBodyTooLargeErrorValue})
				return
			}
			context.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": requestBodyReadErrorValue})
			return
		}

		request.Body = io.NopCloser(bytes.NewReader(body))
		request.ContentLength = int64(len(body))
		context.Next()
	}
}

func SecurityHeaders() gin.HandlerFunc {
	return func(context *gin.Context) {
		headers := context.Writer.Header()
		headers.Set(securityHeaderPermissionsPolicy, securityValuePermissionsPolicy)
		headers.Set(securityHeaderReferrerPolicy, securityValueReferrerPolicy)
		headers.Set(securityHeaderXContentTypeOptions, securityValueXContentTypeOptions)
		headers.Set(securityHeaderXFrameOptions, securityValueXFrameOptions)
		headers.Set(securityHeaderXPermittedCrossDomainRules, securityValueXPermittedCrossDomainRules)
		if requestUsesHTTPS(context) {
			headers.Set(securityHeaderStrictTransportSecurity, securityValueStrictTransportSecurity)
		}
		context.Next()
	}
}

// TrustedProxyHeaders removes identity and edge metadata received outside their configured proxy boundaries.
func TrustedProxyHeaders(trustedProxyCIDRs []netip.Prefix, trustedEdgeGeoProxyCIDRs []netip.Prefix) gin.HandlerFunc {
	return func(context *gin.Context) {
		peerAddress, peerAddressOK := immediatePeerAddress(context)
		if !peerAddressOK || !addressWithinAnyPrefix(peerAddress, trustedProxyCIDRs) {
			deleteRequestHeaders(context, forwardedRequestHeaders[:])
		}
		if !peerAddressOK || !addressWithinAnyPrefix(peerAddress, trustedEdgeGeoProxyCIDRs) {
			deleteRequestHeaders(context, edgeGeoRequestHeaders[:])
		}
		context.Next()
	}
}

func immediatePeerAddress(context *gin.Context) (netip.Addr, bool) {
	if context == nil || context.Request == nil {
		return netip.Addr{}, false
	}
	host, _, splitError := net.SplitHostPort(strings.TrimSpace(context.Request.RemoteAddr))
	if splitError != nil {
		return netip.Addr{}, false
	}
	address, parseError := netip.ParseAddr(host)
	if parseError != nil {
		return netip.Addr{}, false
	}
	return address.Unmap(), true
}

func addressWithinAnyPrefix(address netip.Addr, prefixes []netip.Prefix) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func deleteRequestHeaders(context *gin.Context, headerNames []string) {
	if context == nil || context.Request == nil {
		return
	}
	for _, headerName := range headerNames {
		context.Request.Header.Del(headerName)
	}
}

func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(context *gin.Context) {
		start := time.Now()
		context.Next()
		if context.Request.URL.Path == HealthPath && context.Writer.Status() == http.StatusOK {
			return
		}
		logger.Info("http",
			zap.String("method", sanitizeRequestLogValue(context.Request.Method)),
			zap.String("path", sanitizeRequestLogValue(context.Request.URL.Path)),
			zap.Int("status", context.Writer.Status()),
			zap.Duration("dur", time.Since(start)),
			zap.String("ip", sanitizeRequestLogValue(context.ClientIP())),
			zap.String("ua", sanitizeRequestLogValue(context.Request.UserAgent())),
		)
	}
}

func sanitizeRequestLogValue(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ToValidUTF8(value, "\uFFFD")
	if len(value) <= requestLogValueMaximumBytes {
		return value
	}
	end := requestLogValueMaximumBytes
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}

func requestUsesHTTPS(context *gin.Context) bool {
	if context == nil || context.Request == nil {
		return false
	}
	if context.Request.TLS != nil {
		return true
	}
	forwardedProto := strings.TrimSpace(context.GetHeader("X-Forwarded-Proto"))
	if forwardedProto == "" {
		return parseForwardedProtoHeaderValue(strings.TrimSpace(context.GetHeader(headerForwarded))) == urlSchemeHTTPS
	}
	return strings.EqualFold(strings.TrimSpace(strings.Split(forwardedProto, ",")[0]), "https")
}
