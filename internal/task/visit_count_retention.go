package task

import (
	"context"
	"fmt"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"gorm.io/gorm"
)

// PruneVisitCounts removes dates outside the configured calendar-day window.
// The current UTC date is day one of the window.
func PruneVisitCounts(ctx context.Context, database *gorm.DB, now time.Time, days int) error {
	cutoff := now.UTC().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	if err := database.WithContext(ctx).Where("date < ?", cutoff).Delete(&model.SiteVisitCount{}).Error; err != nil {
		return fmt.Errorf("prune daily visit counts: %w", err)
	}
	return nil
}
