package httpapi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/task"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
	"github.com/MarkoPoloResearchLab/loopaware/pkg/favicon"
)

const (
	testFaviconOrigin              = "https://favicon.example"
	testFaviconContentType         = "image/png"
	testFaviconUpdateCallbackName  = "force_favicon_update_error"
	testFaviconUpdateErrorMessage  = "favicon update failed"
	testFaviconQueueSiteIdentifier = "favicon-queue-site"
)

type staticAssetResolver struct {
	asset      *favicon.Asset
	resolveErr error
	callCount  int
}

func (resolver *staticAssetResolver) Resolve(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (resolver *staticAssetResolver) ResolveAsset(_ context.Context, _ string) (*favicon.Asset, error) {
	resolver.callCount++
	if resolver.resolveErr != nil {
		return nil, resolver.resolveErr
	}
	return resolver.asset, nil
}

func openFaviconManagerDatabase(testingT *testing.T) *gorm.DB {
	testingT.Helper()
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))
	return database
}

func TestSiteFaviconManagerShouldFetchBranches(testingT *testing.T) {
	referenceTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	manager := &SiteFaviconManager{
		retryInterval:   time.Minute,
		refreshInterval: 2 * time.Minute,
		now:             func() time.Time { return referenceTime },
	}

	testCases := []struct {
		name             string
		site             model.Site
		normalizedOrigin string
		expected         bool
	}{
		{
			name:             "empty stored origin",
			site:             model.Site{FaviconData: []byte{0x01}, FaviconFetchedAt: referenceTime},
			normalizedOrigin: testFaviconOrigin,
			expected:         true,
		},
		{
			name:             "origin mismatch",
			site:             model.Site{FaviconOrigin: "https://old.example", FaviconData: []byte{0x01}, FaviconFetchedAt: referenceTime},
			normalizedOrigin: testFaviconOrigin,
			expected:         true,
		},
		{
			name:             "missing data without attempt",
			site:             model.Site{FaviconOrigin: testFaviconOrigin},
			normalizedOrigin: testFaviconOrigin,
			expected:         true,
		},
		{
			name: "missing data with recent attempt",
			site: model.Site{
				FaviconOrigin:        testFaviconOrigin,
				FaviconLastAttemptAt: referenceTime.Add(-30 * time.Second),
			},
			normalizedOrigin: testFaviconOrigin,
			expected:         false,
		},
		{
			name: "missing data with stale attempt",
			site: model.Site{
				FaviconOrigin:        testFaviconOrigin,
				FaviconLastAttemptAt: referenceTime.Add(-2 * time.Minute),
			},
			normalizedOrigin: testFaviconOrigin,
			expected:         true,
		},
		{
			name:             "data present without fetch timestamp",
			site:             model.Site{FaviconOrigin: testFaviconOrigin, FaviconData: []byte{0x02}},
			normalizedOrigin: testFaviconOrigin,
			expected:         true,
		},
		{
			name: "data present and fresh",
			site: model.Site{
				FaviconOrigin:    testFaviconOrigin,
				FaviconData:      []byte{0x02},
				FaviconFetchedAt: referenceTime.Add(-30 * time.Second),
			},
			normalizedOrigin: testFaviconOrigin,
			expected:         false,
		},
		{
			name: "data present and stale",
			site: model.Site{
				FaviconOrigin:    testFaviconOrigin,
				FaviconData:      []byte{0x02},
				FaviconFetchedAt: referenceTime.Add(-3 * time.Minute),
			},
			normalizedOrigin: testFaviconOrigin,
			expected:         true,
		},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(nestedT *testing.T) {
			result := manager.shouldFetch(testCase.site, testCase.normalizedOrigin)
			require.Equal(nestedT, testCase.expected, result)
		})
	}
}

func TestSiteFaviconManagerShouldForceFetchHonorsStoredData(testingT *testing.T) {
	referenceTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	manager := &SiteFaviconManager{}

	site := model.Site{
		FaviconOrigin:    testFaviconOrigin,
		FaviconData:      []byte{0x01},
		FaviconFetchedAt: referenceTime,
	}
	require.False(testingT, manager.shouldForceFetch(site, testFaviconOrigin))

	site.FaviconData = nil
	require.True(testingT, manager.shouldForceFetch(site, testFaviconOrigin))
}

func TestSiteFaviconManagerShouldNotifySubscribersHonorsStoredData(testingT *testing.T) {
	referenceTime := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	manager := &SiteFaviconManager{}

	site := model.Site{
		FaviconOrigin:    testFaviconOrigin,
		FaviconData:      []byte{0x01},
		FaviconFetchedAt: referenceTime,
	}
	require.False(testingT, manager.shouldNotifySubscribers(site, testFaviconOrigin))

	site.FaviconFetchedAt = time.Time{}
	require.True(testingT, manager.shouldNotifySubscribers(site, testFaviconOrigin))
}

