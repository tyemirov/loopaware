package storage_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

const (
	testSiteNameValue                = "Test Site"
	testSiteAllowedOriginValue       = "http://localhost"
	testFeedbackContactValue         = "user@example.com"
	testFeedbackMessageValue         = "Hello"
	testFeedbackIPAddressValue       = "127.0.0.1"
	testFeedbackUserAgentValue       = "test-agent"
	testUnsupportedDriverName        = "unsupported-driver"
	testUnsupportedDriverDescription = "unsupported driver"
	testMissingDriverDescription     = "missing driver"
	testMissingDataSourceDescription = "missing data source"
	testOwnerEmailValue              = "owner@example.com"
	testExistingCreatorEmail         = "existing@example.com"
	testSubscriberEmailValue         = "subscriber@example.com"
	testSubscriberNameValue          = "Test User"
)

func TestOpenDatabaseWithSQLiteConfiguration(t *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)

	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, openErr)
	database = testutil.ConfigureDatabaseLogger(t, database)
	require.NotNil(t, database)

	require.NoError(t, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          testSiteNameValue,
		AllowedOrigin: testSiteAllowedOriginValue,
	}
	require.NoError(t, database.Create(&site).Error)

	feedback := model.Feedback{
		ID:        storage.NewID(),
		SiteID:    site.ID,
		Contact:   testFeedbackContactValue,
		Message:   testFeedbackMessageValue,
		IP:        testFeedbackIPAddressValue,
		UserAgent: testFeedbackUserAgentValue,
	}
	require.NoError(t, database.Create(&feedback).Error)

	var fetchedSite model.Site
	require.NoError(t, database.First(&fetchedSite, "id = ?", site.ID).Error)
	require.Equal(t, testSiteNameValue, fetchedSite.Name)
}

func TestAutoMigrateBackfillsMissingCreatorEmails(t *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)

	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, openErr)
	database = testutil.ConfigureDatabaseLogger(t, database)

	require.NoError(t, storage.AutoMigrate(database))

	missingCreatorSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Missing Creator",
		AllowedOrigin: testSiteAllowedOriginValue,
		OwnerEmail:    testOwnerEmailValue,
	}
	require.NoError(t, database.Create(&missingCreatorSite).Error)

	nullCreatorSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Null Creator",
		AllowedOrigin: testSiteAllowedOriginValue,
		OwnerEmail:    testOwnerEmailValue,
		CreatorEmail:  testExistingCreatorEmail,
	}
	require.NoError(t, database.Create(&nullCreatorSite).Error)
	require.NoError(t, database.Model(&model.Site{}).Where("id = ?", nullCreatorSite.ID).Update("creator_email", nil).Error)

	existingCreatorSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Existing Creator",
		AllowedOrigin: testSiteAllowedOriginValue,
		OwnerEmail:    testOwnerEmailValue,
		CreatorEmail:  testExistingCreatorEmail,
	}
	require.NoError(t, database.Create(&existingCreatorSite).Error)

	require.NoError(t, storage.AutoMigrate(database))

	var refreshedMissing model.Site
	require.NoError(t, database.First(&refreshedMissing, "id = ?", missingCreatorSite.ID).Error)
	require.Equal(t, storage.DefaultSiteCreatorEmail, refreshedMissing.CreatorEmail)

	var refreshedNull model.Site
	require.NoError(t, database.First(&refreshedNull, "id = ?", nullCreatorSite.ID).Error)
	require.Equal(t, storage.DefaultSiteCreatorEmail, refreshedNull.CreatorEmail)

	var refreshedExisting model.Site
	require.NoError(t, database.First(&refreshedExisting, "id = ?", existingCreatorSite.ID).Error)
	require.Equal(t, testExistingCreatorEmail, refreshedExisting.CreatorEmail)
}

func TestAutoMigrateBackfillsWidgetFeedbackVisibility(t *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)

	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, openErr)
	database = testutil.ConfigureDatabaseLogger(t, database)

	require.NoError(t, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Hidden Feedback Inputs",
		AllowedOrigin: testSiteAllowedOriginValue,
		OwnerEmail:    testOwnerEmailValue,
	}
	require.NoError(t, database.Create(&site).Error)
	require.NoError(t, database.Model(&model.Site{}).Where("id = ?", site.ID).Updates(map[string]any{
		"widget_show_message_input":     false,
		"widget_show_sentiment_buttons": false,
	}).Error)

	require.NoError(t, storage.AutoMigrate(database))

	var refreshedSite model.Site
	require.NoError(t, database.First(&refreshedSite, "id = ?", site.ID).Error)
	require.True(t, refreshedSite.WidgetShowMessageInput)
	require.True(t, refreshedSite.WidgetShowSentimentButtons)
}

