package api

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"gorm.io/gorm"
)

const (
	defaultVisitTrendDays = 7
	maxVisitTrendDays     = 30
	visitTrendDayLayout   = "2006-01-02"
	visitTrendDateTimeUTC = "2006-01-02 15:04:05-07:00"
	visitTrendDateTimeTZ  = "2006-01-02 15:04:05-07"
	visitTrendDateTime    = "2006-01-02 15:04:05"

	topPagesCanonicalPathExpression = "CASE WHEN TRIM(path, '/') = '' THEN '/' ELSE RTRIM(path, '/') END"
	topPagesSelectStatement         = topPagesCanonicalPathExpression + " as path, COUNT(*) as visit_count"

	defaultVisitAttributionLimit = 10
	maxVisitAttributionLimit     = 50
	attributionUTMSourceKey      = "utm_source"
	attributionUTMMediumKey      = "utm_medium"
	attributionUTMCampaignKey    = "utm_campaign"
	attributionDefaultSource     = "direct"
	attributionDefaultMedium     = "direct"
	attributionReferralMedium    = "referral"
	attributionDefaultCampaign   = "none"
	attributionWWWPrefix         = "www."
	attributionValueMaxLength    = 120

	defaultVisitEngagementDays       = 30
	maxVisitEngagementDays           = 90
	visitDepthSinglePageMax          = 1
	visitDepthTwoToThreePagesMax     = 3
	visitDepthFourToSevenPagesMax    = 7
	visitDurationUnderThirtySeconds  = 30
	visitDurationUnderTwoMinutes     = 120
	visitDurationUnderTenMinutes     = 600
	visitEngagementMetricRoundFactor = 100
)

var visitTrendParseLayouts = [...]string{
	visitTrendDayLayout,
	time.RFC3339,
	visitTrendDateTimeUTC,
	visitTrendDateTimeTZ,
	visitTrendDateTime,
}

// SiteStatisticsProvider exposes site metadata such as feedback counts.
type SiteStatisticsProvider interface {
	FeedbackCount(ctx context.Context, siteID string) (int64, error)
	SubscriberCount(ctx context.Context, siteID string) (int64, error)
	VisitCount(ctx context.Context, siteID string) (int64, error)
	UniqueVisitorCount(ctx context.Context, siteID string) (int64, error)
	TopPages(ctx context.Context, siteID string, limit int) ([]TopPageStat, error)
	VisitTrend(ctx context.Context, siteID string, days int) ([]DailyVisitTrendStat, error)
	VisitAttribution(ctx context.Context, siteID string, limit int) (VisitAttributionBreakdown, error)
	VisitEngagement(ctx context.Context, siteID string, days int) (VisitEngagementStat, error)
}

// DatabaseSiteStatisticsProvider implements SiteStatisticsProvider using GORM.
type DatabaseSiteStatisticsProvider struct {
	database *gorm.DB
}

// NewDatabaseSiteStatisticsProvider builds a statistics provider backed by the primary database.
func NewDatabaseSiteStatisticsProvider(database *gorm.DB) *DatabaseSiteStatisticsProvider {
	return &DatabaseSiteStatisticsProvider{database: database}
}

// FeedbackCount returns the number of feedback messages for a site.
func (provider *DatabaseSiteStatisticsProvider) FeedbackCount(ctx context.Context, siteID string) (int64, error) {
	if strings.TrimSpace(siteID) == "" {
		return 0, nil
	}
	var count int64
	err := provider.database.WithContext(ctx).Model(&model.Feedback{}).Where("site_id = ?", siteID).Count(&count).Error
	return count, err
}

// SubscriberCount returns the number of subscribers for a site.
func (provider *DatabaseSiteStatisticsProvider) SubscriberCount(ctx context.Context, siteID string) (int64, error) {
	if strings.TrimSpace(siteID) == "" {
		return 0, nil
	}
	var count int64
	err := provider.database.WithContext(ctx).Model(&model.Subscriber{}).Where("site_id = ?", siteID).Count(&count).Error
	return count, err
}

// VisitCount returns total page views for a site.
func (provider *DatabaseSiteStatisticsProvider) VisitCount(ctx context.Context, siteID string) (int64, error) {
	if strings.TrimSpace(siteID) == "" {
		return 0, nil
	}
	var count int64
	err := provider.database.WithContext(ctx).Model(&model.SiteVisit{}).Where("site_id = ? AND is_bot = ?", siteID, false).Count(&count).Error
	return count, err
}

// UniqueVisitorCount returns distinct visitor ids for a site.
func (provider *DatabaseSiteStatisticsProvider) UniqueVisitorCount(ctx context.Context, siteID string) (int64, error) {
	if strings.TrimSpace(siteID) == "" {
		return 0, nil
	}
	var count int64
	err := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Where("site_id = ? AND visitor_id <> '' AND is_bot = ?", siteID, false).
		Distinct("visitor_id").
		Count(&count).Error
	return count, err
}

