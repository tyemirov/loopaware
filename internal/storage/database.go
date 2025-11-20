package storage

import (
	"errors"
	"fmt"
	"strings"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	// DriverNameSQLite identifies the SQLite driver implementation.
	DriverNameSQLite = "sqlite"

	errorMessageMissingDatabaseDriverName = "storage: missing database driver name"
	errorMessageUnsupportedDatabaseDriver = "storage: unsupported database driver"
	errorMessageMissingDataSourceName     = "storage: missing database data source name"
	errorMessageOpenDatabase              = "storage: open database"
	errorMessageOpenSQLiteDatabase        = "storage: open sqlite database"
)

var (
	// ErrMissingDatabaseDriverName indicates the database driver name configuration was omitted.
	ErrMissingDatabaseDriverName = errors.New(errorMessageMissingDatabaseDriverName)
	// ErrUnsupportedDatabaseDriver indicates the provided database driver is not supported.
	ErrUnsupportedDatabaseDriver = errors.New(errorMessageUnsupportedDatabaseDriver)
	// ErrMissingDataSourceName indicates the database data source name configuration was omitted.
	ErrMissingDataSourceName = errors.New(errorMessageMissingDataSourceName)
)

type databaseOpener func(Config) (*gorm.DB, error)

var databaseOpeners = map[string]databaseOpener{
	DriverNameSQLite: openSQLiteDatabase,
}

// Config captures database connection configuration.
type Config struct {
	DriverName     string
	DataSourceName string
}

// OpenDatabase opens a database connection using the configured driver and data source name.
func OpenDatabase(configuration Config) (*gorm.DB, error) {
	trimmedDriverName := strings.TrimSpace(configuration.DriverName)
	if trimmedDriverName == "" {
		return nil, ErrMissingDatabaseDriverName
	}

	opener, driverSupported := databaseOpeners[trimmedDriverName]
	if !driverSupported {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedDatabaseDriver, trimmedDriverName)
	}

	database, openErr := opener(Config{
		DriverName:     trimmedDriverName,
		DataSourceName: strings.TrimSpace(configuration.DataSourceName),
	})
	if openErr != nil {
		return nil, fmt.Errorf("%s: %w", errorMessageOpenDatabase, openErr)
	}

	return database, nil
}

func openSQLiteDatabase(configuration Config) (*gorm.DB, error) {
	if configuration.DataSourceName == "" {
		return nil, ErrMissingDataSourceName
	}

	database, openErr := gorm.Open(sqlite.Open(configuration.DataSourceName), &gorm.Config{})
	if openErr != nil {
		return nil, fmt.Errorf("%s: %w", errorMessageOpenSQLiteDatabase, openErr)
	}

	return database, nil
}

// AutoMigrate runs database migrations for the storage layer models.
func AutoMigrate(database *gorm.DB) error {
	if err := database.AutoMigrate(&model.Site{}, &model.Feedback{}, &model.User{}, &model.Subscriber{}); err != nil {
		return err
	}
	return backfillSiteCreatorEmails(database)
}

// NewID generates a new globally unique identifier.
func NewID() string {
	return uuid.NewString()
}
