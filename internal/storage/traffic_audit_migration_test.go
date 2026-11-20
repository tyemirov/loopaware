package storage_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

const (
	trafficAuditFixtureCount = 1103
	trafficAuditMarkerTable  = "traffic_audit_migrations"
)

func TestTrafficIdentityBrowserAuditMigrationPreservesVisitEvidence(t *testing.T) {
	database := newTrafficAuditDatabase(t)
	visits := trafficAuditVisits(t)
	require.NoError(t, database.CreateInBatches(&visits, 100).Error)
	before := trafficAuditStoredVisits(t, database)

	changed, migrationErr := storage.MigrateBrowserAuditVisits(database)
	require.NoError(t, migrationErr)
	require.EqualValues(t, trafficAuditFixtureCount, changed)
	after := trafficAuditStoredVisits(t, database)
	require.Len(t, after, len(before))
	for visitID, original := range before {
		expected := original
		if model.IsBrowserAuditUserAgent(original.UserAgent) {
			expected.IsBot = true
		}
		require.Equal(t, expected, after[visitID], "visit %s must retain every other field", visitID)
	}
	require.EqualValues(t, 1, trafficAuditMarkerCount(t, database))

	newVisit := trafficAuditVisit(t, 9000, "Mozilla/5.0 HeadlessChrome/131.0")
	require.NoError(t, database.Create(&newVisit).Error)
	changed, migrationErr = storage.MigrateBrowserAuditVisits(database)
	require.NoError(t, migrationErr)
	require.Zero(t, changed)
	var storedNewVisit model.SiteVisit
	require.NoError(t, database.First(&storedNewVisit, "id = ?", newVisit.ID).Error)
	require.Equal(t, newVisit, storedNewVisit, "completion prevents later startup reclassification")
	require.EqualValues(t, 1, trafficAuditMarkerCount(t, database))
	t.Logf("migration corrected %d historical visits across multiple batches; rerun corrected %d", trafficAuditFixtureCount, changed)
}

func TestTrafficIdentityBrowserAuditMigrationRollsBackFailedBatch(t *testing.T) {
	database := newTrafficAuditDatabase(t)
	database.Logger = database.Logger.LogMode(logger.Silent)
	visits := trafficAuditVisits(t)
	require.NoError(t, database.CreateInBatches(&visits, 100).Error)
	before := trafficAuditStoredVisits(t, database)
	injectedFailure := errors.New("injected historical browser-audit update failure")
	updates := 0
	callbackRegistered := true
	const callbackName = "test:fail_browser_audit_batch"
	require.NoError(t, database.Callback().Update().After("gorm:update").Register(callbackName, func(transaction *gorm.DB) {
		if transaction.Statement.Table != "site_visits" {
			return
		}
		updates++
		if updates == 2 {
			transaction.AddError(injectedFailure)
		}
	}))
	t.Cleanup(func() {
		if callbackRegistered {
			require.NoError(t, database.Callback().Update().Remove(callbackName))
		}
	})

	changed, migrationErr := storage.MigrateBrowserAuditVisits(database)
	require.ErrorIs(t, migrationErr, injectedFailure)
	require.Contains(t, migrationErr.Error(), "browser-audit")
	require.Zero(t, changed, "rolled-back changes must never be reported as corrected rows")
	require.Equal(t, 2, updates, "failure must occur after an earlier successful batch")
	require.Equal(t, before, trafficAuditStoredVisits(t, database))
	require.Zero(t, trafficAuditMarkerCount(t, database))

	require.NoError(t, database.Callback().Update().Remove(callbackName))
	callbackRegistered = false
	changed, migrationErr = storage.MigrateBrowserAuditVisits(database)
	require.NoError(t, migrationErr)
	require.EqualValues(t, trafficAuditFixtureCount, changed)
	require.EqualValues(t, 1, trafficAuditMarkerCount(t, database))
	changed, migrationErr = storage.MigrateBrowserAuditVisits(database)
	require.NoError(t, migrationErr)
	require.Zero(t, changed)
}