// TopPageStat captures per-page view counts.
type TopPageStat struct {
	Path       string
	VisitCount int64
}

// TopPages returns top pages by visit count.
func (provider *DatabaseSiteStatisticsProvider) TopPages(ctx context.Context, siteID string, limit int) ([]TopPageStat, error) {
	if strings.TrimSpace(siteID) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}
	var results []TopPageStat
	err := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select(topPagesSelectStatement).
		Where("site_id = ? AND path <> '' AND is_bot = ?", siteID, false).
		Group(topPagesCanonicalPathExpression).
		Order("visit_count desc, path asc").
		Limit(limit).
		Scan(&results).Error
	return results, err
}

type dailyVisitTrendRow struct {
	Day            string
	PageViews      int64
	UniqueVisitors int64
}

type DailyVisitTrendStat struct {
	Date           time.Time
	PageViews      int64
	UniqueVisitors int64
}

type AttributionStat struct {
	Value      string
	VisitCount int64
}

type VisitAttributionBreakdown struct {
	Sources   []AttributionStat
	Mediums   []AttributionStat
	Campaigns []AttributionStat
}

type VisitDepthDistributionStat struct {
	SinglePage  int64
	TwoToThree  int64
	FourToSeven int64
	EightOrMore int64
}

type VisitObservedTimeDistributionStat struct {
	UnderThirtySeconds        int64
	ThirtyToOneNineteen       int64
	OneTwentyToFiveNinetyNine int64
	SixHundredOrMore          int64
}

type VisitEngagementStat struct {
	TrackedVisitorCount      int64
	ReturningVisitorCount    int64
	ReturningVisitorRate     float64
	AveragePagesPerVisitor   float64
	DepthDistribution        VisitDepthDistributionStat
	ObservedTimeDistribution VisitObservedTimeDistributionStat
}

func (provider *DatabaseSiteStatisticsProvider) VisitTrend(ctx context.Context, siteID string, days int) ([]DailyVisitTrendStat, error) {
	if strings.TrimSpace(siteID) == "" {
		return nil, nil
	}
	normalizedDays := normalizeVisitTrendDays(days)
	startDay := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -(normalizedDays - 1))

	var rows []dailyVisitTrendRow
	err := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select("DATE(occurred_at) as day, COUNT(*) as page_views, COUNT(DISTINCT CASE WHEN visitor_id <> '' THEN visitor_id END) as unique_visitors").
		Where("site_id = ? AND occurred_at >= ? AND is_bot = ?", siteID, startDay, false).
		Group("DATE(occurred_at)").
		Order("day asc").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	entriesByDay := make(map[string]DailyVisitTrendStat, len(rows))
	for _, row := range rows {
		dayKey, dateValue, normalizeErr := normalizeVisitTrendMapKey(row.Day)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		entriesByDay[dayKey] = DailyVisitTrendStat{
			Date:           dateValue,
			PageViews:      row.PageViews,
			UniqueVisitors: row.UniqueVisitors,
		}
	}

	trend := make([]DailyVisitTrendStat, 0, normalizedDays)
	for dayIndex := 0; dayIndex < normalizedDays; dayIndex++ {
		dateValue := startDay.AddDate(0, 0, dayIndex)
		dayKey := dateValue.Format(visitTrendDayLayout)
		if existingEntry, ok := entriesByDay[dayKey]; ok {
			trend = append(trend, existingEntry)
			continue
		}
		trend = append(trend, DailyVisitTrendStat{
			Date:           dateValue,
			PageViews:      0,
			UniqueVisitors: 0,
		})
	}
	return trend, nil
}

func normalizeVisitTrendDays(days int) int {
	if days <= 0 {
		return defaultVisitTrendDays
	}
	if days > maxVisitTrendDays {
		return maxVisitTrendDays
	}
	return days
}

func normalizeVisitTrendMapKey(rawDayValue string) (string, time.Time, error) {
	normalizedDay := strings.TrimSpace(rawDayValue)
	if normalizedDay == "" {
		return "", time.Time{}, fmt.Errorf("visit_trend_parse_day: empty day value")
	}

	dateValue, parseErr := parseVisitTrendDate(normalizedDay)
	if parseErr != nil {
		return "", time.Time{}, fmt.Errorf("visit_trend_parse_day %q: %w", normalizedDay, parseErr)
	}

	return dateValue.Format(visitTrendDayLayout), dateValue, nil
}

