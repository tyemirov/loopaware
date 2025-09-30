package testutil

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
)

const (
	sqliteTestDatabaseFileName  = "loopaware-test.sqlite"
	sqliteDataSourceNamePattern = "file:%s?_foreign_keys=on"
)

// SQLiteTestDatabase provides helpers for configuring temporary SQLite databases in tests.
type SQLiteTestDatabase struct {
	configuration storage.Config
}

// NewSQLiteTestDatabase creates a SQLiteTestDatabase with a unique temporary database file.
func NewSQLiteTestDatabase(testingT *testing.T) SQLiteTestDatabase {
	testingT.Helper()

	temporaryDirectory := testingT.TempDir()
	databaseFilePath := filepath.Join(temporaryDirectory, sqliteTestDatabaseFileName)

	return SQLiteTestDatabase{
		configuration: storage.Config{
			DriverName:     storage.DriverNameSQLite,
			DataSourceName: fmt.Sprintf(sqliteDataSourceNamePattern, databaseFilePath),
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