func TestAutoMigrateBackfillsWidgetAccentColor(t *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)

	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, openErr)
	database = testutil.ConfigureDatabaseLogger(t, database)

	require.NoError(t, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Missing Accent",
		AllowedOrigin: testSiteAllowedOriginValue,
		OwnerEmail:    testOwnerEmailValue,
	}
	require.NoError(t, database.Create(&site).Error)
	require.NoError(t, database.Model(&model.Site{}).Where("id = ?", site.ID).Update("widget_accent_color", "").Error)

	require.NoError(t, storage.AutoMigrate(database))

	var refreshedSite model.Site
	require.NoError(t, database.First(&refreshedSite, "id = ?", site.ID).Error)
	require.Equal(t, "#0d6efd", refreshedSite.WidgetAccentColor)
}

func TestOpenDatabaseValidation(t *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)

	testCases := []struct {
		name              string
		configuration     storage.Config
		expectedRootError error
	}{
		{
			name: testMissingDriverDescription,
			configuration: storage.Config{
				DriverName:     "",
				DataSourceName: sqliteDatabase.DataSourceName(),
			},
			expectedRootError: storage.ErrMissingDatabaseDriverName,
		},
		{
			name: testUnsupportedDriverDescription,
			configuration: storage.Config{
				DriverName:     testUnsupportedDriverName,
				DataSourceName: sqliteDatabase.DataSourceName(),
			},
			expectedRootError: storage.ErrUnsupportedDatabaseDriver,
		},
		{
			name: testMissingDataSourceDescription,
			configuration: storage.Config{
				DriverName:     storage.DriverNameSQLite,
				DataSourceName: "",
			},
			expectedRootError: storage.ErrMissingDataSourceName,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(testingT *testing.T) {
			_, openErr := storage.OpenDatabase(testCase.configuration)
			require.Error(testingT, openErr)
			require.True(testingT, errors.Is(openErr, testCase.expectedRootError))
		})
	}
}

func TestSubscriberUniqueEmailPerSiteAudience(t *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)

	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, openErr)
	database = testutil.ConfigureDatabaseLogger(t, database)

	require.NoError(t, storage.AutoMigrate(database))

	firstSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Site One",
		AllowedOrigin: testSiteAllowedOriginValue,
	}
	secondSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Site Two",
		AllowedOrigin: testSiteAllowedOriginValue,
	}
	require.NoError(t, database.Create(&firstSite).Error)
	require.NoError(t, database.Create(&secondSite).Error)

	firstSubscriber, err := model.NewSubscriber(model.SubscriberInput{
		SiteID:      firstSite.ID,
		Email:       testSubscriberEmailValue,
		Name:        testSubscriberNameValue,
		AudienceKey: "EBAY",
	})
	require.NoError(t, err)
	require.NoError(t, database.Create(&firstSubscriber).Error)

	duplicateSubscriber, err := model.NewSubscriber(model.SubscriberInput{
		SiteID:      firstSite.ID,
		Email:       testSubscriberEmailValue,
		AudienceKey: "EBAY",
	})
	require.NoError(t, err)
	duplicateErr := database.Create(&duplicateSubscriber).Error
	require.Error(t, duplicateErr)

	otherAudienceSubscriber, err := model.NewSubscriber(model.SubscriberInput{
		SiteID:      firstSite.ID,
		Email:       testSubscriberEmailValue,
		AudienceKey: "WLMT",
	})
	require.NoError(t, err)
	require.NoError(t, database.Create(&otherAudienceSubscriber).Error)

	otherSiteSubscriber, err := model.NewSubscriber(model.SubscriberInput{
		SiteID:      secondSite.ID,
		Email:       testSubscriberEmailValue,
		AudienceKey: "EBAY",
	})
	require.NoError(t, err)
	require.NoError(t, database.Create(&otherSiteSubscriber).Error)
}

func TestAutoMigrateBackfillsSubscriberAudienceAndDropsLegacyIndex(t *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)

	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, openErr)
	database = testutil.ConfigureDatabaseLogger(t, database)

	require.NoError(t, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Legacy Subscribers",
		AllowedOrigin: testSiteAllowedOriginValue,
	}
	require.NoError(t, database.Create(&site).Error)

	legacySubscriber, err := model.NewSubscriber(model.SubscriberInput{
		SiteID:    site.ID,
		Email:     testSubscriberEmailValue,
		SourceURL: "http://scanner.example/settings?waitlist_platform=ebay",
	})
	require.NoError(t, err)
	require.NoError(t, database.Create(&legacySubscriber).Error)
	require.NoError(t, database.Exec("CREATE UNIQUE INDEX idx_subscribers_site_email ON subscribers(site_id, email)").Error)

	require.NoError(t, storage.AutoMigrate(database))

	var refreshedSubscriber model.Subscriber
	require.NoError(t, database.First(&refreshedSubscriber, "id = ?", legacySubscriber.ID).Error)
	require.Equal(t, "EBAY", refreshedSubscriber.AudienceKey)

	otherAudienceSubscriber, err := model.NewSubscriber(model.SubscriberInput{
		SiteID:      site.ID,
		Email:       testSubscriberEmailValue,
		AudienceKey: "WLMT",
	})
	require.NoError(t, err)
	require.NoError(t, database.Create(&otherAudienceSubscriber).Error)
}
