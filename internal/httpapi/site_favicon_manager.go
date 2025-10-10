package httpapi

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
)

const (
	defaultFaviconRetryInterval = 30 * time.Minute
)

type SiteFaviconManager struct {
	database      *gorm.DB
	resolver      FaviconResolver
	logger        *zap.Logger
	retryInterval time.Duration
	inFlight      sync.Map
	now           func() time.Time
}

func NewSiteFaviconManager(database *gorm.DB, resolver FaviconResolver, logger *zap.Logger) *SiteFaviconManager {
	manager := &SiteFaviconManager{
		database:      database,
		resolver:      resolver,
		logger:        logger,
		retryInterval: defaultFaviconRetryInterval,
		now:           time.Now,
	}
	return manager
}

func (manager *SiteFaviconManager) ScheduleFetch(site model.Site) {
	if manager == nil || manager.resolver == nil || manager.database == nil {
		return
	}

	normalizedOrigin := strings.TrimSpace(site.AllowedOrigin)
	if normalizedOrigin == "" {
		return
	}

	if !manager.shouldFetch(site, normalizedOrigin) {
		return
	}

	if _, alreadyInFlight := manager.inFlight.LoadOrStore(site.ID, struct{}{}); alreadyInFlight {
		return
	}

	go manager.fetchAndStore(site.ID, normalizedOrigin)
}

func (manager *SiteFaviconManager) shouldFetch(site model.Site, normalizedOrigin string) bool {
	storedOrigin := strings.TrimSpace(site.FaviconOrigin)
	if storedOrigin == "" {
		return true
	}
	if !strings.EqualFold(storedOrigin, normalizedOrigin) {
		return true
	}
	if len(site.FaviconData) == 0 {
		if site.FaviconLastAttemptAt.IsZero() {
			return true
		}
		return manager.now().Sub(site.FaviconLastAttemptAt) >= manager.retryInterval
	}
	return false
}

func (manager *SiteFaviconManager) fetchAndStore(siteID string, normalizedOrigin string) {
	defer manager.inFlight.Delete(siteID)

	ctx := context.Background()
	asset, resolveErr := manager.resolver.ResolveAsset(ctx, normalizedOrigin)
	now := manager.now()

	updates := map[string]any{
		"favicon_origin":          normalizedOrigin,
		"favicon_last_attempt_at": now,
	}

	if resolveErr != nil {
		if manager.logger != nil {
			manager.logger.Debug(
				"fetch_site_favicon_failed",
				zap.String("site_id", siteID),
				zap.String("allowed_origin", normalizedOrigin),
				zap.Error(resolveErr),
			)
		}
	} else if asset != nil && len(asset.Data) > 0 {
		updates["favicon_data"] = asset.Data
		updates["favicon_content_type"] = asset.ContentType
		updates["favicon_fetched_at"] = now
	}

	if updateErr := manager.database.Model(&model.Site{ID: siteID}).Updates(updates).Error; updateErr != nil {
		if manager.logger != nil {
			manager.logger.Warn("persist_site_favicon_failed", zap.String("site_id", siteID), zap.Error(updateErr))
		}
	}
}