func parseVisitTrendDate(rawValue string) (time.Time, error) {
	for _, layout := range visitTrendParseLayouts {
		parsedValue, parseErr := time.ParseInLocation(layout, rawValue, time.UTC)
		if parseErr == nil {
			return parsedValue.UTC().Truncate(24 * time.Hour), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported visit trend date format: %s", rawValue)
}

type visitAttributionRow struct {
	URL      string
	Referrer string
}

func (provider *DatabaseSiteStatisticsProvider) VisitAttribution(ctx context.Context, siteID string, limit int) (VisitAttributionBreakdown, error) {
	if strings.TrimSpace(siteID) == "" {
		return VisitAttributionBreakdown{}, nil
	}

	normalizedLimit := normalizeVisitAttributionLimit(limit)
	var rows []visitAttributionRow
	err := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select("url, referrer").
		Where("site_id = ? AND is_bot = ?", siteID, false).
		Scan(&rows).Error
	if err != nil {
		return VisitAttributionBreakdown{}, err
	}

	sourceCounts := make(map[string]int64)
	mediumCounts := make(map[string]int64)
	campaignCounts := make(map[string]int64)
	for _, row := range rows {
		sourceValue, mediumValue, campaignValue := resolveVisitAttribution(row.URL, row.Referrer)
		sourceCounts[sourceValue]++
		mediumCounts[mediumValue]++
		campaignCounts[campaignValue]++
	}

	return VisitAttributionBreakdown{
		Sources:   topAttributionStats(sourceCounts, normalizedLimit),
		Mediums:   topAttributionStats(mediumCounts, normalizedLimit),
		Campaigns: topAttributionStats(campaignCounts, normalizedLimit),
	}, nil
}

func normalizeVisitAttributionLimit(limit int) int {
	if limit <= 0 {
		return defaultVisitAttributionLimit
	}
	if limit > maxVisitAttributionLimit {
		return maxVisitAttributionLimit
	}
	return limit
}

func resolveVisitAttribution(rawVisitURL string, rawReferrer string) (string, string, string) {
	parsedVisitURL, parsedVisitURLErr := url.Parse(strings.TrimSpace(rawVisitURL))
	if parsedVisitURLErr != nil {
		parsedVisitURL = nil
	}
	referrerHost := normalizeReferrerHost(rawReferrer)

	sourceValue := readUTMValue(parsedVisitURL, attributionUTMSourceKey)
	if sourceValue == "" {
		if referrerHost != "" {
			sourceValue = referrerHost
		} else {
			sourceValue = attributionDefaultSource
		}
	}

	mediumValue := readUTMValue(parsedVisitURL, attributionUTMMediumKey)
	if mediumValue == "" {
		if referrerHost != "" {
			mediumValue = attributionReferralMedium
		} else {
			mediumValue = attributionDefaultMedium
		}
	}

	campaignValue := readUTMValue(parsedVisitURL, attributionUTMCampaignKey)
	if campaignValue == "" {
		campaignValue = attributionDefaultCampaign
	}

	return sourceValue, mediumValue, campaignValue
}

func readUTMValue(parsedVisitURL *url.URL, key string) string {
	if parsedVisitURL == nil {
		return ""
	}
	return normalizeAttributionValue(parsedVisitURL.Query().Get(key))
}

func normalizeReferrerHost(rawReferrer string) string {
	parsedReferrer, parsedReferrerErr := url.Parse(strings.TrimSpace(rawReferrer))
	if parsedReferrerErr != nil {
		return ""
	}
	normalizedHost := strings.ToLower(strings.TrimSpace(parsedReferrer.Hostname()))
	if normalizedHost == "" {
		return ""
	}
	normalizedHost = strings.TrimPrefix(normalizedHost, attributionWWWPrefix)
	return normalizeAttributionValue(normalizedHost)
}

func normalizeAttributionValue(rawValue string) string {
	normalizedValue := strings.ToLower(strings.TrimSpace(rawValue))
	if normalizedValue == "" {
		return ""
	}
	if len(normalizedValue) > attributionValueMaxLength {
		return normalizedValue[:attributionValueMaxLength]
	}
	return normalizedValue
}

func topAttributionStats(counts map[string]int64, limit int) []AttributionStat {
	if len(counts) == 0 {
		return nil
	}
	entries := make([]AttributionStat, 0, len(counts))
	for value, visitCount := range counts {
		entries = append(entries, AttributionStat{
			Value:      value,
			VisitCount: visitCount,
		})
	}
	sort.Slice(entries, func(leftIndex int, rightIndex int) bool {
		if entries[leftIndex].VisitCount == entries[rightIndex].VisitCount {
			return entries[leftIndex].Value < entries[rightIndex].Value
		}
		return entries[leftIndex].VisitCount > entries[rightIndex].VisitCount
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries
}

type visitEngagementRow struct {
	VisitorID  string
	OccurredAt time.Time
}

type visitorEngagementAggregate struct {
	VisitCount int64
	FirstSeen  time.Time
	LastSeen   time.Time
}

func (provider *DatabaseSiteStatisticsProvider) VisitEngagement(ctx context.Context, siteID string, days int) (VisitEngagementStat, error) {
	if strings.TrimSpace(siteID) == "" {
		return VisitEngagementStat{}, nil
	}

	normalizedDays := normalizeVisitEngagementDays(days)
	startDay := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -(normalizedDays - 1))
	var rows []visitEngagementRow
	err := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select("visitor_id, occurred_at").
		Where("site_id = ? AND is_bot = ? AND visitor_id <> '' AND occurred_at >= ?", siteID, false, startDay).
		Scan(&rows).Error
	if err != nil {
		return VisitEngagementStat{}, err
	}

	aggregateByVisitor := make(map[string]visitorEngagementAggregate)
	for _, row := range rows {
		visitorIdentifier := strings.TrimSpace(row.VisitorID)
		if visitorIdentifier == "" {
			continue
		}
		occurredAtValue := row.OccurredAt.UTC()
		existingAggregate, found := aggregateByVisitor[visitorIdentifier]
		if !found {
			aggregateByVisitor[visitorIdentifier] = visitorEngagementAggregate{
				VisitCount: 1,
				FirstSeen:  occurredAtValue,
				LastSeen:   occurredAtValue,
			}
			continue
		}
		existingAggregate.VisitCount++
		if occurredAtValue.Before(existingAggregate.FirstSeen) {
			existingAggregate.FirstSeen = occurredAtValue
		}
		if occurredAtValue.After(existingAggregate.LastSeen) {
			existingAggregate.LastSeen = occurredAtValue
		}
		aggregateByVisitor[visitorIdentifier] = existingAggregate
	}

	metrics := VisitEngagementStat{
		TrackedVisitorCount: int64(len(aggregateByVisitor)),
	}
	var totalTrackedVisits int64
	for _, aggregate := range aggregateByVisitor {
		totalTrackedVisits += aggregate.VisitCount
		if aggregate.VisitCount > visitDepthSinglePageMax {
			metrics.ReturningVisitorCount++
		}

		metrics.DepthDistribution = accumulateDepthDistribution(metrics.DepthDistribution, aggregate.VisitCount)
		observedDurationSeconds := observedVisitDurationSeconds(aggregate.FirstSeen, aggregate.LastSeen)
		metrics.ObservedTimeDistribution = accumulateObservedTimeDistribution(metrics.ObservedTimeDistribution, observedDurationSeconds)
	}

	if metrics.TrackedVisitorCount > 0 {
		trackedVisitors := float64(metrics.TrackedVisitorCount)
		metrics.ReturningVisitorRate = roundVisitEngagementMetric(float64(metrics.ReturningVisitorCount) / trackedVisitors)
		metrics.AveragePagesPerVisitor = roundVisitEngagementMetric(float64(totalTrackedVisits) / trackedVisitors)
	}

	return metrics, nil
}

func normalizeVisitEngagementDays(days int) int {
	if days <= 0 {
		return defaultVisitEngagementDays
	}
	if days > maxVisitEngagementDays {
		return maxVisitEngagementDays
	}
	return days
}

func accumulateDepthDistribution(distribution VisitDepthDistributionStat, visitCount int64) VisitDepthDistributionStat {
	switch {
	case visitCount <= visitDepthSinglePageMax:
		distribution.SinglePage++
	case visitCount <= visitDepthTwoToThreePagesMax:
		distribution.TwoToThree++
	case visitCount <= visitDepthFourToSevenPagesMax:
		distribution.FourToSeven++
	default:
		distribution.EightOrMore++
	}
	return distribution
}

func accumulateObservedTimeDistribution(distribution VisitObservedTimeDistributionStat, observedDurationSeconds int64) VisitObservedTimeDistributionStat {
	switch {
	case observedDurationSeconds < visitDurationUnderThirtySeconds:
		distribution.UnderThirtySeconds++
	case observedDurationSeconds < visitDurationUnderTwoMinutes:
		distribution.ThirtyToOneNineteen++
	case observedDurationSeconds < visitDurationUnderTenMinutes:
		distribution.OneTwentyToFiveNinetyNine++
	default:
		distribution.SixHundredOrMore++
	}
	return distribution
}

func observedVisitDurationSeconds(firstSeen time.Time, lastSeen time.Time) int64 {
	if firstSeen.IsZero() || lastSeen.IsZero() {
		return 0
	}
	duration := lastSeen.Sub(firstSeen)
	if duration < 0 {
		return 0
	}
	return int64(duration.Seconds())
}

func roundVisitEngagementMetric(value float64) float64 {
	return math.Round(value*visitEngagementMetricRoundFactor) / visitEngagementMetricRoundFactor
}
