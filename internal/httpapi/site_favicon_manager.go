package httpapi

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/task"
)

const (
	defaultFaviconRetryInterval   = 30 * time.Minute
	defaultFaviconRefreshInterval = 24 * time.Hour
	defaultFaviconScanInterval    = time.Hour
	defaultFaviconQueueCapacity   = 64
)

type SiteFaviconEvent struct {
	SiteID     string
	FaviconURL string
	UpdatedAt  time.Time
}

type SiteFaviconManagerOption func(*SiteFaviconManager)

type SiteFaviconManager struct {
	database        *gorm.DB
	resolver        FaviconResolver
	logger          *zap.Logger
	retryInterval   time.Duration
	refreshInterval time.Duration
	scanInterval    time.Duration
	now             func() time.Time

	inFlight  sync.Map
	workQueue chan fetchTask

	scheduler    *task.Scheduler
	workerCancel context.CancelFunc
	startOnce    sync.Once
	stopOnce     sync.Once

	subscribersMutex sync.RWMutex
	subscribers      map[int64]*faviconSubscriber
	nextSubscriberID int64
}

type fetchTask struct {
	siteID string
}

type faviconSubscriber struct {
	identifier int64
	events     chan SiteFaviconEvent
}

type SiteFaviconSubscription struct {
	manager    *SiteFaviconManager
	identifier int64
	events     chan SiteFaviconEvent
	closeOnce  sync.Once
}

func NewSiteFaviconManager(database *gorm.DB, resolver FaviconResolver, logger *zap.Logger, options ...SiteFaviconManagerOption) *SiteFaviconManager {
	manager := &SiteFaviconManager{
		database:        database,
		resolver:        resolver,
		logger:          logger,
		retryInterval:   defaultFaviconRetryInterval,
		refreshInterval: defaultFaviconRefreshInterval,
		scanInterval:    defaultFaviconScanInterval,
		now:             time.Now,
		workQueue:       make(chan fetchTask, defaultFaviconQueueCapacity),
		subscribers:     make(map[int64]*faviconSubscriber),
	}
	for _, option := range options {
		if option != nil {
			option(manager)
		}
	}
	manager.scheduler = task.NewScheduler(manager.scanInterval, manager.performScheduledRefresh)
	return manager
}

func WithFaviconIntervals(retryInterval time.Duration, refreshInterval time.Duration) SiteFaviconManagerOption {
	return func(manager *SiteFaviconManager) {
		if retryInterval > 0 {
			manager.retryInterval = retryInterval
		}
		if refreshInterval > 0 {
			manager.refreshInterval = refreshInterval
		}
	}
}

func WithFaviconScanInterval(scanInterval time.Duration) SiteFaviconManagerOption {
	return func(manager *SiteFaviconManager) {
		if scanInterval > 0 {
			manager.scanInterval = scanInterval
		}
	}
}

func WithFaviconClock(clock func() time.Time) SiteFaviconManagerOption {
	return func(manager *SiteFaviconManager) {
		if clock != nil {
			manager.now = clock
		}
	}
}

func (manager *SiteFaviconManager) Start(ctx context.Context) {
	if manager == nil {
		return
	}
	manager.startOnce.Do(func() {
		workerCtx, cancel := context.WithCancel(ctx)
		manager.workerCancel = cancel
		go manager.runWorker(workerCtx)
		if manager.scheduler != nil {
			manager.scheduler.Start(workerCtx)
			manager.scheduler.Trigger()
		}
	})
}

func (manager *SiteFaviconManager) Stop() {
	if manager == nil {
		return
	}
	manager.stopOnce.Do(func() {
		if manager.scheduler != nil {
			manager.scheduler.Stop()
		}
		if manager.workerCancel != nil {
			manager.workerCancel()
		}
		manager.closeSubscribers()
	})
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

	task := fetchTask{siteID: site.ID}
	select {
	case manager.workQueue <- task:
	default:
		go manager.enqueueTask(task)
	}
}

func (manager *SiteFaviconManager) TriggerScheduledRefresh() {
	if manager == nil || manager.scheduler == nil {
		return
	}
	manager.scheduler.Trigger()
}

func (manager *SiteFaviconManager) Subscribe() *SiteFaviconSubscription {
	if manager == nil {
		return nil
	}
	subscriptionChannel := make(chan SiteFaviconEvent, 8)
	identifier := atomic.AddInt64(&manager.nextSubscriberID, 1)
	manager.subscribersMutex.Lock()
	manager.subscribers[identifier] = &faviconSubscriber{
		identifier: identifier,
		events:     subscriptionChannel,
	}
	manager.subscribersMutex.Unlock()
	return &SiteFaviconSubscription{
		manager:    manager,
		identifier: identifier,
		events:     subscriptionChannel,
	}
}

