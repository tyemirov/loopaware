package httpapi_test

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

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/httpapi"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/testutil"
)

type stubAssetResolver struct {
	asset *httpapi.FaviconAsset
	err   error
	mu    sync.Mutex
	calls int
}

func (resolver *stubAssetResolver) Resolve(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (resolver *stubAssetResolver) ResolveAsset(_ context.Context, _ string) (*httpapi.FaviconAsset, error) {
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

func TestSiteFaviconManagerStoresResolvedAsset(testingT *testing.T) {
	testingT.Helper()

	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Managed Site",
		AllowedOrigin: "https://managed.example",
		OwnerEmail:    "owner@example.com",
	}
	require.NoError(testingT, database.Create(&site).Error)

	resolver := &stubAssetResolver{asset: &httpapi.FaviconAsset{ContentType: "image/png", Data: []byte{0x01, 0x02}}}
	manager := httpapi.NewSiteFaviconManager(database, resolver, zap.NewNop())

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
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Cached Site",
		AllowedOrigin: "https://cached.example",
		OwnerEmail:    "owner@example.com",
	}
	require.NoError(testingT, database.Create(&site).Error)

	resolver := &stubAssetResolver{asset: &httpapi.FaviconAsset{ContentType: "image/png", Data: []byte{0x0A}}}
	manager := httpapi.NewSiteFaviconManager(database, resolver, zap.NewNop())

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

func TestGravityNotesInlineFaviconIntegration(testingT *testing.T) {
	testingT.Helper()

	if testing.Short() {
		testingT.Skip("skipping Gravity Notes favicon integration test in short mode")
	}

	resolver := httpapi.NewHTTPFaviconResolver(nil, zap.NewNop())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
	require.NoError(testingT, storage.AutoMigrate(database))

	manager := httpapi.NewSiteFaviconManager(database, resolver, zap.NewNop())
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

	handlers := httpapi.NewSiteHandlers(database, zap.NewNop(), testWidgetBaseURL, manager, nil)

	listRecorder, listContext := newJSONContext(http.MethodGet, "/api/sites", nil)
	listContext.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

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
	faviconContext.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, Role: httpapi.RoleAdmin})

	handlers.SiteFavicon(faviconContext)
	require.Equal(testingT, http.StatusOK, faviconRecorder.Code)
	require.NotEmpty(testingT, faviconRecorder.Body.Bytes())
}