func TestSiteFaviconManagerShouldForceFetchHandlesOriginChanges(testingT *testing.T) {
	referenceTime := time.Date(2024, 1, 2, 9, 0, 0, 0, time.UTC)
	manager := &SiteFaviconManager{}

	site := model.Site{
		FaviconOrigin:    "",
		FaviconData:      []byte{0x01},
		FaviconFetchedAt: referenceTime,
	}
	require.True(testingT, manager.shouldForceFetch(site, testFaviconOrigin))

	site.FaviconOrigin = "https://different.example"
	require.True(testingT, manager.shouldForceFetch(site, testFaviconOrigin))

	site.FaviconOrigin = testFaviconOrigin
	site.FaviconFetchedAt = time.Time{}
	require.True(testingT, manager.shouldForceFetch(site, testFaviconOrigin))
}

func TestSiteFaviconManagerShouldNotifySubscribersHandlesOriginChanges(testingT *testing.T) {
	referenceTime := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)
	manager := &SiteFaviconManager{}

	site := model.Site{
		FaviconOrigin:    "",
		FaviconData:      []byte{0x01},
		FaviconFetchedAt: referenceTime,
	}
	require.True(testingT, manager.shouldNotifySubscribers(site, testFaviconOrigin))

	site.FaviconOrigin = "https://different.example"
	require.True(testingT, manager.shouldNotifySubscribers(site, testFaviconOrigin))
}

func TestSiteFaviconManagerScheduleFetchSkipsEmptyOrigin(testingT *testing.T) {
	manager := &SiteFaviconManager{workQueue: make(chan fetchTask, 1)}
	site := model.Site{ID: "site-empty-origin"}

	manager.scheduleFetch(site, "", true, true)

	_, exists := manager.inFlight.Load(site.ID)
	require.False(testingT, exists)
}

func TestSiteFaviconManagerScheduleFetchSkipsWhenFresh(testingT *testing.T) {
	referenceTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	manager := &SiteFaviconManager{
		workQueue:       make(chan fetchTask, 1),
		retryInterval:   time.Minute,
		refreshInterval: time.Hour,
		now:             func() time.Time { return referenceTime },
	}
	site := model.Site{
		ID:               "site-fresh",
		FaviconOrigin:    testFaviconOrigin,
		FaviconData:      []byte{0x01},
		FaviconFetchedAt: referenceTime,
	}

	manager.scheduleFetch(site, testFaviconOrigin, false, false)

	_, exists := manager.inFlight.Load(site.ID)
	require.False(testingT, exists)
	select {
	case <-manager.workQueue:
		testingT.Fatalf("unexpected fetch task queued")
	default:
	}
}

func TestSiteFaviconManagerScheduleFetchSkipsWhenInFlight(testingT *testing.T) {
	manager := &SiteFaviconManager{workQueue: make(chan fetchTask, 1)}
	site := model.Site{ID: "site-inflight"}
	manager.inFlight.Store(site.ID, struct{}{})

	manager.scheduleFetch(site, testFaviconOrigin, true, true)

	select {
	case <-manager.workQueue:
		testingT.Fatalf("unexpected fetch task queued")
	default:
	}
}

func TestSiteFaviconManagerScheduleFetchEnqueuesTask(testingT *testing.T) {
	manager := &SiteFaviconManager{workQueue: make(chan fetchTask, 1)}
	site := model.Site{ID: "site-queued"}

	manager.scheduleFetch(site, testFaviconOrigin, true, true)

	select {
	case task := <-manager.workQueue:
		require.Equal(testingT, site.ID, task.siteID)
		require.True(testingT, task.force)
		require.True(testingT, task.notify)
	case <-time.After(time.Second):
		testingT.Fatalf("expected fetch task")
	}
}

func TestSiteFaviconManagerProcessFetchSkipsEmptyAllowedOrigin(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Empty Origin Site",
		AllowedOrigin: " ",
		OwnerEmail:    "owner@example.com",
	}
	require.NoError(testingT, database.Create(&site).Error)

	resolver := &staticAssetResolver{asset: &favicon.Asset{ContentType: testFaviconContentType, Data: []byte{0x01}}}
	service := favicon.NewService(resolver)
	manager := &SiteFaviconManager{
		database: database,
		service:  service,
		logger:   zap.NewNop(),
		now:      func() time.Time { return time.Now().UTC() },
	}
	manager.processFetch(context.Background(), fetchTask{siteID: site.ID})
	require.Equal(testingT, 0, resolver.callCount)
}

