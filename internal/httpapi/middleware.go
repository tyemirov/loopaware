package httpapi

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	// AuthorizationHeaderName is the HTTP header used for bearer authentication.
	AuthorizationHeaderName = "Authorization"
	// BearerTokenPrefix prefixes bearer credentials in the Authorization header.
	BearerTokenPrefix = "Bearer "
)

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

func AdminAuthMiddleware(adminBearerToken string) gin.HandlerFunc {
	return func(context *gin.Context) {
		if adminBearerToken == "" {
			context.AbortWithStatusJSON(503, gin.H{"error": "admin disabled"})
			return
		}
		authorizationHeader := strings.TrimSpace(context.GetHeader(AuthorizationHeaderName))
		if !strings.HasPrefix(authorizationHeader, BearerTokenPrefix) {
			context.AbortWithStatusJSON(401, gin.H{"error": "missing bearer"})
			return
		}
		provided := strings.TrimPrefix(authorizationHeader, BearerTokenPrefix)
		if provided != adminBearerToken {
			context.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
			return
		}
		context.Next()
	}
}
