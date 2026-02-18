package api

import (
	"context"
	"net/url"
	"strings"
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

func TestDatabaseSiteStatisticsProviderTopPagesMergesTrailingSlashVariants(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	siteID := storage.NewID()

	decisioningTrailingOne, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:   siteID,
		URL:      "https://example.com/decisioning/",
		Occurred: time.Date(2024, 1, 2, 1, 0, 0, 0, time.UTC),
	})
	require.NoError(testingT, visitErr)
	decisioningTrailingTwo, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:   siteID,
		URL:      "https://example.com/decisioning/",
		Occurred: time.Date(2024, 1, 2, 2, 0, 0, 0, time.UTC),
	})
	require.NoError(testingT, visitErr)
	decisioningPlain, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:   siteID,
		URL:      "https://example.com/decisioning",
		Occurred: time.Date(2024, 1, 2, 3, 0, 0, 0, time.UTC),
	})
	require.NoError(testingT, visitErr)
	civilizationTrailingOne, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:   siteID,
		URL:      "https://example.com/civilization/",
		Occurred: time.Date(2024, 1, 2, 4, 0, 0, 0, time.UTC),
	})
	require.NoError(testingT, visitErr)
	civilizationTrailingTwo, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:   siteID,
		URL:      "https://example.com/civilization/",
		Occurred: time.Date(2024, 1, 2, 5, 0, 0, 0, time.UTC),
	})
	require.NoError(testingT, visitErr)
	civilizationPlain, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:   siteID,
		URL:      "https://example.com/civilization",
		Occurred: time.Date(2024, 1, 2, 6, 0, 0, 0, time.UTC),
	})
	require.NoError(testingT, visitErr)

	require.NoError(testingT, database.Create(&decisioningTrailingOne).Error)
	require.NoError(testingT, database.Create(&decisioningTrailingTwo).Error)
	require.NoError(testingT, database.Create(&decisioningPlain).Error)
	require.NoError(testingT, database.Create(&civilizationTrailingOne).Error)
	require.NoError(testingT, database.Create(&civilizationTrailingTwo).Error)
	require.NoError(testingT, database.Create(&civilizationPlain).Error)

	provider := NewDatabaseSiteStatisticsProvider(database)
	results, err := provider.TopPages(context.Background(), siteID, 10)
	require.NoError(testingT, err)
	require.Len(testingT, results, 2)

	visitCountByPath := make(map[string]int64, len(results))
	for _, result := range results {
		visitCountByPath[result.Path] = result.VisitCount
	}
	require.Equal(testingT, int64(3), visitCountByPath["/decisioning"])
	require.Equal(testingT, int64(3), visitCountByPath["/civilization"])
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

func TestDatabaseSiteStatisticsProviderExcludesBotsFromDefaultStats(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	siteID := storage.NewID()

	humanVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/alpha",
		VisitorID: storage.NewID(),
		Occurred:  time.Now().UTC().Add(-time.Hour),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	botVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/alpha",
		VisitorID: storage.NewID(),
		Occurred:  time.Now().UTC(),
		IsBot:     true,
	})
	require.NoError(testingT, visitErr)
	require.NoError(testingT, database.Create(&humanVisit).Error)
	require.NoError(testingT, database.Create(&botVisit).Error)

	provider := NewDatabaseSiteStatisticsProvider(database)
	visitCount, visitErr := provider.VisitCount(context.Background(), siteID)
	require.NoError(testingT, visitErr)
	require.Equal(testingT, int64(1), visitCount)

	uniqueCount, uniqueErr := provider.UniqueVisitorCount(context.Background(), siteID)
	require.NoError(testingT, uniqueErr)
	require.Equal(testingT, int64(1), uniqueCount)

	topPages, topPagesErr := provider.TopPages(context.Background(), siteID, 10)
	require.NoError(testingT, topPagesErr)
	require.Len(testingT, topPages, 1)
	require.Equal(testingT, "/alpha", topPages[0].Path)
	require.Equal(testingT, int64(1), topPages[0].VisitCount)
}