func TestSiteFaviconManagerPerformScheduledRefreshReportsQueryError(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	sqlDatabase, sqlErr := database.DB()
	require.NoError(testingT, sqlErr)
	require.NoError(testingT, sqlDatabase.Close())

	manager := &SiteFaviconManager{
		database: database,
		logger:   zap.NewNop(),
	}
	manager.performScheduledRefresh(context.Background())
}

func TestSiteFaviconManagerTriggerScheduledRefreshHandlesNil(testingT *testing.T) {
	var manager *SiteFaviconManager
	manager.TriggerScheduledRefresh()
}

func TestSiteFaviconManagerTriggerScheduledRefreshFiresScheduler(testingT *testing.T) {
	triggered := make(chan struct{}, 1)
	scheduler := task.NewScheduler(time.Hour, func(_ context.Context) {
		triggered <- struct{}{}
	})
	manager := &SiteFaviconManager{scheduler: scheduler}
	scheduler.Start(context.Background())
	testingT.Cleanup(scheduler.Stop)

	manager.TriggerScheduledRefresh()

	select {
	case <-triggered:
	case <-time.After(time.Second):
		testingT.Fatalf("expected scheduled refresh")
	}
}

func TestSiteFaviconSubscriptionEventsHandlesNil(testingT *testing.T) {
	var subscription *SiteFaviconSubscription
	require.Nil(testingT, subscription.Events())
}

func TestSiteFaviconManagerCloseSubscribersClosesChannels(testingT *testing.T) {
	firstChannel := make(chan SiteFaviconEvent)
	secondChannel := make(chan SiteFaviconEvent)
	manager := &SiteFaviconManager{
		subscribers: map[int64]*faviconSubscriber{
			1: {identifier: 1, events: firstChannel},
			2: {identifier: 2, events: secondChannel},
		},
	}

	manager.closeSubscribers()

	_, open := <-firstChannel
	require.False(testingT, open)
	_, open = <-secondChannel
	require.False(testingT, open)
	require.Empty(testingT, manager.subscribers)
}

func TestSiteFaviconManagerProcessFetchSkipsMissingSite(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	manager := NewSiteFaviconManager(database, favicon.NewService(&staticAssetResolver{}), zap.NewNop())

	siteID := "missing-site"
	manager.inFlight.Store(siteID, struct{}{})
	manager.processFetch(context.Background(), fetchTask{siteID: siteID})

	_, exists := manager.inFlight.Load(siteID)
	require.False(testingT, exists)
}

func TestSiteFaviconManagerProcessFetchSkipsEmptyOrigin(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	site := model.Site{ID: storage.NewID(), Name: "Originless", AllowedOrigin: "   ", OwnerEmail: "owner@example.com"}
	require.NoError(testingT, database.Create(&site).Error)

	resolver := &staticAssetResolver{asset: &favicon.Asset{ContentType: "image/png", Data: []byte{0x01}}}
	manager := NewSiteFaviconManager(database, favicon.NewService(resolver), zap.NewNop())

	manager.processFetch(context.Background(), fetchTask{siteID: site.ID, force: true})
	require.Equal(testingT, 0, resolver.callCount)
}

func TestSiteFaviconManagerProcessFetchSkipsWhenNoUpdates(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	site := model.Site{ID: storage.NewID(), Name: "No Updates", AllowedOrigin: testFaviconOrigin, OwnerEmail: "owner@example.com"}
	require.NoError(testingT, database.Create(&site).Error)

	manager := NewSiteFaviconManager(database, &favicon.Service{}, zap.NewNop())
	manager.processFetch(context.Background(), fetchTask{siteID: site.ID, force: true})

	var refreshed model.Site
	require.NoError(testingT, database.First(&refreshed, "id = ?", site.ID).Error)
	require.Empty(testingT, refreshed.FaviconData)
}

func TestSiteFaviconManagerProcessFetchBroadcastsUpdates(testingT *testing.T) {
	referenceTime := time.Date(2024, 1, 2, 8, 0, 0, 0, time.UTC)
	database := openFaviconManagerDatabase(testingT)
	site := model.Site{ID: storage.NewID(), Name: "Broadcast", AllowedOrigin: testFaviconOrigin, OwnerEmail: "owner@example.com"}
	require.NoError(testingT, database.Create(&site).Error)

	resolver := &staticAssetResolver{asset: &favicon.Asset{ContentType: "image/png", Data: []byte{0x0A}}}
	service := favicon.NewService(resolver)
	manager := NewSiteFaviconManager(
		database,
		service,
		zap.NewNop(),
		WithFaviconClock(func() time.Time { return referenceTime }),
	)

	subscription := manager.Subscribe()
	require.NotNil(testingT, subscription)
	testingT.Cleanup(subscription.Close)

	manager.processFetch(context.Background(), fetchTask{siteID: site.ID, force: true, notify: true})

	var refreshed model.Site
	require.NoError(testingT, database.First(&refreshed, "id = ?", site.ID).Error)
	require.Equal(testingT, []byte{0x0A}, refreshed.FaviconData)
	require.Equal(testingT, "image/png", refreshed.FaviconContentType)
	require.False(testingT, refreshed.FaviconFetchedAt.IsZero())

	select {
	case event := <-subscription.Events():
		require.Equal(testingT, site.ID, event.SiteID)
		require.Contains(testingT, event.FaviconURL, site.ID)
		require.Equal(testingT, referenceTime, event.UpdatedAt)
	case <-time.After(time.Second):
		testingT.Fatalf("expected favicon update event")
	}
}

