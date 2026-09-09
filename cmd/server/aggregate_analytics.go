package main

import (
	"context"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/task"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func startVisitCountRetention(database *gorm.DB, logger *zap.Logger, days int, now func() time.Time) (func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	if err := task.PruneVisitCounts(ctx, database, now(), days); err != nil {
		cancel()
		return nil, err
	}
	ticker := time.NewTicker(24 * time.Hour)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := task.PruneVisitCounts(ctx, database, now(), days); err != nil {
					logger.Error("visit_count_retention_failed", zap.Error(err))
				}
			}
		}
	}()
	return func() { ticker.Stop(); cancel(); <-done }, nil
}
