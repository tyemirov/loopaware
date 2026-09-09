package task_test

import (
	"context"
	"testing"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/task"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestVisitCountRetentionUsesInclusiveUTCDates(testingT *testing.T) {
	for _, testCase := range []struct {
		name     string
		days     int
		expected []string
	}{
		{"current day", 1, []string{"2026-09-08"}},
		{"two days", 2, []string{"2026-09-07", "2026-09-08"}},
	} {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			sqlite := testutil.NewSQLiteTestDatabase(testingT)
			database, err := storage.OpenDatabase(sqlite.Configuration())
			require.NoError(testingT, err)
			require.NoError(testingT, storage.AutoMigrate(database))
			for _, date := range []string{"2026-09-06", "2026-09-07", "2026-09-08"} {
				require.NoError(testingT, database.Create(&model.SiteVisitCount{SiteID: "site", Date: date, Count: 1}).Error)
			}
			now := time.Date(2026, 9, 7, 17, 30, 0, 0, time.FixedZone("Pacific", -7*60*60))
			require.NoError(testingT, task.PruneVisitCounts(context.Background(), database, now, testCase.days))
			var dates []string
			require.NoError(testingT, database.Model(&model.SiteVisitCount{}).Order("date").Pluck("date", &dates).Error)
			require.Equal(testingT, testCase.expected, dates)
		})
	}
}
