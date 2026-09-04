package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HealthPath is the public readiness resource.
const HealthPath = "/healthz"

// Health checks local datastore access without customer or provider activity.
func (h *PublicHandlers) Health(request *gin.Context) {
	request.Header("Cache-Control", "no-store")
	probeContext, cancel := context.WithTimeout(request.Request.Context(), time.Second)
	defer cancel()
	if err := h.database.WithContext(probeContext).Exec("SELECT 1").Error; err != nil {
		h.logger.Error("health_probe_failed", zap.Error(err))
		request.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}
	request.JSON(http.StatusOK, gin.H{"status": "ok"})
}
