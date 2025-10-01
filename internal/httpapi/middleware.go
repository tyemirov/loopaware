package httpapi

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
