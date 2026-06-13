package storage

import (
	"errors"
	"fmt"
	"net/url"
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
	if err := dropLegacySubscriberUniqueIndex(database); err != nil {
		return err
	}
	if err := dropLegacyPortfolioTrafficScheduleUserEmailIndex(database); err != nil {
		return err
	}
	if err := database.AutoMigrate(&model.Site{}, &model.Feedback{}, &model.User{}, &model.Subscriber{}, &model.SiteVisit{}, &model.SiteVisitRollup{}, &model.TrafficReportSchedule{}, &model.PortfolioTrafficReportSchedule{}, &model.PortfolioTrafficReportDefinition{}, &model.PortfolioTrafficReportDefinitionSite{}, &model.SentryIssue{}, &model.SentryOccurrence{}, &model.SiteMobileApp{}, &model.SiteTeamMember{}); err != nil {
		return err
	}
	if err := backfillPortfolioTrafficReportScheduleIDs(database); err != nil {
		return err
	}
	if err := backfillSiteCreatorEmails(database); err != nil {
		return err
	}
	if err := backfillSubscriberAudienceKeys(database); err != nil {
		return err
	}
	if err := backfillSiteWidgetFeedbackVisibility(database); err != nil {
		return err
	}
	if err := backfillSiteWidgetAccentColors(database); err != nil {
		return err
	}
	return backfillFeedbackSourceKinds(database)
}

func dropLegacySubscriberUniqueIndex(database *gorm.DB) error {
	if !database.Migrator().HasIndex(&model.Subscriber{}, "idx_subscribers_site_email") {
		return nil
	}
	return database.Migrator().DropIndex(&model.Subscriber{}, "idx_subscribers_site_email")
}

func dropLegacyPortfolioTrafficScheduleUserEmailIndex(database *gorm.DB) error {
	indexNames := []string{
		"idx_portfolio_traffic_report_schedules_user_email",
		"idx_portfolio_traffic_report_schedule_user_email",
	}
	for _, indexName := range indexNames {
		if database.Migrator().HasIndex(&model.PortfolioTrafficReportSchedule{}, indexName) {
			if err := database.Migrator().DropIndex(&model.PortfolioTrafficReportSchedule{}, indexName); err != nil {
				return err
			}
		}
	}
	return nil
}

func backfillPortfolioTrafficReportScheduleIDs(database *gorm.DB) error {
	if !database.Migrator().HasColumn(&model.PortfolioTrafficReportSchedule{}, "report_id") {
		return nil
	}
	return database.Model(&model.PortfolioTrafficReportSchedule{}).
		Where("report_id = ? OR report_id IS NULL", "").
		Update("report_id", model.PortfolioTrafficReportDefaultID).Error
}

func backfillSubscriberAudienceKeys(database *gorm.DB) error {
	type subscriberAudienceBackfillRecord struct {
		ID        string
		SourceURL string
	}

	var subscribers []subscriberAudienceBackfillRecord
	if err := database.Model(&model.Subscriber{}).
		Select("id", "source_url").
		Where("audience_key = ? OR audience_key = '' OR audience_key IS NULL", model.SubscriberAudienceDefault).
		Find(&subscribers).Error; err != nil {
		return err
	}
	for _, subscriber := range subscribers {
		audienceKey := subscriberAudienceKeyFromSourceURL(subscriber.SourceURL)
		if err := database.Model(&model.Subscriber{}).
			Where("id = ?", subscriber.ID).
			Update("audience_key", audienceKey).Error; err != nil {
			return err
		}
	}
	return nil
}

func subscriberAudienceKeyFromSourceURL(sourceURL string) string {
	parsedURL, parseErr := url.Parse(strings.TrimSpace(sourceURL))
	if parseErr != nil || parsedURL == nil {
		return model.SubscriberAudienceDefault
	}
	audienceKey, audienceErr := model.NormalizeSubscriberAudienceKey(parsedURL.Query().Get("waitlist_platform"))
	if audienceErr != nil {
		return model.SubscriberAudienceDefault
	}
	return audienceKey
}

// NewID generates a new globally unique identifier.
func NewID() string {
	return uuid.NewString()
}
