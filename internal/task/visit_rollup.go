package task

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

// VisitRollupConfig defines rollup behavior.
type VisitRollupConfig struct {
	RetentionDays int
}

// VisitRollupJob aggregates visits into daily rollups and prunes old rows.
type VisitRollupJob struct {
	database *gorm.DB
	logger   *zap.Logger
	config   VisitRollupConfig
}

// NewVisitRollupJob builds a VisitRollupJob.
func NewVisitRollupJob(database *gorm.DB, logger *zap.Logger, config VisitRollupConfig) *VisitRollupJob {
	return &VisitRollupJob{
		database: database,
		logger:   logger,
		config:   config,
	}
}

// Run executes aggregation then pruning.
func (job *VisitRollupJob) Run(ctx context.Context) error {
	if err := job.aggregatePreviousDay(ctx); err != nil {
		return err
	}
	if job.config.RetentionDays > 0 {
		return job.pruneOldVisits(ctx)
	}
	return nil
}

func (job *VisitRollupJob) aggregatePreviousDay(ctx context.Context) error {
	day := time.Now().UTC().Add(-24 * time.Hour)
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	type aggregateResult struct {
		SiteID         string
		PageViews      int64
		UniqueVisitors int64
	}
	var results []aggregateResult
	err := job.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select("site_id, COUNT(*) as page_views, COUNT(distinct visitor_id) as unique_visitors").
		Where("occurred_at >= ? AND occurred_at < ? AND is_bot = ?", start, end, false).
		Group("site_id").
		Scan(&results).Error
	if err != nil {
		return err
	}
	for _, res := range results {
		rollup, rollupErr := model.NewSiteVisitRollup(res.SiteID, start, res.PageViews, res.UniqueVisitors)
		if rollupErr != nil {
			if job.logger != nil {
				job.logger.Warn("visit_rollup_invalid", zap.Error(rollupErr), zap.String("site_id", res.SiteID))
			}
			continue
		}
		if err := job.database.WithContext(ctx).
			Where("site_id = ? AND date = ?", rollup.SiteID, rollup.Date).
			Assign(rollup).
			FirstOrCreate(&rollup).Error; err != nil && job.logger != nil {
			job.logger.Warn("visit_rollup_save_failed", zap.Error(err), zap.String("site_id", rollup.SiteID))
		}
	}
	return nil
}

func (job *VisitRollupJob) pruneOldVisits(ctx context.Context) error {
	if job.config.RetentionDays <= 0 {
		return nil
	}
	cutoff := time.Now().UTC().Add(-time.Duration(job.config.RetentionDays) * 24 * time.Hour).Truncate(24 * time.Hour)
	return job.database.WithContext(ctx).Where("occurred_at < ?", cutoff).Delete(&model.SiteVisit{}).Error
}
