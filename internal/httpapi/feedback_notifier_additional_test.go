package httpapi

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

type stubFeedbackNotifier struct {
	delivery  string
	notifyErr error
}

func (stub stubFeedbackNotifier) NotifyFeedback(ctx context.Context, site model.Site, feedback model.Feedback) (string, error) {
	return stub.delivery, stub.notifyErr
}

func TestApplyFeedbackNotificationSkipsNilInputs(testingT *testing.T) {
	applyFeedbackNotification(context.Background(), nil, nil, nil, model.Site{}, nil)
}

func TestApplyFeedbackNotificationUpdatesDelivery(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Feedback Site",
		AllowedOrigin: "https://example.com",
		OwnerEmail:    "owner@example.com",
	}
	require.NoError(testingT, database.Create(&site).Error)

	feedback := model.Feedback{
		ID:       storage.NewID(),
		SiteID:   site.ID,
		Contact:  "user@example.com",
		Message:  "Hello",
		Delivery: model.FeedbackDeliveryNone,
	}
	require.NoError(testingT, database.Create(&feedback).Error)

	notifier := stubFeedbackNotifier{delivery: model.FeedbackDeliveryTexted}
	applyFeedbackNotification(context.Background(), database, zap.NewNop(), notifier, site, &feedback)

	var refreshed model.Feedback
	require.NoError(testingT, database.First(&refreshed, "id = ?", feedback.ID).Error)
	require.Equal(testingT, model.FeedbackDeliveryTexted, refreshed.Delivery)
	require.Equal(testingT, model.FeedbackDeliveryTexted, feedback.Delivery)
}

func TestApplyFeedbackNotificationSkipsWhenDatabaseNil(testingT *testing.T) {
	feedback := model.Feedback{
		ID:       storage.NewID(),
		SiteID:   storage.NewID(),
		Contact:  "user@example.com",
		Message:  "Hello",
		Delivery: model.FeedbackDeliveryNone,
	}
	notifier := stubFeedbackNotifier{delivery: model.FeedbackDeliveryTexted}
	applyFeedbackNotification(context.Background(), nil, zap.NewNop(), notifier, model.Site{}, &feedback)
	require.Equal(testingT, model.FeedbackDeliveryNone, feedback.Delivery)
}

func TestApplyFeedbackNotificationHandlesNotifierError(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Error Site",
		AllowedOrigin: "https://example.com",
		OwnerEmail:    "owner@example.com",
	}
	require.NoError(testingT, database.Create(&site).Error)

	feedback := model.Feedback{
		ID:       storage.NewID(),
		SiteID:   site.ID,
		Contact:  "user@example.com",
		Message:  "Hello",
		Delivery: model.FeedbackDeliveryMailed,
	}
	require.NoError(testingT, database.Create(&feedback).Error)

	notifier := stubFeedbackNotifier{
		delivery:  model.FeedbackDeliveryTexted,
		notifyErr: errors.New("notify failed"),
	}
	applyFeedbackNotification(context.Background(), database, zap.NewNop(), notifier, site, &feedback)

	var refreshed model.Feedback
	require.NoError(testingT, database.First(&refreshed, "id = ?", feedback.ID).Error)
	require.Equal(testingT, model.FeedbackDeliveryNone, refreshed.Delivery)
	require.Equal(testingT, model.FeedbackDeliveryNone, feedback.Delivery)
}

func TestApplyFeedbackNotificationHandlesUpdateError(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Update Site",
		AllowedOrigin: "https://example.com",
		OwnerEmail:    "owner@example.com",
	}
	require.NoError(testingT, database.Create(&site).Error)

	feedback := model.Feedback{
		ID:       storage.NewID(),
		SiteID:   site.ID,
		Contact:  "user@example.com",
		Message:  "Hello",
		Delivery: model.FeedbackDeliveryMailed,
	}
	require.NoError(testingT, database.Create(&feedback).Error)

	sqlDatabase, sqlErr := database.DB()
	require.NoError(testingT, sqlErr)
	require.NoError(testingT, sqlDatabase.Close())

	notifier := stubFeedbackNotifier{delivery: model.FeedbackDeliveryTexted}
	applyFeedbackNotification(context.Background(), database, zap.NewNop(), notifier, site, &feedback)
	require.Equal(testingT, model.FeedbackDeliveryMailed, feedback.Delivery)
}
