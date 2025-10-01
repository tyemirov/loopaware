package testutil

import (
	"fmt"
	"testing"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
)

const (
	sqliteTestDatabaseNamePrefix        = "loopaware-test-db"
	sqliteInMemoryDataSourceNamePattern = "file:%s?mode=memory&cache=shared&_foreign_keys=on"
)

// SQLiteTestDatabase provides helpers for configuring temporary SQLite databases in tests.
type SQLiteTestDatabase struct {
	configuration storage.Config
}

// NewSQLiteTestDatabase creates a SQLiteTestDatabase with a unique in-memory database configuration.
func NewSQLiteTestDatabase(testingT *testing.T) SQLiteTestDatabase {
	testingT.Helper()

	databaseName := fmt.Sprintf("%s-%s", sqliteTestDatabaseNamePrefix, storage.NewID())

	return SQLiteTestDatabase{
		configuration: storage.Config{
			DriverName:     storage.DriverNameSQLite,
			DataSourceName: fmt.Sprintf(sqliteInMemoryDataSourceNamePattern, databaseName),
		},
	}
}

// Configuration returns the storage configuration for the temporary SQLite database.
func (database SQLiteTestDatabase) Configuration() storage.Config {
	return database.configuration
}

// DataSourceName returns the SQLite data source name for the temporary database.
func (database SQLiteTestDatabase) DataSourceName() string {
	return database.configuration.DataSourceName
}
