package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
)

func TestTestingLogWriterWrite(testingT *testing.T) {
	testCases := []struct {
		name        string
		inputBytes  []byte
		expectCount int
	}{
		{
			name:        "records message",
			inputBytes:  []byte(" database log \n"),
			expectCount: len([]byte(" database log \n")),
		},
		{
			name:        "ignores empty message",
			inputBytes:  []byte("   \n"),
			expectCount: len([]byte("   \n")),
		},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			writer := testingLogWriter{testingT: testingT}
			count, writeErr := writer.Write(testCase.inputBytes)
			require.NoError(testingT, writeErr)
			require.Equal(testingT, testCase.expectCount, count)
		})
	}
}

func TestConfigureDatabaseLoggerReturnsSession(testingT *testing.T) {
	sqliteDatabase := NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)

	configured := ConfigureDatabaseLogger(testingT, database)
	require.NotNil(testingT, configured)
	require.NotNil(testingT, configured.Logger)
}

func TestSQLiteTestDatabaseDataSourceName(testingT *testing.T) {
	sqliteDatabase := NewSQLiteTestDatabase(testingT)
	require.Equal(testingT, sqliteDatabase.Configuration().DataSourceName, sqliteDatabase.DataSourceName())
}
