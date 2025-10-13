package testutil

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

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

type testingLogWriter struct {
	testingT *testing.T
}

func (writer testingLogWriter) Write(data []byte) (int, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed != "" {
		writer.testingT.Log(trimmed)
	}
	return len(data), nil
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

// ConfigureDatabaseLogger returns a database session that suppresses record-not-found logs during tests.
func ConfigureDatabaseLogger(testingT *testing.T, database *gorm.DB) *gorm.DB {
	testingT.Helper()
	if database == nil {
		testingT.Fatalf("configure database logger: nil database")
	}
	gormLogger := logger.New(
		log.New(testingLogWriter{testingT: testingT}, "", 0),
		logger.Config{
			IgnoreRecordNotFoundError: true,
			LogLevel:                  logger.Error,
		},
	)
	return database.Session(&gorm.Session{Logger: gormLogger})
}