func TestDatabaseSiteStatisticsProviderVisitTrendDefaultsToSevenDays(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	siteID := storage.NewID()
	startOfToday := time.Now().UTC().Truncate(24 * time.Hour)
	yesterday := startOfToday.AddDate(0, 0, -1)

	yesterdayVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/yesterday",
		VisitorID: storage.NewID(),
		Occurred:  yesterday.Add(4 * time.Hour),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	todayVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/today",
		VisitorID: storage.NewID(),
		Occurred:  startOfToday.Add(2 * time.Hour),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	todayBotVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/today",
		VisitorID: storage.NewID(),
		Occurred:  startOfToday.Add(3 * time.Hour),
		IsBot:     true,
	})
	require.NoError(testingT, visitErr)
	require.NoError(testingT, database.Create(&yesterdayVisit).Error)
	require.NoError(testingT, database.Create(&todayVisit).Error)
	require.NoError(testingT, database.Create(&todayBotVisit).Error)

	provider := NewDatabaseSiteStatisticsProvider(database)
	trend, trendErr := provider.VisitTrend(context.Background(), siteID, 0)
	require.NoError(testingT, trendErr)
	require.Len(testingT, trend, 7)

	expectedStartDay := startOfToday.AddDate(0, 0, -6)
	require.Equal(testingT, expectedStartDay, trend[0].Date)
	require.Equal(testingT, startOfToday, trend[len(trend)-1].Date)

	trendByDate := make(map[string]DailyVisitTrendStat, len(trend))
	for _, point := range trend {
		trendByDate[point.Date.Format("2006-01-02")] = point
	}

	yesterdayKey := yesterday.Format("2006-01-02")
	require.Equal(testingT, int64(1), trendByDate[yesterdayKey].PageViews)
	require.Equal(testingT, int64(1), trendByDate[yesterdayKey].UniqueVisitors)

	todayKey := startOfToday.Format("2006-01-02")
	require.Equal(testingT, int64(1), trendByDate[todayKey].PageViews)
	require.Equal(testingT, int64(1), trendByDate[todayKey].UniqueVisitors)
}

func TestDatabaseSiteStatisticsProviderVisitAttributionExcludesBots(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	siteID := storage.NewID()

	adVisitOne, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/pricing?utm_source=google&utm_medium=cpc&utm_campaign=spring",
		VisitorID: storage.NewID(),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	adVisitTwo, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/signup?utm_source=google&utm_medium=cpc&utm_campaign=spring",
		VisitorID: storage.NewID(),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	referralVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/blog",
		Referrer:  "https://news.ycombinator.com/item?id=123",
		VisitorID: storage.NewID(),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	directVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/contact",
		VisitorID: storage.NewID(),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	botVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/crawl?utm_source=bot&utm_medium=automation&utm_campaign=spider",
		VisitorID: storage.NewID(),
		IsBot:     true,
	})
	require.NoError(testingT, visitErr)

	require.NoError(testingT, database.Create(&adVisitOne).Error)
	require.NoError(testingT, database.Create(&adVisitTwo).Error)
	require.NoError(testingT, database.Create(&referralVisit).Error)
	require.NoError(testingT, database.Create(&directVisit).Error)
	require.NoError(testingT, database.Create(&botVisit).Error)

	provider := NewDatabaseSiteStatisticsProvider(database)
	breakdown, breakdownErr := provider.VisitAttribution(context.Background(), siteID, 10)
	require.NoError(testingT, breakdownErr)

	sourceByValue := make(map[string]AttributionStat, len(breakdown.Sources))
	for _, entry := range breakdown.Sources {
		sourceByValue[entry.Value] = entry
	}
	require.Equal(testingT, int64(2), sourceByValue["google"].VisitCount)
	require.Equal(testingT, int64(1), sourceByValue["news.ycombinator.com"].VisitCount)
	require.Equal(testingT, int64(1), sourceByValue["direct"].VisitCount)
	_, hasBotSource := sourceByValue["bot"]
	require.False(testingT, hasBotSource)

	mediumByValue := make(map[string]AttributionStat, len(breakdown.Mediums))
	for _, entry := range breakdown.Mediums {
		mediumByValue[entry.Value] = entry
	}
	require.Equal(testingT, int64(2), mediumByValue["cpc"].VisitCount)
	require.Equal(testingT, int64(1), mediumByValue["referral"].VisitCount)
	require.Equal(testingT, int64(1), mediumByValue["direct"].VisitCount)
	_, hasBotMedium := mediumByValue["automation"]
	require.False(testingT, hasBotMedium)

	campaignByValue := make(map[string]AttributionStat, len(breakdown.Campaigns))
	for _, entry := range breakdown.Campaigns {
		campaignByValue[entry.Value] = entry
	}
	require.Equal(testingT, int64(2), campaignByValue["spring"].VisitCount)
	require.Equal(testingT, int64(2), campaignByValue["none"].VisitCount)
	_, hasBotCampaign := campaignByValue["spider"]
	require.False(testingT, hasBotCampaign)
}

