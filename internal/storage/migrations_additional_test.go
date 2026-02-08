package storage

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

func TestAutoMigrateReportsErrorOnClosedDatabase(testingT *testing.T) {
	dataSourceName := fmt.Sprintf("file:%s?mode=memory&cache=shared&_foreign_keys=on", strings.ReplaceAll(testingT.Name(), "/", "_"))
	database, openErr := OpenDatabase(Config{
		DriverName:     DriverNameSQLite,
		DataSourceName: dataSourceName,
	})
	require.NoError(testingT, openErr)

	sqlDatabase, sqlErr := database.DB()
	require.NoError(testingT, sqlErr)
	require.NoError(testingT, sqlDatabase.Close())

	migrateErr := AutoMigrate(database)
	require.Error(testingT, migrateErr)
}

func TestBackfillSiteCreatorEmailsReportsErrorOnClosedDatabase(testingT *testing.T) {
	dataSourceName := fmt.Sprintf("file:%s?mode=memory&cache=shared&_foreign_keys=on", strings.ReplaceAll(testingT.Name(), "/", "_"))
	database, openErr := OpenDatabase(Config{
		DriverName:     DriverNameSQLite,
		DataSourceName: dataSourceName,
	})
	require.NoError(testingT, openErr)

	sqlDatabase, sqlErr := database.DB()
	require.NoError(testingT, sqlErr)
	require.NoError(testingT, sqlDatabase.Close())

	backfillErr := backfillSiteCreatorEmails(database)
	require.Error(testingT, backfillErr)
}

func TestOpenSQLiteDatabaseRequiresDataSourceName(testingT *testing.T) {
	database, openErr := openSQLiteDatabase(Config{DriverName: DriverNameSQLite})
	require.ErrorIs(testingT, openErr, ErrMissingDataSourceName)
	require.Nil(testingT, database)
}

func TestBackfillSiteCreatorEmailsUpdatesMissingCreator(testingT *testing.T) {
	dataSourceName := fmt.Sprintf("file:%s?mode=memory&cache=shared&_foreign_keys=on", strings.ReplaceAll(testingT.Name(), "/", "_"))
	database, openErr := OpenDatabase(Config{
		DriverName:     DriverNameSQLite,
		DataSourceName: dataSourceName,
	})
	require.NoError(testingT, openErr)
	require.NoError(testingT, AutoMigrate(database))

	site := model.Site{
		ID:            NewID(),
		Name:          "Missing Creator",
		AllowedOrigin: "https://creator.example",
		OwnerEmail:    "owner@example.com",
	}
	require.NoError(testingT, database.Create(&site).Error)

	backfillErr := backfillSiteCreatorEmails(database)
	require.NoError(testingT, backfillErr)

	var updated model.Site
	require.NoError(testingT, database.First(&updated, "id = ?", site.ID).Error)
	require.Equal(testingT, strings.ToLower(DefaultSiteCreatorEmail), updated.CreatorEmail)
}
