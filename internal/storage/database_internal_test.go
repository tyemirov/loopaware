package storage

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const testOpenDatabaseFailureMessage = "open failure"

func TestOpenDatabaseWrapsOpenerError(testingT *testing.T) {
	originalOpeners := databaseOpeners
	testingT.Cleanup(func() {
		databaseOpeners = originalOpeners
	})

	databaseOpeners = map[string]databaseOpener{
		DriverNameSQLite: func(Config) (*gorm.DB, error) {
			return nil, errors.New(testOpenDatabaseFailureMessage)
		},
	}

	_, openErr := OpenDatabase(Config{
		DriverName:     DriverNameSQLite,
		DataSourceName: "file:invalid",
	})
	require.Error(testingT, openErr)
	require.Contains(testingT, openErr.Error(), errorMessageOpenDatabase)
}

func TestOpenSQLiteDatabaseReportsOpenError(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	missingDirectory := filepath.Join(tempDirectory, "missing")
	dataSourceName := fmt.Sprintf("file:%s?mode=rwc&_foreign_keys=on", filepath.Join(missingDirectory, "test.db"))

	_, openErr := openSQLiteDatabase(Config{DataSourceName: dataSourceName})
	require.Error(testingT, openErr)
	require.Contains(testingT, openErr.Error(), errorMessageOpenSQLiteDatabase)
}