func TestDatabaseSiteStatisticsProviderVisitAttributionRespectsLimit(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	siteID := storage.NewID()

	googleVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/path?utm_source=google&utm_medium=cpc&utm_campaign=sale",
		VisitorID: storage.NewID(),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	directVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/path",
		VisitorID: storage.NewID(),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	referralVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/path",
		Referrer:  "https://www.github.com/repo",
		VisitorID: storage.NewID(),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)

	require.NoError(testingT, database.Create(&googleVisit).Error)
	require.NoError(testingT, database.Create(&directVisit).Error)
	require.NoError(testingT, database.Create(&referralVisit).Error)

	provider := NewDatabaseSiteStatisticsProvider(database)
	breakdown, breakdownErr := provider.VisitAttribution(context.Background(), siteID, 1)
	require.NoError(testingT, breakdownErr)
	require.Len(testingT, breakdown.Sources, 1)
	require.Len(testingT, breakdown.Mediums, 1)
	require.Len(testingT, breakdown.Campaigns, 1)
}

func TestDatabaseSiteStatisticsProviderVisitAggregationsSkipBlankSite(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	provider := NewDatabaseSiteStatisticsProvider(database)

	trend, trendErr := provider.VisitTrend(context.Background(), "   ", 7)
	require.NoError(testingT, trendErr)
	require.Nil(testingT, trend)

	attribution, attributionErr := provider.VisitAttribution(context.Background(), "   ", 10)
	require.NoError(testingT, attributionErr)
	require.Empty(testingT, attribution.Sources)
	require.Empty(testingT, attribution.Mediums)
	require.Empty(testingT, attribution.Campaigns)

	engagement, engagementErr := provider.VisitEngagement(context.Background(), "   ", 30)
	require.NoError(testingT, engagementErr)
	require.Equal(testingT, VisitEngagementStat{}, engagement)
}

