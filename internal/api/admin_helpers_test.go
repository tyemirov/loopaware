package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

const (
	testBrowserEdgeUserAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/110.0.0.0 Safari/537.36 Edg/110.0.0.0"
	testBrowserOperaUserAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36 OPR/106.0.0.0"
	testBrowserChromeUserAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"
	testBrowserSafariUserAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Safari/605.1.15"
	testBrowserFirefoxUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/109.0"
	testBrowserIEUserAgent      = "Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.1; Trident/6.0)"
	testBrowserCurlUserAgent    = "curl/8.0.1"
	testOwnerEmail              = "owner@example.com"
	testCreatorEmail            = "creator@example.com"
	testSiteID                  = "site-id"
	testSiteName                = "Example Site"
	testAllowedOriginPrimary    = "https://example.com"
	testAllowedOriginExtra      = "https://docs.example.com"
	testInvalidValue            = "invalid"
)

type stubStatsProvider struct {
	feedbackCountValue      int64
	subscriberCountValue    int64
	visitCountValue         int64
	uniqueVisitorCountValue int64
	feedbackCountError      error
	subscriberCountError    error
	visitCountError         error
	uniqueVisitorCountError error
}

func (provider *stubStatsProvider) FeedbackCount(context.Context, string) (int64, error) {
	return provider.feedbackCountValue, provider.feedbackCountError
}

func (provider *stubStatsProvider) SubscriberCount(context.Context, string) (int64, error) {
	return provider.subscriberCountValue, provider.subscriberCountError
}

func (provider *stubStatsProvider) VisitCount(context.Context, string) (int64, error) {
	return provider.visitCountValue, provider.visitCountError
}

func (provider *stubStatsProvider) UniqueVisitorCount(context.Context, string) (int64, error) {
	return provider.uniqueVisitorCountValue, provider.uniqueVisitorCountError
}

func (provider *stubStatsProvider) TopPages(context.Context, string, int) ([]TopPageStat, error) {
	return nil, nil
}

func TestClassifyVisitBrowser(testingT *testing.T) {
	testCases := []struct {
		name        string
		userAgent   string
		expectClass string
	}{
		{name: "unknown", userAgent: "", expectClass: "Unknown"},
		{name: "edge", userAgent: testBrowserEdgeUserAgent, expectClass: "Microsoft Edge"},
		{name: "opera", userAgent: testBrowserOperaUserAgent, expectClass: "Opera"},
		{name: "chrome", userAgent: testBrowserChromeUserAgent, expectClass: "Google Chrome"},
		{name: "safari", userAgent: testBrowserSafariUserAgent, expectClass: "Safari"},
		{name: "firefox", userAgent: testBrowserFirefoxUserAgent, expectClass: "Firefox"},
		{name: "internet explorer", userAgent: testBrowserIEUserAgent, expectClass: "Internet Explorer"},
		{name: "curl", userAgent: testBrowserCurlUserAgent, expectClass: "curl"},
		{name: "other", userAgent: "CustomAgent", expectClass: "Other"},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			require.Equal(testingT, testCase.expectClass, classifyVisitBrowser(testCase.userAgent))
		})
	}
}

func TestClassifyVisitCountry(testingT *testing.T) {
	testCases := []struct {
		name        string
		ipAddress   string
		expectLabel string
	}{
		{name: "empty", ipAddress: "", expectLabel: "Unknown"},
		{name: "invalid", ipAddress: testInvalidValue, expectLabel: "Unknown"},
		{name: "loopback", ipAddress: "127.0.0.1", expectLabel: "Local network"},
		{name: "private", ipAddress: "10.0.0.1", expectLabel: "Local network"},
		{name: "public", ipAddress: "8.8.8.8", expectLabel: "Unknown"},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			require.Equal(testingT, testCase.expectLabel, classifyVisitCountry(testCase.ipAddress))
		})
	}
}

func TestNormalizeAllowedOrigins(testingT *testing.T) {
	rawValue := testAllowedOriginPrimary + " , " + testAllowedOriginExtra + " " + testAllowedOriginPrimary
	normalized := normalizeAllowedOrigins(rawValue)
	require.Equal(testingT, testAllowedOriginPrimary+" "+testAllowedOriginExtra, normalized)
}

func TestNormalizeWidgetBaseURL(testingT *testing.T) {
	require.Equal(testingT, "https://example.com", normalizeWidgetBaseURL("https://example.com/"))
}

func TestSanitizeWidgetBubbleSide(testingT *testing.T) {
	normalized, normalizeErr := sanitizeWidgetBubbleSide("")
	require.NoError(testingT, normalizeErr)
	require.Equal(testingT, defaultWidgetBubbleSide, normalized)

	normalized, normalizeErr = sanitizeWidgetBubbleSide(widgetBubbleSideLeft)
	require.NoError(testingT, normalizeErr)
	require.Equal(testingT, widgetBubbleSideLeft, normalized)

	_, normalizeErr = sanitizeWidgetBubbleSide(testInvalidValue)
	require.Error(testingT, normalizeErr)
}

