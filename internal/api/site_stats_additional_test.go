package api

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
)

func TestDatabaseSiteStatisticsProviderTopPagesDefaultsLimit(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	siteID := storage.NewID()

	visitOne, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:   siteID,
		URL:      "https://example.com/alpha",
		Occurred: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(testingT, visitErr)
	visitTwo, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:   siteID,
		URL:      "https://example.com/alpha",
		Occurred: time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC),
	})
	require.NoError(testingT, visitErr)
	visitThree, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:   siteID,
		URL:      "https://example.com/beta",
		Occurred: time.Date(2024, 1, 1, 2, 0, 0, 0, time.UTC),
	})
	require.NoError(testingT, visitErr)
	require.NoError(testingT, database.Create(&visitOne).Error)
	require.NoError(testingT, database.Create(&visitTwo).Error)
	require.NoError(testingT, database.Create(&visitThree).Error)

	provider := NewDatabaseSiteStatisticsProvider(database)
	results, err := provider.TopPages(context.Background(), siteID, 0)
	require.NoError(testingT, err)
	require.NotEmpty(testingT, results)
	require.Equal(testingT, "/alpha", results[0].Path)
	require.Equal(testingT, int64(2), results[0].VisitCount)
}

func TestDatabaseSiteStatisticsProviderTopPagesSkipsBlankSite(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	provider := NewDatabaseSiteStatisticsProvider(database)
	results, err := provider.TopPages(context.Background(), "   ", 1)
	require.NoError(testingT, err)
	require.Nil(testingT, results)
}

func TestSiteHandlersRecentVisitsDefaultsLimit(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	siteID := storage.NewID()
	firstVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:   siteID,
		URL:      "https://example.com/first",
		Occurred: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(testingT, visitErr)
	secondVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:   siteID,
		URL:      "https://example.com/second",
		Occurred: time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC),
	})
	require.NoError(testingT, visitErr)
	require.NoError(testingT, database.Create(&firstVisit).Error)
	require.NoError(testingT, database.Create(&secondVisit).Error)

	handlers := &SiteHandlers{database: database}
	entries, err := handlers.recentVisits(context.Background(), siteID, 0)
	require.NoError(testingT, err)
	require.NotEmpty(testingT, entries)
	require.Equal(testingT, secondVisit.URL, entries[0].URL)
}

func TestSiteHandlersRecentVisitsSkipsBlankSite(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	handlers := &SiteHandlers{database: database}
	entries, err := handlers.recentVisits(context.Background(), "   ", 1)
	require.NoError(testingT, err)
	require.Nil(testingT, entries)
}

func TestDatabaseSiteStatisticsProviderCountsSkipBlankSite(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	provider := NewDatabaseSiteStatisticsProvider(database)

	feedbackCount, feedbackErr := provider.FeedbackCount(context.Background(), " ")
	require.NoError(testingT, feedbackErr)
	require.Equal(testingT, int64(0), feedbackCount)

	subscriberCount, subscriberErr := provider.SubscriberCount(context.Background(), "\t")
	require.NoError(testingT, subscriberErr)
	require.Equal(testingT, int64(0), subscriberCount)

	visitCount, visitErr := provider.VisitCount(context.Background(), "\n")
	require.NoError(testingT, visitErr)
	require.Equal(testingT, int64(0), visitCount)

	uniqueCount, uniqueErr := provider.UniqueVisitorCount(context.Background(), " ")
	require.NoError(testingT, uniqueErr)
	require.Equal(testingT, int64(0), uniqueCount)
}