func (subscription *SiteFaviconSubscription) Events() <-chan SiteFaviconEvent {
	if subscription == nil {
		return nil
	}
	return subscription.events
}

func (subscription *SiteFaviconSubscription) Close() {
	if subscription == nil {
		return
	}
	subscription.closeOnce.Do(func() {
		if subscription.manager != nil {
			subscription.manager.removeSubscriber(subscription.identifier)
		}
	})
}

func (manager *SiteFaviconManager) enqueueTask(task fetchTask) {
	if manager == nil {
		return
	}
	select {
	case manager.workQueue <- task:
	case <-time.After(time.Second):
		manager.inFlight.Delete(task.siteID)
	}
}

func (manager *SiteFaviconManager) runWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-manager.workQueue:
			manager.processFetch(ctx, task.siteID)
		}
	}
}

func (manager *SiteFaviconManager) processFetch(ctx context.Context, siteID string) {
	defer manager.inFlight.Delete(siteID)

	if manager.database == nil || manager.resolver == nil {
		return
	}

	var site model.Site
	if err := manager.database.First(&site, "id = ?", siteID).Error; err != nil {
		if manager.logger != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			manager.logger.Warn("load_site_for_favicon", zap.String("site_id", siteID), zap.Error(err))
		}
		return
	}

	normalizedOrigin := strings.TrimSpace(site.AllowedOrigin)
	if normalizedOrigin == "" {
		return
	}

	if !manager.shouldFetch(site, normalizedOrigin) {
		return
	}

	asset, resolveErr := manager.resolver.ResolveAsset(ctx, normalizedOrigin)
	now := manager.now()

	updates := map[string]any{
		"favicon_origin":          normalizedOrigin,
		"favicon_last_attempt_at": now,
	}

	shouldNotify := false
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
		if !bytes.Equal(site.FaviconData, asset.Data) || !strings.EqualFold(strings.TrimSpace(site.FaviconContentType), strings.TrimSpace(asset.ContentType)) || site.FaviconFetchedAt.IsZero() {
			updates["favicon_data"] = asset.Data
			updates["favicon_content_type"] = asset.ContentType
			updates["favicon_fetched_at"] = now
			shouldNotify = true
		} else {
			updates["favicon_fetched_at"] = now
		}
	}

	if updateErr := manager.database.Model(&model.Site{ID: siteID}).Updates(updates).Error; updateErr != nil {
		if manager.logger != nil {
			manager.logger.Warn("persist_site_favicon_failed", zap.String("site_id", siteID), zap.Error(updateErr))
		}
		return
	}

	if shouldNotify {
		event := SiteFaviconEvent{
			SiteID:     siteID,
			FaviconURL: versionedSiteFaviconURL(siteID, now),
			UpdatedAt:  now,
		}
		manager.broadcast(event)
	}
}

func (manager *SiteFaviconManager) performScheduledRefresh(ctx context.Context) {
	if manager.database == nil {
		return
	}

	var sites []model.Site
	if err := manager.database.
		Select("id", "allowed_origin", "favicon_origin", "favicon_data", "favicon_fetched_at", "favicon_last_attempt_at").
		Find(&sites).Error; err != nil {
		if manager.logger != nil {
			manager.logger.Warn("load_sites_for_favicon_refresh", zap.Error(err))
		}
		return
	}

	for _, site := range sites {
		select {
		case <-ctx.Done():
			return
		default:
			manager.ScheduleFetch(site)
		}
	}
}

func (manager *SiteFaviconManager) broadcast(event SiteFaviconEvent) {
	manager.subscribersMutex.RLock()
	defer manager.subscribersMutex.RUnlock()
	for _, subscriber := range manager.subscribers {
		select {
		case subscriber.events <- event:
		default:
		}
	}
}

func (manager *SiteFaviconManager) removeSubscriber(identifier int64) {
	manager.subscribersMutex.Lock()
	subscriber, exists := manager.subscribers[identifier]
	if exists {
		delete(manager.subscribers, identifier)
	}
	manager.subscribersMutex.Unlock()
	if exists {
		close(subscriber.events)
	}
}

func (manager *SiteFaviconManager) closeSubscribers() {
	manager.subscribersMutex.Lock()
	for identifier, subscriber := range manager.subscribers {
		close(subscriber.events)
		delete(manager.subscribers, identifier)
	}
	manager.subscribersMutex.Unlock()
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
	if site.FaviconFetchedAt.IsZero() {
		return true
	}
	return manager.now().Sub(site.FaviconFetchedAt) >= manager.refreshInterval
}