func TestDatabaseSiteStatisticsProviderVisitEngagementDefaultsAndExcludesBots(testingT *testing.T) {
	database := openFaviconManagerDatabase(testingT)
	siteID := storage.NewID()
	startOfToday := time.Now().UTC().Truncate(24 * time.Hour)

	visitorOneVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/one",
		VisitorID: "11111111-1111-1111-1111-111111111111",
		Occurred:  startOfToday.Add(2 * time.Hour),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	visitorTwoVisitOne, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/two-a",
		VisitorID: "22222222-2222-2222-2222-222222222222",
		Occurred:  startOfToday.Add(3 * time.Hour),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	visitorTwoVisitTwo, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/two-b",
		VisitorID: "22222222-2222-2222-2222-222222222222",
		Occurred:  startOfToday.Add(3*time.Hour + 10*time.Second),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	visitorThreeVisitOne, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/three-a",
		VisitorID: "33333333-3333-3333-3333-333333333333",
		Occurred:  startOfToday.Add(4 * time.Hour),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	visitorThreeVisitTwo, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/three-b",
		VisitorID: "33333333-3333-3333-3333-333333333333",
		Occurred:  startOfToday.Add(4*time.Hour + 2*time.Minute),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	visitorThreeVisitThree, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/three-c",
		VisitorID: "33333333-3333-3333-3333-333333333333",
		Occurred:  startOfToday.Add(4*time.Hour + 4*time.Minute),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	visitorFourVisitOne, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/four-a",
		VisitorID: "44444444-4444-4444-4444-444444444444",
		Occurred:  startOfToday.Add(5 * time.Hour),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	visitorFourVisitTwo, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/four-b",
		VisitorID: "44444444-4444-4444-4444-444444444444",
		Occurred:  startOfToday.Add(5*time.Hour + 12*time.Minute),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	visitorFourVisitThree, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/four-c",
		VisitorID: "44444444-4444-4444-4444-444444444444",
		Occurred:  startOfToday.Add(5*time.Hour + 24*time.Minute),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	visitorFourVisitFour, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/four-d",
		VisitorID: "44444444-4444-4444-4444-444444444444",
		Occurred:  startOfToday.Add(5*time.Hour + 36*time.Minute),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	visitorFourVisitFive, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/four-e",
		VisitorID: "44444444-4444-4444-4444-444444444444",
		Occurred:  startOfToday.Add(5*time.Hour + 48*time.Minute),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	visitorFourVisitSix, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/four-f",
		VisitorID: "44444444-4444-4444-4444-444444444444",
		Occurred:  startOfToday.Add(6 * time.Hour),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	visitorFourVisitSeven, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/four-g",
		VisitorID: "44444444-4444-4444-4444-444444444444",
		Occurred:  startOfToday.Add(6*time.Hour + 12*time.Minute),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	visitorFourVisitEight, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/four-h",
		VisitorID: "44444444-4444-4444-4444-444444444444",
		Occurred:  startOfToday.Add(6*time.Hour + 24*time.Minute),
		IsBot:     false,
	})
	require.NoError(testingT, visitErr)
	botVisit, visitErr := model.NewSiteVisit(model.SiteVisitInput{
		SiteID:    siteID,
		URL:       "https://example.com/bot",
		VisitorID: "55555555-5555-5555-5555-555555555555",
		Occurred:  startOfToday.Add(7 * time.Hour),
		IsBot:     true,
	})
	require.NoError(testingT, visitErr)

	require.NoError(testingT, database.Create(&visitorOneVisit).Error)
	require.NoError(testingT, database.Create(&visitorTwoVisitOne).Error)
	require.NoError(testingT, database.Create(&visitorTwoVisitTwo).Error)
	require.NoError(testingT, database.Create(&visitorThreeVisitOne).Error)
	require.NoError(testingT, database.Create(&visitorThreeVisitTwo).Error)
	require.NoError(testingT, database.Create(&visitorThreeVisitThree).Error)
	require.NoError(testingT, database.Create(&visitorFourVisitOne).Error)
	require.NoError(testingT, database.Create(&visitorFourVisitTwo).Error)
	require.NoError(testingT, database.Create(&visitorFourVisitThree).Error)
	require.NoError(testingT, database.Create(&visitorFourVisitFour).Error)
	require.NoError(testingT, database.Create(&visitorFourVisitFive).Error)
	require.NoError(testingT, database.Create(&visitorFourVisitSix).Error)
	require.NoError(testingT, database.Create(&visitorFourVisitSeven).Error)
	require.NoError(testingT, database.Create(&visitorFourVisitEight).Error)
	require.NoError(testingT, database.Create(&botVisit).Error)

	provider := NewDatabaseSiteStatisticsProvider(database)
	engagement, engagementErr := provider.VisitEngagement(context.Background(), siteID, 0)
	require.NoError(testingT, engagementErr)

	require.Equal(testingT, int64(4), engagement.TrackedVisitorCount)
	require.Equal(testingT, int64(3), engagement.ReturningVisitorCount)
	require.Equal(testingT, 0.75, engagement.ReturningVisitorRate)
	require.Equal(testingT, 3.5, engagement.AveragePagesPerVisitor)

	require.Equal(testingT, int64(1), engagement.DepthDistribution.SinglePage)
	require.Equal(testingT, int64(2), engagement.DepthDistribution.TwoToThree)
	require.Equal(testingT, int64(0), engagement.DepthDistribution.FourToSeven)
	require.Equal(testingT, int64(1), engagement.DepthDistribution.EightOrMore)

	require.Equal(testingT, int64(2), engagement.ObservedTimeDistribution.UnderThirtySeconds)
	require.Equal(testingT, int64(0), engagement.ObservedTimeDistribution.ThirtyToOneNineteen)
	require.Equal(testingT, int64(1), engagement.ObservedTimeDistribution.OneTwentyToFiveNinetyNine)
	require.Equal(testingT, int64(1), engagement.ObservedTimeDistribution.SixHundredOrMore)
}

