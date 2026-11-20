package storage

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	browserAuditMigrationID = "historical-browser-audit-visits-v1"
	browserAuditBatchSize   = 500
)

type trafficAuditMigration struct {
	ID string `gorm:"primaryKey;size:80"`
}

type browserAuditVisitCandidate struct {
	ID        string
	UserAgent string
}

// MigrateBrowserAuditVisits reclassifies recognizable historical browser-audit visits once.
func MigrateBrowserAuditVisits(database *gorm.DB) (int64, error) {
	var changedRows int64
	migrationErr := database.Transaction(func(transaction *gorm.DB) error {
		var completionCount int64
		if err := transaction.Model(&trafficAuditMigration{}).Where("id = ?", browserAuditMigrationID).Count(&completionCount).Error; err != nil {
			return fmt.Errorf("read completion marker: %w", err)
		}
		if completionCount != 0 {
			return nil
		}

		lastVisitID := ""
		for {
			var candidates []browserAuditVisitCandidate
			if err := transaction.Model(&model.SiteVisit{}).
				Select("id", "user_agent").
				Where("id > ? AND is_bot = ?", lastVisitID, false).
				Order("id").Limit(browserAuditBatchSize).Find(&candidates).Error; err != nil {
				return fmt.Errorf("read visits after %q: %w", lastVisitID, err)
			}
			if len(candidates) == 0 {
				break
			}

			auditVisitIDs := make([]string, 0, len(candidates))
			for _, candidate := range candidates {
				if model.IsBrowserAuditUserAgent(candidate.UserAgent) {
					auditVisitIDs = append(auditVisitIDs, candidate.ID)
				}
			}
			if len(auditVisitIDs) != 0 {
				updated := transaction.Model(&model.SiteVisit{}).
					Where("id IN ? AND is_bot = ?", auditVisitIDs, false).
					UpdateColumn("is_bot", true)
				if updated.Error != nil {
					return fmt.Errorf("mark visits after %q: %w", lastVisitID, updated.Error)
				}
				changedRows += updated.RowsAffected
			}
			lastVisitID = candidates[len(candidates)-1].ID
		}

		if err := transaction.Create(&trafficAuditMigration{ID: browserAuditMigrationID}).Error; err != nil {
			return fmt.Errorf("write completion marker: %w", err)
		}
		return nil
	})
	if migrationErr != nil {
		return 0, fmt.Errorf("storage: migrate historical browser-audit visits: %w", migrationErr)
	}
	return changedRows, nil
}