func TestSanitizeWidgetBubbleBottomOffset(testingT *testing.T) {
	normalized, normalizeErr := sanitizeWidgetBubbleBottomOffset(nil)
	require.NoError(testingT, normalizeErr)
	require.Equal(testingT, defaultWidgetBubbleBottomOffset, normalized)

	validOffset := defaultWidgetBubbleBottomOffset + 1
	normalized, normalizeErr = sanitizeWidgetBubbleBottomOffset(&validOffset)
	require.NoError(testingT, normalizeErr)
	require.Equal(testingT, validOffset, normalized)

	invalidOffset := minWidgetBubbleBottomOffset - 1
	_, normalizeErr = sanitizeWidgetBubbleBottomOffset(&invalidOffset)
	require.Error(testingT, normalizeErr)
}

func TestEnsureWidgetBubblePlacementDefaults(testingT *testing.T) {
	site := &model.Site{
		WidgetBubbleSide:           "invalid",
		WidgetBubbleBottomOffsetPx: minWidgetBubbleBottomOffset - 1,
	}
	ensureWidgetBubblePlacementDefaults(site)
	require.Equal(testingT, defaultWidgetBubbleSide, site.WidgetBubbleSide)
	require.Equal(testingT, defaultWidgetBubbleBottomOffset, site.WidgetBubbleBottomOffsetPx)
}

func TestGinRequestContext(testingT *testing.T) {
	handlers := &SiteHandlers{}
	require.Equal(testingT, context.Background(), handlers.ginRequestContext(nil))

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	requestContext, cancel := context.WithCancel(context.Background())
	testingT.Cleanup(cancel)
	ginContext.Request = request.WithContext(requestContext)
	require.Equal(testingT, requestContext, handlers.ginRequestContext(ginContext))
}

func TestUserCanAccessSite(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            testSiteID,
		Name:          testSiteName,
		OwnerEmail:    testOwnerEmail,
		CreatorEmail:  testCreatorEmail,
		AllowedOrigin: testAllowedOriginPrimary,
	}
	require.NoError(testingT, database.Create(&site).Error)

	handlers := &SiteHandlers{database: database}
	requestContext := context.Background()

	ownerUser := &CurrentUser{Email: testOwnerEmail, Role: RoleUser}
	require.True(testingT, handlers.userCanAccessSite(requestContext, ownerUser, testSiteID))

	creatorUser := &CurrentUser{Email: testCreatorEmail, Role: RoleUser}
	require.True(testingT, handlers.userCanAccessSite(requestContext, creatorUser, testSiteID))

	otherUser := &CurrentUser{Email: "other@example.com", Role: RoleUser}
	require.False(testingT, handlers.userCanAccessSite(requestContext, otherUser, testSiteID))
}

func TestAllowedOriginConflictExists(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Primary Site",
		AllowedOrigin: testAllowedOriginPrimary,
		OwnerEmail:    testOwnerEmail,
	}
	conflictingSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Conflicting Site",
		AllowedOrigin: testAllowedOriginPrimary + " " + testAllowedOriginExtra,
		OwnerEmail:    testOwnerEmail,
	}
	require.NoError(testingT, database.Create(&site).Error)
	require.NoError(testingT, database.Create(&conflictingSite).Error)

	handlers := &SiteHandlers{database: database}
	conflict, conflictErr := handlers.allowedOriginConflictExists(testAllowedOriginPrimary, "")
	require.NoError(testingT, conflictErr)
	require.True(testingT, conflict)

	conflict, conflictErr = handlers.allowedOriginConflictExists(testAllowedOriginExtra, conflictingSite.ID)
	require.NoError(testingT, conflictErr)
	require.False(testingT, conflict)
}

func TestCountsHandleErrors(testingT *testing.T) {
	statsProvider := &stubStatsProvider{
		feedbackCountError:      context.DeadlineExceeded,
		subscriberCountError:    context.DeadlineExceeded,
		visitCountError:         context.DeadlineExceeded,
		uniqueVisitorCountError: context.DeadlineExceeded,
	}
	handlers := &SiteHandlers{
		logger:        zap.NewNop(),
		statsProvider: statsProvider,
	}
	requestContext := context.Background()

	require.Equal(testingT, int64(0), handlers.feedbackCount(requestContext, testSiteID))
	require.Equal(testingT, int64(0), handlers.subscriberCount(requestContext, testSiteID))
	require.Equal(testingT, int64(0), handlers.visitCount(requestContext, testSiteID))
	require.Equal(testingT, int64(0), handlers.uniqueVisitorCount(requestContext, testSiteID))
}

