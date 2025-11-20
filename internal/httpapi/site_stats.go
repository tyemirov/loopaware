package httpapi

import (
	"context"
	"strings"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"gorm.io/gorm"
)

// SiteStatisticsProvider exposes site metadata such as feedback counts.
type SiteStatisticsProvider interface {
	FeedbackCount(ctx context.Context, siteID string) (int64, error)
	SubscriberCount(ctx context.Context, siteID string) (int64, error)
}

// DatabaseSiteStatisticsProvider implements SiteStatisticsProvider using GORM.
type DatabaseSiteStatisticsProvider struct {
	database *gorm.DB
}

// NewDatabaseSiteStatisticsProvider builds a statistics provider backed by the primary database.
func NewDatabaseSiteStatisticsProvider(database *gorm.DB) *DatabaseSiteStatisticsProvider {
	return &DatabaseSiteStatisticsProvider{database: database}
}

// FeedbackCount returns the number of feedback messages for a site.
func (provider *DatabaseSiteStatisticsProvider) FeedbackCount(ctx context.Context, siteID string) (int64, error) {
	if strings.TrimSpace(siteID) == "" {
		return 0, nil
	}
	var count int64
	err := provider.database.WithContext(ctx).Model(&model.Feedback{}).Where("site_id = ?", siteID).Count(&count).Error
	return count, err
}

// SubscriberCount returns the number of subscribers for a site.
func (provider *DatabaseSiteStatisticsProvider) SubscriberCount(ctx context.Context, siteID string) (int64, error) {
	if strings.TrimSpace(siteID) == "" {
		return 0, nil
	}
	var count int64
	err := provider.database.WithContext(ctx).Model(&model.Subscriber{}).Where("site_id = ?", siteID).Count(&count).Error
	return count, err
}