func TestSiteFaviconManagerPerformScheduledRefreshQueuesSites(testingT *testing.T) {
	referenceTime := time.Date(2024, 1, 2, 9, 0, 0, 0, time.UTC)
	database := openFaviconManagerDatabase(testingT)
	primarySite := model.Site{ID: storage.NewID(), Name: "Primary", AllowedOrigin: testFaviconOrigin, OwnerEmail: "owner@example.com"}
	emptySite := model.Site{ID: storage.NewID(), Name: "Empty", AllowedOrigin: "   ", OwnerEmail: "owner@example.com"}
	require.NoError(testingT, database.Create(&primarySite).Error)
	require.NoError(testingT, database.Create(&emptySite).Error)

	manager := &SiteFaviconManager{
		database:        database,
		workQueue:       make(chan fetchTask, 2),
		retryInterval:   time.Minute,
		refreshInterval: time.Minute,
		now:             func() time.Time { return referenceTime },
		subscribers:     make(map[int64]*faviconSubscriber),
	}

	manager.performScheduledRefresh(context.Background())

	select {
	case task := <-manager.workQueue:
		require.Equal(testingT, primarySite.ID, task.siteID)
	case <-time.After(time.Second):
		testingT.Fatalf("expected scheduled fetch")
	}
}

func TestSiteFaviconManagerPerformScheduledRefreshHonorsCancel(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	site := model.Site{ID: storage.NewID(), Name: "Cancel", AllowedOrigin: testFaviconOrigin, OwnerEmail: "owner@example.com"}
	require.NoError(testingT, database.Create(&site).Error)

	manager := &SiteFaviconManager{
		database:        database,
		workQueue:       make(chan fetchTask, 1),
		retryInterval:   time.Minute,
		refreshInterval: time.Minute,
		now:             time.Now,
		subscribers:     make(map[int64]*faviconSubscriber),
	}

	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()

	manager.performScheduledRefresh(canceledContext)

	select {
	case <-manager.workQueue:
		testingT.Fatalf("unexpected scheduled fetch")
	default:
	}
}

func TestSiteFaviconManagerEnqueueTaskDropsInFlightOnTimeout(testingT *testing.T) {
	manager := &SiteFaviconManager{workQueue: make(chan fetchTask)}
	task := fetchTask{siteID: testFaviconQueueSiteIdentifier}
	manager.inFlight.Store(task.siteID, struct{}{})

	manager.enqueueTask(task)

	_, exists := manager.inFlight.Load(task.siteID)
	require.False(testingT, exists)
}

func TestSiteFaviconManagerProcessFetchReportsUpdateError(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	site := model.Site{ID: storage.NewID(), Name: "Update Error", AllowedOrigin: testFaviconOrigin, OwnerEmail: "owner@example.com"}
	require.NoError(testingT, database.Create(&site).Error)

	database.Callback().Update().Before("gorm:update").Register(testFaviconUpdateCallbackName, func(database *gorm.DB) {
		database.AddError(errors.New(testFaviconUpdateErrorMessage))
	})
	testingT.Cleanup(func() {
		database.Callback().Update().Remove(testFaviconUpdateCallbackName)
	})

	resolver := &staticAssetResolver{asset: &favicon.Asset{ContentType: "image/png", Data: []byte{0x01}}}
	manager := NewSiteFaviconManager(database, favicon.NewService(resolver), zap.NewNop())

	manager.processFetch(context.Background(), fetchTask{siteID: site.ID, force: true, notify: true})

	var refreshed model.Site
	require.NoError(testingT, database.First(&refreshed, "id = ?", site.ID).Error)
	require.Empty(testingT, refreshed.FaviconData)
}

func TestVersionedSiteFaviconURLFormatsTimestamp(testingT *testing.T) {
	fetchTime := time.Date(2024, 1, 1, 1, 2, 3, 0, time.UTC)
	require.Empty(testingT, versionedSiteFaviconURL("", fetchTime))
	require.Equal(testingT, "/api/sites/site-id/favicon", versionedSiteFaviconURL("site-id", time.Time{}))
	require.Equal(testingT, "/api/sites/site-id/favicon?ts=1704070923", versionedSiteFaviconURL("site-id", fetchTime))
}