func TestNormalizeVisitTrendMapKeyNormalizesTimestampLikeDayStrings(testingT *testing.T) {
	dayKey, dateValue, normalizeErr := normalizeVisitTrendMapKey("2025-01-03T12:13:14Z")
	require.NoError(testingT, normalizeErr)
	require.Equal(testingT, "2025-01-03", dayKey)
	require.Equal(testingT, time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), dateValue)
	require.NotEqual(testingT, "2025-01-03T12:13:14Z", dayKey)

	_, _, emptyDayErr := normalizeVisitTrendMapKey("  ")
	require.Error(testingT, emptyDayErr)
	require.ErrorContains(testingT, emptyDayErr, "visit_trend_parse_day: empty day value")
}

func TestVisitTrendHelperNormalizersAndParsing(testingT *testing.T) {
	require.Equal(testingT, defaultVisitTrendDays, normalizeVisitTrendDays(0))
	require.Equal(testingT, maxVisitTrendDays, normalizeVisitTrendDays(maxVisitTrendDays+10))
	require.Equal(testingT, 14, normalizeVisitTrendDays(14))

	dayOnlyValue, dayOnlyErr := parseVisitTrendDate("2025-01-02")
	require.NoError(testingT, dayOnlyErr)
	require.Equal(testingT, time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), dayOnlyValue)

	rfc3339Value, rfc3339Err := parseVisitTrendDate("2025-01-03T12:13:14Z")
	require.NoError(testingT, rfc3339Err)
	require.Equal(testingT, time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), rfc3339Value)

	dateTimeValue, dateTimeErr := parseVisitTrendDate("2025-01-04 08:09:10+00:00")
	require.NoError(testingT, dateTimeErr)
	require.Equal(testingT, time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC), dateTimeValue)

	_, unsupportedErr := parseVisitTrendDate("not-a-date")
	require.Error(testingT, unsupportedErr)
	require.ErrorContains(testingT, unsupportedErr, "unsupported visit trend date format")
}

