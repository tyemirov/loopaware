package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	// VisitCountsPath is the aggregate collector's public resource path.
	VisitCountsPath                  = "/public/sites/:id/visit-counts"
	visitCountsBodyLimit             = 256
	errorValueTrafficProfileConflict = "traffic_profile_conflict"
)

// NewVisitCountHandler constructs the aggregate collector with its storage and UTC clock.
func NewVisitCountHandler(database *gorm.DB, now func() time.Time) gin.HandlerFunc {
	return (&visitCountHandler{database: database, now: now}).collect
}

type visitCountHandler struct {
	database *gorm.DB
	now      func() time.Time
}

func (h *visitCountHandler) collect(context *gin.Context) {
	context.Header("Cache-Control", "no-store")
	if context.Request.URL.RawQuery != "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}
	mediaType, _, mediaErr := mime.ParseMediaType(context.GetHeader("Content-Type"))
	if mediaErr != nil || mediaType != "application/json" {
		context.JSON(http.StatusUnsupportedMediaType, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}
	body, readErr := io.ReadAll(http.MaxBytesReader(context.Writer, context.Request.Body, visitCountsBodyLimit))
	if readErr != nil {
		context.JSON(http.StatusRequestEntityTooLarge, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}
	var payload map[string]json.RawMessage
	if json.Unmarshal(body, &payload) != nil || payload == nil || len(payload) != 0 || !bytes.HasPrefix(bytes.TrimSpace(body), []byte("{")) {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}
	var site model.Site
	database := h.database.Session(&gorm.Session{Logger: logger.Discard}).WithContext(context.Request.Context())
	if err := database.First(&site, "id = ?", context.Param("id")).Error; err != nil {
		status := http.StatusInternalServerError
		code := errorValueQueryFailed
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status, code = http.StatusNotFound, errorValueUnknownSite
		}
		context.JSON(status, gin.H{jsonKeyError: code})
		return
	}
	if site.TrafficProfile != model.TrafficProfileAggregate {
		context.JSON(http.StatusConflict, gin.H{jsonKeyError: errorValueTrafficProfileConflict})
		return
	}
	allowedOrigins := mergedAllowedOrigins(site.AllowedOrigin, site.TrafficAllowedOrigins)
	if !isOriginAllowed(allowedOrigins, strings.TrimSpace(context.GetHeader("Origin")), "", "") {
		context.JSON(http.StatusForbidden, gin.H{jsonKeyError: "origin_forbidden"})
		return
	}
	// The conditional insert also enforces the current profile if it changes after lookup.
	result := database.Exec(`INSERT INTO site_visit_counts (site_id, date, count)
		SELECT id, ?, 1 FROM sites WHERE id = ? AND traffic_profile = ?
		ON CONFLICT(site_id, date) DO UPDATE SET count = count + 1`,
		h.now().UTC().Format(visitTrendDateFormat), site.ID, model.TrafficProfileAggregate)
	if result.Error != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}
	if result.RowsAffected != 1 {
		context.JSON(http.StatusConflict, gin.H{jsonKeyError: errorValueTrafficProfileConflict})
		return
	}
	context.Status(http.StatusNoContent)
}

// VisitCountHandler supplies the aggregate collector for the public API.
func (h *PublicHandlers) VisitCountHandler() gin.HandlerFunc {
	return NewVisitCountHandler(h.database, time.Now)
}
