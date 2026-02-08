package api

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

func TestSiteHandlersScheduleFaviconFetchSkipsNilManager(testingT *testing.T) {
	handlers := &SiteHandlers{}
	handlers.scheduleFaviconFetch(model.Site{ID: "site-id"})
}

func TestSiteHandlersScheduleFaviconFetchCallsManager(testingT *testing.T) {
	handlers := &SiteHandlers{faviconManager: &SiteFaviconManager{}}
	handlers.scheduleFaviconFetch(model.Site{ID: "site-id"})
}

func TestAllowedOriginConflictExistsReportsDatabaseError(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Primary Site",
		AllowedOrigin: testAllowedOriginPrimary,
		OwnerEmail:    testOwnerEmail,
	}
	require.NoError(testingT, database.Create(&site).Error)

	sqlDatabase, sqlErr := database.DB()
	require.NoError(testingT, sqlErr)
	require.NoError(testingT, sqlDatabase.Close())

	handlers := &SiteHandlers{database: database}
	conflict, conflictErr := handlers.allowedOriginConflictExists(testAllowedOriginPrimary, "")
	require.Error(testingT, conflictErr)
	require.False(testingT, conflict)
}
