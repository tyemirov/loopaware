package testutil_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/testutil"
)

const (
	testCaseDescriptionMemoryModeParameter  = "includes memory mode parameter"
	testCaseDescriptionSharedCacheParameter = "includes shared cache parameter"
	testCaseDescriptionForeignKeysParameter = "enforces foreign keys"
	sqliteModeMemoryParameter               = "mode=memory"
	sqliteSharedCacheParameter              = "cache=shared"
	sqliteForeignKeysParameter              = "_foreign_keys=on"
)

func TestNewSQLiteTestDatabaseProvidesInMemoryConfiguration(t *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)
	configuration := sqliteDatabase.Configuration()

	require.Equal(t, storage.DriverNameSQLite, configuration.DriverName)

	testCases := []struct {
		name              string
		expectedSubstring string
	}{
		{name: testCaseDescriptionMemoryModeParameter, expectedSubstring: sqliteModeMemoryParameter},
		{name: testCaseDescriptionSharedCacheParameter, expectedSubstring: sqliteSharedCacheParameter},
		{name: testCaseDescriptionForeignKeysParameter, expectedSubstring: sqliteForeignKeysParameter},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(testingT *testing.T) {
			require.Contains(testingT, configuration.DataSourceName, testCase.expectedSubstring)
		})
	}
}

func TestNewSQLiteTestDatabaseReturnsUniqueDataSourceNames(t *testing.T) {
	firstDatabase := testutil.NewSQLiteTestDatabase(t)
	secondDatabase := testutil.NewSQLiteTestDatabase(t)

	require.NotEqual(t, firstDatabase.DataSourceName(), secondDatabase.DataSourceName())
}
