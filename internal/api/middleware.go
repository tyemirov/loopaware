package api

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	securityHeaderContentSecurityPolicy      = "Content-Security-Policy"
	securityHeaderPermissionsPolicy          = "Permissions-Policy"
	securityHeaderReferrerPolicy             = "Referrer-Policy"
	securityHeaderStrictTransportSecurity    = "Strict-Transport-Security"
	securityHeaderXContentTypeOptions        = "X-Content-Type-Options"
	securityHeaderXFrameOptions              = "X-Frame-Options"
	securityHeaderXPermittedCrossDomainRules = "X-Permitted-Cross-Domain-Policies"
	securityValueContentSecurityPolicy       = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"
	securityValuePermissionsPolicy           = "accelerometer=(), autoplay=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()"
	securityValueReferrerPolicy              = "strict-origin-when-cross-origin"
	securityValueStrictTransportSecurity     = "max-age=31536000; includeSubDomains"
	securityValueXContentTypeOptions         = "nosniff"
	securityValueXFrameOptions               = "DENY"
	securityValueXPermittedCrossDomainRules  = "none"
)

func SecurityHeaders() gin.HandlerFunc {
	return func(context *gin.Context) {
		headers := context.Writer.Header()
		headers.Set(securityHeaderContentSecurityPolicy, securityValueContentSecurityPolicy)
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

func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(context *gin.Context) {
		start := time.Now()
		context.Next()
		logger.Info("http",
			zap.String("method", context.Request.Method),
			zap.String("path", context.Request.URL.Path),
			zap.Int("status", context.Writer.Status()),
			zap.Duration("dur", time.Since(start)),
			zap.String("ip", context.ClientIP()),
			zap.String("ua", context.Request.UserAgent()),
		)
	}
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
		return false
	}
	return strings.EqualFold(strings.TrimSpace(strings.Split(forwardedProto, ",")[0]), "https")
}