func TestTrafficIdentityBrowserAuditMigrationRollsBackCompletionFailure(t *testing.T) {
	database := newTrafficAuditDatabase(t)
	database.Logger = database.Logger.LogMode(logger.Silent)
	visits := trafficAuditVisits(t)
	require.NoError(t, database.CreateInBatches(&visits, 100).Error)
	before := trafficAuditStoredVisits(t, database)
	injectedFailure := errors.New("injected historical browser-audit completion failure")
	const callbackName = "test:fail_browser_audit_completion"
	require.NoError(t, database.Callback().Create().After("gorm:create").Register(callbackName, func(transaction *gorm.DB) {
		if transaction.Statement.Table == trafficAuditMarkerTable {
			transaction.AddError(injectedFailure)
		}
	}))
	t.Cleanup(func() {
		require.NoError(t, database.Callback().Create().Remove(callbackName))
	})

	changed, migrationErr := storage.MigrateBrowserAuditVisits(database)
	require.ErrorIs(t, migrationErr, injectedFailure)
	require.Zero(t, changed)
	require.Equal(t, before, trafficAuditStoredVisits(t, database))
	require.Zero(t, trafficAuditMarkerCount(t, database))
}

func newTrafficAuditDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	fixture := testutil.NewSQLiteTestDatabase(t)
	database, openErr := storage.OpenDatabase(fixture.Configuration())
	require.NoError(t, openErr)
	database = testutil.ConfigureDatabaseLogger(t, database)
	require.NoError(t, storage.AutoMigrate(database))
	return database
}

func trafficAuditVisits(t *testing.T) []model.SiteVisit {
	t.Helper()
	auditUserAgents := []string{
		"Mozilla/5.0 HeadlessChrome/131.0 Safari/537.36",
		"Mozilla/5.0 (compatible; GOOGLE-INSPECTIONTOOL/1.0)",
		"Mozilla/5.0 Chrome-Lighthouse Chrome/131.0",
	}
	visits := make([]model.SiteVisit, 0, trafficAuditFixtureCount+4)
	for index := 0; index < trafficAuditFixtureCount; index++ {
		visits = append(visits, trafficAuditVisit(t, index+1, auditUserAgents[index%len(auditUserAgents)]))
	}
	for index, userAgent := range []string{
		"Mozilla/5.0 Chrome/131.0 Safari/537.36",
		"WhatsApp/2.24.10",
		"Googlebot/2.1",
		"Mozilla/5.0 Firefox/133.0",
	} {
		visit := trafficAuditVisit(t, 5000+index, userAgent)
		visit.IsBot = index == 3
		visits = append(visits, visit)
	}
	return visits
}

func trafficAuditVisit(t *testing.T, index int, userAgent string) model.SiteVisit {
	t.Helper()
	visit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID: "historical-site", URL: fmt.Sprintf("https://example.com/page/%d?source=audit", index),
		VisitorID: "00000000-0000-0000-0000-000000000001", IP: "192.168.96.5",
		UserAgent: userAgent, Referrer: "https://referrer.example.com/", ScreenResolution: "1920x1080",
		Viewport: "1280x800", Timezone: "America/Los_Angeles", Locale: "en-US",
		GeoSource: "edge_geo", GeoCountry: "US", GeoRegion: "California", GeoCity: "San Francisco",
		GeoLatitude: 37.7749, GeoLongitude: -122.4194, PageTitle: "Historical page",
		Occurred: time.Date(2026, time.November, 18, 12, 30, index%60, 0, time.UTC),
	})
	require.NoError(t, visitErr)
	visit.ID = fmt.Sprintf("%036d", index)
	return visit
}

func trafficAuditStoredVisits(t *testing.T, database *gorm.DB) map[string]model.SiteVisit {
	t.Helper()
	var visits []model.SiteVisit
	require.NoError(t, database.Order("id").Find(&visits).Error)
	stored := make(map[string]model.SiteVisit, len(visits))
	for _, visit := range visits {
		stored[visit.ID] = visit
	}
	return stored
}

func trafficAuditMarkerCount(t *testing.T, database *gorm.DB) int64 {
	t.Helper()
	var count int64
	require.NoError(t, database.Table(trafficAuditMarkerTable).Count(&count).Error)
	return count
}
