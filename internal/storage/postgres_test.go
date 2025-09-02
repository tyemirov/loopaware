package storage_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/testutil"
)

func TestOpenAndMigratePostgres(t *testing.T) {
	dsn := testutil.DSN()
	require.NotEmpty(t, dsn)

	db, openErr := storage.OpenPostgres(dsn)
	require.NoError(t, openErr)
	require.NotNil(t, db)

	require.NoError(t, storage.AutoMigrate(db))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Test Site",
		AllowedOrigin: "http://localhost",
	}
	require.NoError(t, db.Create(&site).Error)

	feedback := model.Feedback{
		ID:        storage.NewID(),
		SiteID:    site.ID,
		Contact:   "user@example.com",
		Message:   "Hello",
		IP:        "127.0.0.1",
		UserAgent: "test",
	}
	require.NoError(t, db.Create(&feedback).Error)

	var fetchedSite model.Site
	require.NoError(t, db.First(&fetchedSite, "id = ?", site.ID).Error)
	require.Equal(t, site.Name, fetchedSite.Name)
}
