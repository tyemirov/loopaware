package httpapi

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/task"
	"github.com/MarkoPoloResearchLab/loopaware/pkg/favicon"
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
	service         *favicon.Service
	logger          *zap.Logger
	retryInterval   time.Duration
	refreshInterval time.Duration
	scanInterval    time.Duration
	now             func() time.Time

	inFlight  sync.Map
	workQueue chan fetchTask

	scheduler    *task.Scheduler
	workerCancel context.CancelFunc
	workerGroup  sync.WaitGroup
	startOnce    sync.Once
	stopOnce     sync.Once

	subscribersMutex sync.RWMutex
	subscribers      map[int64]*faviconSubscriber
	nextSubscriberID int64
}

type fetchTask struct {
	siteID string
	force  bool
	notify bool
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

func NewSiteFaviconManager(database *gorm.DB, service *favicon.Service, logger *zap.Logger, options ...SiteFaviconManagerOption) *SiteFaviconManager {
	manager := &SiteFaviconManager{
		database:        database,
		service:         service,
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
		manager.workerGroup.Add(1)
		go func() {
			defer manager.workerGroup.Done()
			manager.runWorker(workerCtx)
		}()
		if manager.scheduler != nil {
			manager.scheduler.Start(workerCtx)
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
		manager.workerGroup.Wait()
		manager.workerCancel = nil
		manager.closeSubscribers()
	})
}

func (manager *SiteFaviconManager) ScheduleFetch(site model.Site) {
	if manager == nil || manager.service == nil || manager.database == nil {
		return
	}

	normalizedOrigin := strings.TrimSpace(site.AllowedOrigin)
	manager.scheduleFetch(
		site,
		normalizedOrigin,
		manager.shouldForceFetch(site, normalizedOrigin),
		manager.shouldNotifySubscribers(site, normalizedOrigin),
	)
}

func (manager *SiteFaviconManager) scheduleFetch(site model.Site, normalizedOrigin string, force bool, notify bool) {
	if normalizedOrigin == "" {
		return
	}

	if !force && !manager.shouldFetch(site, normalizedOrigin) {
		return
	}

	if _, alreadyInFlight := manager.inFlight.LoadOrStore(site.ID, struct{}{}); alreadyInFlight {
		return
	}

	task := fetchTask{siteID: site.ID, force: force, notify: notify}
	select {
	case manager.workQueue <- task:
	default:
		go manager.enqueueTask(task)
	}
}

func (manager *SiteFaviconManager) shouldForceFetch(site model.Site, normalizedOrigin string) bool {
	if len(site.FaviconData) == 0 {
		return true
	}
	if site.FaviconFetchedAt.IsZero() {
		return true
	}
	storedOrigin := strings.TrimSpace(site.FaviconOrigin)
	if storedOrigin == "" {
		return true
	}
	if !strings.EqualFold(storedOrigin, normalizedOrigin) {
		return true
	}
	return false
}

func (manager *SiteFaviconManager) shouldNotifySubscribers(site model.Site, normalizedOrigin string) bool {
	if len(site.FaviconData) == 0 {
		return true
	}
	if site.FaviconFetchedAt.IsZero() {
		return true
	}
	storedOrigin := strings.TrimSpace(site.FaviconOrigin)
	if storedOrigin == "" {
		return true
	}
	if !strings.EqualFold(storedOrigin, normalizedOrigin) {
		return true
	}
	return false
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
			manager.processFetch(ctx, task)
		}
	}
}

func (manager *SiteFaviconManager) processFetch(ctx context.Context, task fetchTask) {
	defer manager.inFlight.Delete(task.siteID)

	if manager.database == nil || manager.service == nil {
		return
	}

	var site model.Site
	if err := manager.database.First(&site, "id = ?", task.siteID).Error; err != nil {
		if manager.logger != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			manager.logger.Warn("load_site_for_favicon", zap.String("site_id", task.siteID), zap.Error(err))
		}
		return
	}

	normalizedOrigin := strings.TrimSpace(site.AllowedOrigin)
	if normalizedOrigin == "" {
		return
	}

	if !task.force && !manager.shouldFetch(site, normalizedOrigin) {
		return
	}

	currentTime := manager.now()
	siteSnapshot := favicon.Site{
		FaviconData:        site.FaviconData,
		FaviconContentType: site.FaviconContentType,
		FaviconFetchedAt:   site.FaviconFetchedAt,
	}
	result, resolveErr := manager.service.Collect(ctx, siteSnapshot, normalizedOrigin, task.notify, currentTime)
	if resolveErr != nil && manager.logger != nil {
		manager.logger.Debug(
			"fetch_site_favicon_failed",
			zap.String("site_id", task.siteID),
			zap.String("allowed_origin", normalizedOrigin),
			zap.Error(resolveErr),
		)
	}

	if len(result.Updates) == 0 {
		return
	}

	if updateErr := manager.database.Model(&model.Site{ID: task.siteID}).Updates(result.Updates).Error; updateErr != nil {
		if manager.logger != nil {
			manager.logger.Warn("persist_site_favicon_failed", zap.String("site_id", task.siteID), zap.Error(updateErr))
		}
		return
	}

	if result.ShouldNotify {
		eventTimestamp := result.EventTimestamp
		if eventTimestamp.IsZero() {
			eventTimestamp = currentTime
		}
		event := SiteFaviconEvent{
			SiteID:     task.siteID,
			FaviconURL: versionedSiteFaviconURL(task.siteID, eventTimestamp),
			UpdatedAt:  eventTimestamp,
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
			normalizedOrigin := strings.TrimSpace(site.AllowedOrigin)
			manager.scheduleFetch(
				site,
				normalizedOrigin,
				false,
				manager.shouldNotifySubscribers(site, normalizedOrigin),
			)
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
