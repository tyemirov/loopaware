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
	VisitCount(ctx context.Context, siteID string) (int64, error)
	UniqueVisitorCount(ctx context.Context, siteID string) (int64, error)
	TopPages(ctx context.Context, siteID string, limit int) ([]TopPageStat, error)
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

// VisitCount returns total page views for a site.
func (provider *DatabaseSiteStatisticsProvider) VisitCount(ctx context.Context, siteID string) (int64, error) {
	if strings.TrimSpace(siteID) == "" {
		return 0, nil
	}
	var count int64
	err := provider.database.WithContext(ctx).Model(&model.SiteVisit{}).Where("site_id = ?", siteID).Count(&count).Error
	return count, err
}

// UniqueVisitorCount returns distinct visitor ids for a site.
func (provider *DatabaseSiteStatisticsProvider) UniqueVisitorCount(ctx context.Context, siteID string) (int64, error) {
	if strings.TrimSpace(siteID) == "" {
		return 0, nil
	}
	var count int64
	err := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Where("site_id = ? AND visitor_id <> ''", siteID).
		Distinct("visitor_id").
		Count(&count).Error
	return count, err
}

// TopPageStat captures per-page view counts.
type TopPageStat struct {
	Path       string
	VisitCount int64
}

// TopPages returns top pages by visit count.
func (provider *DatabaseSiteStatisticsProvider) TopPages(ctx context.Context, siteID string, limit int) ([]TopPageStat, error) {
	if strings.TrimSpace(siteID) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}
	var results []TopPageStat
	err := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select("path, COUNT(*) as visit_count").
		Where("site_id = ? AND path <> ''", siteID).
		Group("path").
		Order("visit_count desc").
		Limit(limit).
		Scan(&results).Error
	return results, err
}
