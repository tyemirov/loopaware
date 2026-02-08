package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/api"
	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
	"github.com/MarkoPoloResearchLab/loopaware/pkg/favicon"
)

type stubAssetResolver struct {
	asset *favicon.Asset
	err   error
	mu    sync.Mutex
	calls int
}

func (resolver *stubAssetResolver) Resolve(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (resolver *stubAssetResolver) ResolveAsset(_ context.Context, _ string) (*favicon.Asset, error) {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	resolver.calls++
	if resolver.err != nil {
		return nil, resolver.err
	}
	return resolver.asset, nil
}

func (resolver *stubAssetResolver) callCount() int {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	return resolver.calls
}

type blockingAssetResolver struct {
	asset       *favicon.Asset
	started     chan struct{}
	released    chan struct{}
	startOnce   sync.Once
	releaseOnce sync.Once
}

func newBlockingAssetResolver(asset *favicon.Asset) *blockingAssetResolver {
	return &blockingAssetResolver{
		asset:    asset,
		started:  make(chan struct{}),
		released: make(chan struct{}),
	}
}

func (resolver *blockingAssetResolver) Resolve(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (resolver *blockingAssetResolver) ResolveAsset(_ context.Context, _ string) (*favicon.Asset, error) {
	resolver.startOnce.Do(func() {
		close(resolver.started)
	})
	<-resolver.released
	return resolver.asset, nil
}

func (resolver *blockingAssetResolver) waitForStart(testingT *testing.T) {
	testingT.Helper()
	select {
	case <-resolver.started:
	case <-time.After(time.Second):
		testingT.Fatalf("resolver did not start")
	}
}

func (resolver *blockingAssetResolver) release() {
	resolver.releaseOnce.Do(func() {
		close(resolver.released)
	})
}

func TestSiteFaviconManagerStoresResolvedAsset(testingT *testing.T) {
	testingT.Helper()

	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Managed Site",
		AllowedOrigin: "https://managed.example",
		OwnerEmail:    "owner@example.com",
	}
	require.NoError(testingT, database.Create(&site).Error)

	resolver := &stubAssetResolver{asset: &favicon.Asset{ContentType: "image/png", Data: []byte{0x01, 0x02}}}
	service := favicon.NewService(resolver)
	manager := api.NewSiteFaviconManager(database, service, zap.NewNop())
	manager.Start(context.Background())
	testingT.Cleanup(manager.Stop)

	manager.ScheduleFetch(site)

	require.Eventually(testingT, func() bool {
		var refreshed model.Site
		if err := database.First(&refreshed, "id = ?", site.ID).Error; err != nil {
			return false
		}
		return len(refreshed.FaviconData) == 2 &&
			refreshed.FaviconContentType == "image/png" &&
			!refreshed.FaviconFetchedAt.IsZero()
	}, time.Second, 10*time.Millisecond)

	require.Equal(testingT, 1, resolver.callCount())
}

func TestSiteFaviconManagerAvoidsDuplicateFetches(testingT *testing.T) {
	testingT.Helper()

	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Cached Site",
		AllowedOrigin: "https://cached.example",
		OwnerEmail:    "owner@example.com",
	}
	require.NoError(testingT, database.Create(&site).Error)

	resolver := &stubAssetResolver{asset: &favicon.Asset{ContentType: "image/png", Data: []byte{0x0A}}}
	service := favicon.NewService(resolver)
	manager := api.NewSiteFaviconManager(database, service, zap.NewNop())
	manager.Start(context.Background())
	testingT.Cleanup(manager.Stop)

	manager.ScheduleFetch(site)

	require.Eventually(testingT, func() bool {
		var refreshed model.Site
		if err := database.First(&refreshed, "id = ?", site.ID).Error; err != nil {
			return false
		}
		return len(refreshed.FaviconData) > 0
	}, time.Second, 10*time.Millisecond)

	initialCalls := resolver.callCount()

	var refreshed model.Site
	require.NoError(testingT, database.First(&refreshed, "id = ?", site.ID).Error)
	manager.ScheduleFetch(refreshed)

	time.Sleep(50 * time.Millisecond)
	require.Equal(testingT, initialCalls, resolver.callCount())
}

func TestSiteFaviconManagerScheduledRefreshQueuesStaleSites(testingT *testing.T) {
	testingT.Helper()

	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	staleReferenceTime := time.Now()
	site := model.Site{
		ID:               storage.NewID(),
		Name:             "Stale Site",
		AllowedOrigin:    "https://stale.example",
		OwnerEmail:       "owner@example.com",
		FaviconData:      []byte{0x01},
		FaviconOrigin:    "https://stale.example",
		FaviconFetchedAt: staleReferenceTime.Add(-48 * time.Hour),
	}
	require.NoError(testingT, database.Create(&site).Error)

	resolver := &stubAssetResolver{asset: &favicon.Asset{ContentType: "image/png", Data: []byte{0x02}}}
	service := favicon.NewService(resolver)
	manager := api.NewSiteFaviconManager(
		database,
		service,
		zap.NewNop(),
		api.WithFaviconIntervals(5*time.Millisecond, 5*time.Millisecond),
		api.WithFaviconScanInterval(5*time.Millisecond),
		api.WithFaviconClock(func() time.Time { return staleReferenceTime }),
	)
	manager.Start(context.Background())
	testingT.Cleanup(manager.Stop)

	manager.TriggerScheduledRefresh()

	require.Eventually(testingT, func() bool {
		return resolver.callCount() > 0
	}, time.Second, 10*time.Millisecond)
}