func TestCountsReturnValues(testingT *testing.T) {
	statsProvider := &stubStatsProvider{
		feedbackCountValue:      3,
		subscriberCountValue:    2,
		visitCountValue:         5,
		uniqueVisitorCountValue: 4,
	}
	handlers := &SiteHandlers{
		statsProvider: statsProvider,
	}
	requestContext := context.Background()

	require.Equal(testingT, int64(3), handlers.feedbackCount(requestContext, testSiteID))
	require.Equal(testingT, int64(2), handlers.subscriberCount(requestContext, testSiteID))
	require.Equal(testingT, int64(5), handlers.visitCount(requestContext, testSiteID))
	require.Equal(testingT, int64(4), handlers.uniqueVisitorCount(requestContext, testSiteID))
}

func TestEnsureWidgetBubblePlacementDefaultsHandlesNil(testingT *testing.T) {
	ensureWidgetBubblePlacementDefaults(nil)
}

func TestNormalizeAllowedOriginsHandlesEmpty(testingT *testing.T) {
	require.Equal(testingT, "", normalizeAllowedOrigins("  "))
}

func TestSanitizeWidgetBubbleBottomOffsetUpperBound(testingT *testing.T) {
	invalidOffset := maxWidgetBubbleBottomOffset + 1
	_, normalizeErr := sanitizeWidgetBubbleBottomOffset(&invalidOffset)
	require.Error(testingT, normalizeErr)
}

func TestEnsureWidgetBubblePlacementDefaultsKeepsValid(testingT *testing.T) {
	site := &model.Site{
		WidgetBubbleSide:           widgetBubbleSideRight,
		WidgetBubbleBottomOffsetPx: minWidgetBubbleBottomOffset + 1,
	}
	ensureWidgetBubblePlacementDefaults(site)
	require.Equal(testingT, widgetBubbleSideRight, site.WidgetBubbleSide)
	require.Equal(testingT, minWidgetBubbleBottomOffset+1, site.WidgetBubbleBottomOffsetPx)
}

func TestNormalizeWidgetBaseURLHandlesEmpty(testingT *testing.T) {
	require.Equal(testingT, "", normalizeWidgetBaseURL("   "))
}

func TestClassifyVisitCountryHandlesPrivateRanges(testingT *testing.T) {
	require.Equal(testingT, "Local network", classifyVisitCountry("192.168.1.5"))
}

func TestUserCanAccessSiteReturnsFalseForMissingData(testingT *testing.T) {
	handlers := &SiteHandlers{}
	require.False(testingT, handlers.userCanAccessSite(context.Background(), nil, testSiteID))
}

func TestAllowedOriginConflictHandlesEmpty(testingT *testing.T) {
	handlers := &SiteHandlers{}
	conflict, conflictErr := handlers.allowedOriginConflictExists("", "")
	require.NoError(testingT, conflictErr)
	require.False(testingT, conflict)
}

func TestUserCanAccessSiteReturnsTrueForOwner(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          testSiteName,
		AllowedOrigin: testAllowedOriginPrimary,
		OwnerEmail:    testOwnerEmail,
	}
	require.NoError(testingT, database.Create(&site).Error)

	handlers := &SiteHandlers{database: database}
	currentUser := &CurrentUser{Email: testOwnerEmail, Role: RoleUser}
	require.True(testingT, handlers.userCanAccessSite(context.Background(), currentUser, site.ID))
}

func TestUserCanAccessSiteReturnsFalseOnQueryError(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	sqlDatabase, sqlErr := database.DB()
	require.NoError(testingT, sqlErr)
	require.NoError(testingT, sqlDatabase.Close())

	handlers := &SiteHandlers{database: database}
	currentUser := &CurrentUser{Email: testOwnerEmail, Role: RoleUser}
	require.False(testingT, handlers.userCanAccessSite(context.Background(), currentUser, testSiteID))
}

func TestFeedbackCountHandlesNilProvider(testingT *testing.T) {
	handlers := &SiteHandlers{}
	require.Equal(testingT, int64(0), handlers.feedbackCount(context.Background(), testSiteID))
}

func TestSubscriberCountHandlesNilProvider(testingT *testing.T) {
	handlers := &SiteHandlers{}
	require.Equal(testingT, int64(0), handlers.subscriberCount(context.Background(), testSiteID))
}

func TestVisitCountHandlesNilProvider(testingT *testing.T) {
	handlers := &SiteHandlers{}
	require.Equal(testingT, int64(0), handlers.visitCount(context.Background(), testSiteID))
}

func TestUniqueVisitorCountHandlesNilProvider(testingT *testing.T) {
	handlers := &SiteHandlers{}
	require.Equal(testingT, int64(0), handlers.uniqueVisitorCount(context.Background(), testSiteID))
}

