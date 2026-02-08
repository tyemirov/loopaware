package task

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

func TestVisitRollupJobSkipsPruneWhenRetentionZero(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	require.NoError(testingT, storage.AutoMigrate(database))

	siteID := storage.NewID()
	now := time.Now().UTC()
	yesterday := now.Add(-24 * time.Hour)
	visitRecent, _ := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/recent",
		VisitorID: storage.NewID(),
		Occurred:  yesterday,
	})
	visitOld, _ := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/old",
		VisitorID: storage.NewID(),
		Occurred:  now.Add(-72 * time.Hour),
	})

	require.NoError(testingT, database.Create(&visitRecent).Error)
	require.NoError(testingT, database.Create(&visitOld).Error)

	job := NewVisitRollupJob(database, zap.NewNop(), VisitRollupConfig{})
	require.NoError(testingT, job.Run(context.Background()))

	var visitCount int64
	require.NoError(testingT, database.Model(&model.SiteVisit{}).Count(&visitCount).Error)
	require.Equal(testingT, int64(2), visitCount)
}

func TestAggregatePreviousDaySkipsInvalidSiteID(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	require.NoError(testingT, storage.AutoMigrate(database))

	now := time.Now().UTC()
	visit := model.SiteVisit{
		ID:         storage.NewID(),
		SiteID:     " ",
		URL:        "https://example.com",
		Path:       "/",
		Status:     model.VisitStatusRecorded,
		OccurredAt: now.Add(-24 * time.Hour),
	}
	require.NoError(testingT, database.Create(&visit).Error)

	observedCore, observedLogs := observer.New(zap.WarnLevel)
	logger := zap.New(observedCore)
	job := NewVisitRollupJob(database, logger, VisitRollupConfig{RetentionDays: 1})
	require.NoError(testingT, job.aggregatePreviousDay(context.Background()))

	var rollupCount int64
	require.NoError(testingT, database.Model(&model.SiteVisitRollup{}).Count(&rollupCount).Error)
	require.Equal(testingT, int64(0), rollupCount)
	require.GreaterOrEqual(testingT, observedLogs.Len(), 1)
}

func TestAggregatePreviousDayReturnsErrorForCanceledContext(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	require.NoError(testingT, storage.AutoMigrate(database))

	job := NewVisitRollupJob(database, zap.NewNop(), VisitRollupConfig{})
	requestContext, cancel := context.WithCancel(context.Background())
	cancel()

	aggregateErr := job.aggregatePreviousDay(requestContext)
	require.Error(testingT, aggregateErr)
}

func TestPruneOldVisitsSkipsWhenRetentionDisabled(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	require.NoError(testingT, storage.AutoMigrate(database))

	job := NewVisitRollupJob(database, zap.NewNop(), VisitRollupConfig{})
	pruneErr := job.pruneOldVisits(context.Background())
	require.NoError(testingT, pruneErr)
}

func TestVisitRollupJobRunPrunesOldVisits(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	require.NoError(testingT, storage.AutoMigrate(database))

	siteID := storage.NewID()
	now := time.Now().UTC()

	recentVisit, recentErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/recent",
		VisitorID: storage.NewID(),
		Occurred:  now,
	})
	require.NoError(testingT, recentErr)
	oldVisit, oldErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/old",
		VisitorID: storage.NewID(),
		Occurred:  now.Add(-72 * time.Hour),
	})
	require.NoError(testingT, oldErr)

	require.NoError(testingT, database.Create(&recentVisit).Error)
	require.NoError(testingT, database.Create(&oldVisit).Error)

	job := NewVisitRollupJob(database, zap.NewNop(), VisitRollupConfig{RetentionDays: 1})
	runErr := job.Run(context.Background())
	require.NoError(testingT, runErr)

	var visitCount int64
	require.NoError(testingT, database.Model(&model.SiteVisit{}).Count(&visitCount).Error)
	require.Equal(testingT, int64(1), visitCount)
}

func TestVisitRollupJobRunReturnsErrorWhenContextCanceled(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	require.NoError(testingT, storage.AutoMigrate(database))

	job := NewVisitRollupJob(database, zap.NewNop(), VisitRollupConfig{RetentionDays: 1})
	requestContext, cancel := context.WithCancel(context.Background())
	cancel()

	runErr := job.Run(requestContext)
	require.Error(testingT, runErr)
}