func TestSiteFaviconManagerNotifiesSubscribers(testingT *testing.T) {
	testingT.Helper()

	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Observable Site",
		AllowedOrigin: "https://observable.example",
		OwnerEmail:    "owner@example.com",
	}
	require.NoError(testingT, database.Create(&site).Error)

	resolver := &stubAssetResolver{asset: &favicon.Asset{ContentType: "image/png", Data: []byte{0x03}}}
	service := favicon.NewService(resolver)
	manager := api.NewSiteFaviconManager(
		database,
		service,
		zap.NewNop(),
		api.WithFaviconIntervals(5*time.Millisecond, 5*time.Millisecond),
	)
	manager.Start(context.Background())
	testingT.Cleanup(manager.Stop)

	subscription := manager.Subscribe()
	require.NotNil(testingT, subscription)
	defer subscription.Close()

	manager.ScheduleFetch(site)

	var receivedEvent api.SiteFaviconEvent
	require.Eventually(testingT, func() bool {
		select {
		case event, ok := <-subscription.Events():
			if !ok {
				return false
			}
			receivedEvent = event
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	require.Equal(testingT, site.ID, receivedEvent.SiteID)
	require.NotEmpty(testingT, strings.TrimSpace(receivedEvent.FaviconURL))
}

func TestSiteFaviconManagerStopWaitsForInFlightFetch(testingT *testing.T) {
	testingT.Helper()

	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Blocking Site",
		AllowedOrigin: "https://blocking.example",
		OwnerEmail:    "owner@example.com",
	}
	require.NoError(testingT, database.Create(&site).Error)

	resolver := newBlockingAssetResolver(&favicon.Asset{ContentType: "image/png", Data: []byte{0x07}})
	service := favicon.NewService(resolver)
	manager := api.NewSiteFaviconManager(database, service, zap.NewNop())
	manager.Start(context.Background())
	testingT.Cleanup(manager.Stop)

	manager.ScheduleFetch(site)
	resolver.waitForStart(testingT)

	stopCompleted := make(chan struct{})
	go func() {
		manager.Stop()
		close(stopCompleted)
	}()

	select {
	case <-stopCompleted:
		testingT.Fatalf("Stop returned before fetch completion")
	case <-time.After(50 * time.Millisecond):
	}

	resolver.release()

	select {
	case <-stopCompleted:
	case <-time.After(time.Second):
		testingT.Fatalf("Stop did not finish after fetch completion")
	}
}

func TestGravityNotesInlineFaviconIntegration(testingT *testing.T) {
	testingT.Helper()

	if testing.Short() {
		testingT.Skip("skipping Gravity Notes favicon integration test in short mode")
	}

	resolver := favicon.NewHTTPResolver(nil, zap.NewNop())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	testingT.Cleanup(cancel)

	preflightAsset, preflightErr := resolver.ResolveAsset(ctx, "https://gravity.mprlab.com")
	if preflightErr != nil {
		testingT.Skipf("Gravity Notes favicon lookup failed: %v", preflightErr)
	}
	if preflightAsset == nil {
		testingT.Skip("Gravity Notes did not expose an inline favicon during test execution")
	}

	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	service := favicon.NewService(resolver)
	manager := api.NewSiteFaviconManager(database, service, zap.NewNop())
	manager.Start(context.Background())
	testingT.Cleanup(manager.Stop)
	require.NotNil(testingT, manager)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Gravity Notes",
		AllowedOrigin: "https://gravity.mprlab.com",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, database.Create(&site).Error)

	manager.ScheduleFetch(site)

	deadline := time.Now().Add(30 * time.Second)
	for {
		var refreshed model.Site
		require.NoError(testingT, database.First(&refreshed, "id = ?", site.ID).Error)
		if len(refreshed.FaviconData) > 0 {
			site = refreshed
			break
		}
		if time.Now().After(deadline) {
			testingT.Skip("LoopAware favicon was not retrieved within the allotted time")
		}
		time.Sleep(500 * time.Millisecond)
	}

	handlers := api.NewSiteHandlers(database, zap.NewNop(), testWidgetBaseURL, manager, nil, nil)

	listRecorder, listContext := newJSONContext(http.MethodGet, "/api/sites", nil)
	listContext.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	handlers.ListSites(listContext)
	require.Equal(testingT, http.StatusOK, listRecorder.Code)

	var listResponse struct {
		Sites []struct {
			Identifier string `json:"id"`
			FaviconURL string `json:"favicon_url"`
		} `json:"sites"`
	}
	require.NoError(testingT, json.Unmarshal(listRecorder.Body.Bytes(), &listResponse))
	require.NotEmpty(testingT, listResponse.Sites)
	require.NotEmpty(testingT, strings.TrimSpace(listResponse.Sites[0].FaviconURL))

	faviconRecorder, faviconContext := newJSONContext(http.MethodGet, "/api/sites/"+site.ID+"/favicon", nil)
	faviconContext.Params = gin.Params{{Key: "id", Value: site.ID}}
	faviconContext.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	handlers.SiteFavicon(faviconContext)
	require.Equal(testingT, http.StatusOK, faviconRecorder.Code)
	require.NotEmpty(testingT, faviconRecorder.Body.Bytes())
}