func TestResolveAuthorizedSiteRejectsMissingUser(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	handlers := &SiteHandlers{database: database}
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Params = gin.Params{{Key: "id", Value: testSiteID}}
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	_, _, isAuthorized := handlers.resolveAuthorizedSite(ginContext)
	require.False(testingT, isAuthorized)
	require.Equal(testingT, http.StatusUnauthorized, recorder.Code)
}

func TestResolveAuthorizedSiteRejectsMissingSiteID(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	handlers := &SiteHandlers{database: database}
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: testOwnerEmail, Role: RoleUser})

	_, _, isAuthorized := handlers.resolveAuthorizedSite(ginContext)
	require.False(testingT, isAuthorized)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
}

func TestResolveAuthorizedSiteRejectsUnknownSite(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	handlers := &SiteHandlers{database: database}
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Params = gin.Params{{Key: "id", Value: testSiteID}}
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: testOwnerEmail, Role: RoleUser})

	_, _, isAuthorized := handlers.resolveAuthorizedSite(ginContext)
	require.False(testingT, isAuthorized)
	require.Equal(testingT, http.StatusNotFound, recorder.Code)
}

func TestResolveAuthorizedSiteRejectsForbidden(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            testSiteID,
		Name:          "Example Site",
		AllowedOrigin: testAllowedOriginPrimary,
		OwnerEmail:    testOwnerEmail,
	}
	require.NoError(testingT, database.Create(&site).Error)

	handlers := &SiteHandlers{database: database}
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Params = gin.Params{{Key: "id", Value: testSiteID}}
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: "other@example.com", Role: RoleUser})

	_, _, isAuthorized := handlers.resolveAuthorizedSite(ginContext)
	require.False(testingT, isAuthorized)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)
}

func TestResolveAuthorizedSiteReturnsSite(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            testSiteID,
		Name:          "Example Site",
		AllowedOrigin: testAllowedOriginPrimary,
		OwnerEmail:    testOwnerEmail,
	}
	require.NoError(testingT, database.Create(&site).Error)

	handlers := &SiteHandlers{database: database}
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Params = gin.Params{{Key: "id", Value: testSiteID}}
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ginContext.Set(contextKeyCurrentUser, &CurrentUser{Email: testOwnerEmail, Role: RoleUser})

	resolvedSite, resolvedUser, isAuthorized := handlers.resolveAuthorizedSite(ginContext)
	require.True(testingT, isAuthorized)
	require.Equal(testingT, testSiteID, resolvedSite.ID)
	require.Equal(testingT, testOwnerEmail, resolvedUser.Email)
}

func TestToSiteResponseUsesWidgetBaseURL(testingT *testing.T) {
	statsProvider := &stubStatsProvider{}
	handlers := &SiteHandlers{
		statsProvider: statsProvider,
		widgetBaseURL: "https://widget.example.com",
	}
	site := model.Site{
		ID:                         testSiteID,
		Name:                       testSiteName,
		AllowedOrigin:              testAllowedOriginPrimary,
		WidgetBubbleSide:           widgetBubbleSideRight,
		WidgetBubbleBottomOffsetPx: defaultWidgetBubbleBottomOffset,
	}

	response := handlers.toSiteResponse(context.Background(), site, 0)
	require.Contains(testingT, response.Widget, "https://widget.example.com")
}

func TestToSiteResponseUsesAllowedOriginWhenWidgetBaseMissing(testingT *testing.T) {
	statsProvider := &stubStatsProvider{}
	handlers := &SiteHandlers{
		statsProvider: statsProvider,
	}
	site := model.Site{
		ID:                         testSiteID,
		Name:                       testSiteName,
		AllowedOrigin:              testAllowedOriginPrimary,
		WidgetBubbleSide:           widgetBubbleSideRight,
		WidgetBubbleBottomOffsetPx: defaultWidgetBubbleBottomOffset,
	}

	response := handlers.toSiteResponse(context.Background(), site, 0)
	require.Contains(testingT, response.Widget, testAllowedOriginPrimary)
}

func TestToSiteResponseAddsFaviconURL(testingT *testing.T) {
	statsProvider := &stubStatsProvider{}
	handlers := &SiteHandlers{
		statsProvider: statsProvider,
	}
	site := model.Site{
		ID:                         testSiteID,
		Name:                       testSiteName,
		AllowedOrigin:              testAllowedOriginPrimary,
		WidgetBubbleSide:           widgetBubbleSideRight,
		WidgetBubbleBottomOffsetPx: defaultWidgetBubbleBottomOffset,
		FaviconData:                []byte{0x01},
		FaviconFetchedAt:           time.Now().UTC(),
	}

	response := handlers.toSiteResponse(context.Background(), site, 0)
	require.NotEmpty(testingT, response.FaviconURL)
}
