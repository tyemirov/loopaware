package task

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

func TestVisitRollupJobAggregatesAndPrunes(t *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)
	database, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, err)
	require.NoError(t, storage.AutoMigrate(database))

	siteID := storage.NewID()
	now := time.Now().UTC()
	yesterday := now.Add(-24 * time.Hour)
	visit1, _ := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/a",
		VisitorID: storage.NewID(),
		Occurred:  yesterday,
	})
	visit2, _ := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/b",
		VisitorID: storage.NewID(),
		Occurred:  yesterday,
	})
	// Old visit to prune
	oldVisit, _ := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/old",
		VisitorID: storage.NewID(),
		Occurred:  now.Add(-48 * time.Hour),
	})

	require.NoError(t, database.Create(&visit1).Error)
	require.NoError(t, database.Create(&visit2).Error)
	require.NoError(t, database.Create(&oldVisit).Error)

	job := NewVisitRollupJob(database, nil, VisitRollupConfig{RetentionDays: 1})
	require.NoError(t, job.Run(context.Background()))

	var rollups []model.SiteVisitRollup
	require.NoError(t, database.Find(&rollups).Error)
	require.Len(t, rollups, 1)
	require.Equal(t, int64(2), rollups[0].PageViews)
	require.Equal(t, int64(2), rollups[0].UniqueVisitors)

	var remainingVisits int64
	require.NoError(t, database.Model(&model.SiteVisit{}).Count(&remainingVisits).Error)
	require.Equal(t, int64(2), remainingVisits)
}