func TestVisitAttributionHelperFunctions(testingT *testing.T) {
	require.Equal(testingT, defaultVisitAttributionLimit, normalizeVisitAttributionLimit(0))
	require.Equal(testingT, maxVisitAttributionLimit, normalizeVisitAttributionLimit(maxVisitAttributionLimit+10))
	require.Equal(testingT, 7, normalizeVisitAttributionLimit(7))

	visitURL, parseErr := url.Parse("https://example.com/path?utm_source=Google&utm_medium=CPC&utm_campaign=Spring")
	require.NoError(testingT, parseErr)
	require.Equal(testingT, "google", readUTMValue(visitURL, attributionUTMSourceKey))
	require.Equal(testingT, "", readUTMValue(nil, attributionUTMMediumKey))

	require.Equal(testingT, "example.com", normalizeReferrerHost("https://www.Example.com/path"))
	require.Equal(testingT, "", normalizeReferrerHost("https://"))
	require.Equal(testingT, "", normalizeReferrerHost("http://%zz"))

	require.Equal(testingT, "", normalizeAttributionValue(" \t "))
	trimmedValue := normalizeAttributionValue(strings.Repeat("A", attributionValueMaxLength+8))
	require.Equal(testingT, attributionValueMaxLength, len(trimmedValue))

	directSource, directMedium, directCampaign := resolveVisitAttribution("http://%zz", "http://%zz")
	require.Equal(testingT, attributionDefaultSource, directSource)
	require.Equal(testingT, attributionDefaultMedium, directMedium)
	require.Equal(testingT, attributionDefaultCampaign, directCampaign)

	referralSource, referralMedium, referralCampaign := resolveVisitAttribution("http://%zz", "https://www.GitHub.com/repo")
	require.Equal(testingT, "github.com", referralSource)
	require.Equal(testingT, attributionReferralMedium, referralMedium)
	require.Equal(testingT, attributionDefaultCampaign, referralCampaign)

	utmSource, utmMedium, utmCampaign := resolveVisitAttribution(
		"https://example.com/path?utm_source=Google&utm_medium=CPC&utm_campaign=Spring",
		"https://news.ycombinator.com/item?id=1",
	)
	require.Equal(testingT, "google", utmSource)
	require.Equal(testingT, "cpc", utmMedium)
	require.Equal(testingT, "spring", utmCampaign)

	require.Nil(testingT, topAttributionStats(nil, 3))
	rankedStats := topAttributionStats(map[string]int64{
		"beta":  2,
		"gamma": 3,
		"alpha": 2,
	}, 2)
	require.Len(testingT, rankedStats, 2)
	require.Equal(testingT, "gamma", rankedStats[0].Value)
	require.Equal(testingT, int64(3), rankedStats[0].VisitCount)
	require.Equal(testingT, "alpha", rankedStats[1].Value)
	require.Equal(testingT, int64(2), rankedStats[1].VisitCount)
}

func TestVisitEngagementHelperFunctions(testingT *testing.T) {
	require.Equal(testingT, defaultVisitEngagementDays, normalizeVisitEngagementDays(0))
	require.Equal(testingT, maxVisitEngagementDays, normalizeVisitEngagementDays(maxVisitEngagementDays+1))
	require.Equal(testingT, 45, normalizeVisitEngagementDays(45))

	depthDistribution := VisitDepthDistributionStat{}
	depthDistribution = accumulateDepthDistribution(depthDistribution, 1)
	depthDistribution = accumulateDepthDistribution(depthDistribution, 2)
	depthDistribution = accumulateDepthDistribution(depthDistribution, 4)
	depthDistribution = accumulateDepthDistribution(depthDistribution, 8)
	require.Equal(testingT, int64(1), depthDistribution.SinglePage)
	require.Equal(testingT, int64(1), depthDistribution.TwoToThree)
	require.Equal(testingT, int64(1), depthDistribution.FourToSeven)
	require.Equal(testingT, int64(1), depthDistribution.EightOrMore)

	timeDistribution := VisitObservedTimeDistributionStat{}
	timeDistribution = accumulateObservedTimeDistribution(timeDistribution, 29)
	timeDistribution = accumulateObservedTimeDistribution(timeDistribution, 30)
	timeDistribution = accumulateObservedTimeDistribution(timeDistribution, 120)
	timeDistribution = accumulateObservedTimeDistribution(timeDistribution, 600)
	require.Equal(testingT, int64(1), timeDistribution.UnderThirtySeconds)
	require.Equal(testingT, int64(1), timeDistribution.ThirtyToOneNineteen)
	require.Equal(testingT, int64(1), timeDistribution.OneTwentyToFiveNinetyNine)
	require.Equal(testingT, int64(1), timeDistribution.SixHundredOrMore)

	firstSeen := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	lastSeen := firstSeen.Add(95 * time.Second)
	require.Equal(testingT, int64(95), observedVisitDurationSeconds(firstSeen, lastSeen))
	require.Equal(testingT, int64(0), observedVisitDurationSeconds(time.Time{}, lastSeen))
	require.Equal(testingT, int64(0), observedVisitDurationSeconds(lastSeen, firstSeen))

	require.Equal(testingT, 0.33, roundVisitEngagementMetric(1.0/3.0))
}
