package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidVisitRollup = errors.New("invalid_visit_rollup")
)

// SiteVisitRollup captures aggregated visit metrics per day.
type SiteVisitRollup struct {
	ID             string    `gorm:"primaryKey;size:36"`
	SiteID         string    `gorm:"not null;size:36;index"`
	Date           time.Time `gorm:"not null;index"` // UTC date truncated to midnight
	PageViews      int64     `gorm:"not null"`
	UniqueVisitors int64     `gorm:"not null"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

// NewSiteVisitRollup constructs a rollup for a specific date.
func NewSiteVisitRollup(siteID string, date time.Time, pageViews int64, uniqueVisitors int64) (SiteVisitRollup, error) {
	trimmedSiteID := strings.TrimSpace(siteID)
	if trimmedSiteID == "" {
		return SiteVisitRollup{}, fmt.Errorf("%w: missing site_id", ErrInvalidVisitRollup)
	}
	if date.IsZero() {
		return SiteVisitRollup{}, fmt.Errorf("%w: missing date", ErrInvalidVisitRollup)
	}
	if pageViews < 0 || uniqueVisitors < 0 {
		return SiteVisitRollup{}, fmt.Errorf("%w: negative counts", ErrInvalidVisitRollup)
	}
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	return SiteVisitRollup{
		ID:             uuid.NewString(),
		SiteID:         trimmedSiteID,
		Date:           normalizedDate,
		PageViews:      pageViews,
		UniqueVisitors: uniqueVisitors,
	}, nil
}
