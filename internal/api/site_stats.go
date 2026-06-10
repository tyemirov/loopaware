package api

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/url"
	"sort"
	"strconv"
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

	defaultDeviceBreakdownLimit = 10
	maxDeviceBreakdownLimit     = 50
	deviceTypeMobile            = "mobile"
	deviceTypeTablet            = "tablet"
	deviceTypeDesktop           = "desktop"

	defaultLocationDistributionLimit = 10
	maxLocationDistributionLimit     = 50
	locationSourceEdgeGeo            = "edge_geo"
	locationSourceTimezone           = "timezone"
	locationSourceLocale             = "locale"
	locationSourceNetwork            = "network"
	locationSourceUnknown            = "unknown"
	locationUnknownLabel             = "Unknown location"
	locationUnknownSignal            = "missing_signal"
	locationTimezoneUnknownSignal    = "timezone_unknown"
	locationNetworkLocalSignal       = "local_network"
	locationConfidenceEdgeExact      = 95
	locationConfidenceEdgeCoordinate = 88
	locationConfidenceEdgeCountry    = 80
	locationConfidenceTimezone       = 45
	locationConfidenceTimezoneAgree  = 60
	locationConfidenceLocale         = 35
	locationConfidenceNetwork        = 20
	locationConfidenceUnknown        = 0
)

type locationAnchor struct {
	Label     string
	Latitude  float64
	Longitude float64
}

var locationTimezoneAnchors = map[string]locationAnchor{
	"UTC":                            {Label: "UTC", Latitude: 0, Longitude: 0},
	"Etc/UTC":                        {Label: "UTC", Latitude: 0, Longitude: 0},
	"America/Los_Angeles":            {Label: "Los Angeles", Latitude: 34.0522, Longitude: -118.2437},
	"America/Denver":                 {Label: "Denver", Latitude: 39.7392, Longitude: -104.9903},
	"America/Chicago":                {Label: "Chicago", Latitude: 41.8781, Longitude: -87.6298},
	"America/New_York":               {Label: "New York", Latitude: 40.7128, Longitude: -74.006},
	"America/Toronto":                {Label: "Toronto", Latitude: 43.6532, Longitude: -79.3832},
	"America/Mexico_City":            {Label: "Mexico City", Latitude: 19.4326, Longitude: -99.1332},
	"America/Sao_Paulo":              {Label: "Sao Paulo", Latitude: -23.5505, Longitude: -46.6333},
	"America/Argentina/Buenos_Aires": {Label: "Buenos Aires", Latitude: -34.6037, Longitude: -58.3816},
	"Europe/London":                  {Label: "London", Latitude: 51.5072, Longitude: -0.1276},
	"Europe/Paris":                   {Label: "Paris", Latitude: 48.8566, Longitude: 2.3522},
	"Europe/Berlin":                  {Label: "Berlin", Latitude: 52.52, Longitude: 13.405},
	"Europe/Moscow":                  {Label: "Moscow", Latitude: 55.7558, Longitude: 37.6173},
	"Africa/Cairo":                   {Label: "Cairo", Latitude: 30.0444, Longitude: 31.2357},
	"Africa/Johannesburg":            {Label: "Johannesburg", Latitude: -26.2041, Longitude: 28.0473},
	"Asia/Dubai":                     {Label: "Dubai", Latitude: 25.2048, Longitude: 55.2708},
	"Asia/Calcutta":                  {Label: "Calcutta", Latitude: 22.5726, Longitude: 88.3639},
	"Asia/Kolkata":                   {Label: "Kolkata", Latitude: 22.5726, Longitude: 88.3639},
	"Asia/Bangkok":                   {Label: "Bangkok", Latitude: 13.7563, Longitude: 100.5018},
	"Asia/Singapore":                 {Label: "Singapore", Latitude: 1.3521, Longitude: 103.8198},
	"Asia/Hong_Kong":                 {Label: "Hong Kong", Latitude: 22.3193, Longitude: 114.1694},
	"Asia/Shanghai":                  {Label: "Shanghai", Latitude: 31.2304, Longitude: 121.4737},
	"Asia/Tokyo":                     {Label: "Tokyo", Latitude: 35.6762, Longitude: 139.6503},
	"Australia/Sydney":               {Label: "Sydney", Latitude: -33.8688, Longitude: 151.2093},
	"Pacific/Auckland":               {Label: "Auckland", Latitude: -36.8509, Longitude: 174.7645},
}

var locationRegionAnchors = map[string]locationAnchor{
	"Africa":     {Label: "Africa", Latitude: 4, Longitude: 21},
	"America":    {Label: "Americas", Latitude: 28, Longitude: -96},
	"Antarctica": {Label: "Antarctica", Latitude: -72, Longitude: 35},
	"Arctic":     {Label: "Arctic", Latitude: 72, Longitude: 0},
	"Asia":       {Label: "Asia", Latitude: 29, Longitude: 88},
	"Atlantic":   {Label: "Atlantic", Latitude: 0, Longitude: -30},
	"Australia":  {Label: "Australia", Latitude: -25, Longitude: 134},
	"Europe":     {Label: "Europe", Latitude: 50, Longitude: 14},
	"Indian":     {Label: "Indian Ocean", Latitude: -15, Longitude: 75},
	"Pacific":    {Label: "Pacific", Latitude: -12, Longitude: -150},
}

var locationLocaleAnchors = map[string]locationAnchor{
	"AE": {Label: "United Arab Emirates", Latitude: 23.4241, Longitude: 53.8478},
	"AR": {Label: "Argentina", Latitude: -34.6037, Longitude: -58.3816},
	"AU": {Label: "Australia", Latitude: -25.2744, Longitude: 133.7751},
	"BR": {Label: "Brazil", Latitude: -14.235, Longitude: -51.9253},
	"CA": {Label: "Canada", Latitude: 56.1304, Longitude: -106.3468},
	"CN": {Label: "China", Latitude: 35.8617, Longitude: 104.1954},
	"DE": {Label: "Germany", Latitude: 51.1657, Longitude: 10.4515},
	"EG": {Label: "Egypt", Latitude: 26.8206, Longitude: 30.8025},
	"FR": {Label: "France", Latitude: 46.2276, Longitude: 2.2137},
	"GB": {Label: "United Kingdom", Latitude: 55.3781, Longitude: -3.436},
	"HK": {Label: "Hong Kong", Latitude: 22.3193, Longitude: 114.1694},
	"IN": {Label: "India", Latitude: 20.5937, Longitude: 78.9629},
	"JP": {Label: "Japan", Latitude: 36.2048, Longitude: 138.2529},
	"MX": {Label: "Mexico", Latitude: 23.6345, Longitude: -102.5528},
	"NZ": {Label: "New Zealand", Latitude: -40.9006, Longitude: 174.886},
	"RU": {Label: "Russia", Latitude: 61.524, Longitude: 105.3188},
	"SG": {Label: "Singapore", Latitude: 1.3521, Longitude: 103.8198},
	"TH": {Label: "Thailand", Latitude: 15.87, Longitude: 100.9925},
	"UA": {Label: "Ukraine", Latitude: 48.3794, Longitude: 31.1656},
	"US": {Label: "United States", Latitude: 39.8283, Longitude: -98.5795},
	"ZA": {Label: "South Africa", Latitude: -30.5595, Longitude: 22.9375},
}

var locationUnmappedCountryAnchor = locationAnchor{Latitude: -53, Longitude: 152}

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
	VisitCountForDays(ctx context.Context, siteID string, days int) (int64, error)
	UniqueVisitorCount(ctx context.Context, siteID string) (int64, error)
	UniqueVisitorCountForDays(ctx context.Context, siteID string, days int) (int64, error)
	TopPages(ctx context.Context, siteID string, limit int) ([]TopPageStat, error)
	TopPagesForDays(ctx context.Context, siteID string, days int, limit int) ([]TopPageStat, error)
	VisitTrend(ctx context.Context, siteID string, days int) ([]DailyVisitTrendStat, error)
	VisitTrendAll(ctx context.Context, siteID string) ([]DailyVisitTrendStat, error)
	VisitAttribution(ctx context.Context, siteID string, limit int) (VisitAttributionBreakdown, error)
	VisitAttributionForDays(ctx context.Context, siteID string, days int, limit int) (VisitAttributionBreakdown, error)
	VisitEngagement(ctx context.Context, siteID string, days int) (VisitEngagementStat, error)
	VisitEngagementAll(ctx context.Context, siteID string) (VisitEngagementStat, error)
	DeviceBreakdown(ctx context.Context, siteID string, limit int) (DeviceBreakdownStat, error)
	DeviceBreakdownForDays(ctx context.Context, siteID string, days int, limit int) (DeviceBreakdownStat, error)
	LocationDistribution(ctx context.Context, siteID string, limit int) ([]LocationDistributionStat, error)
	LocationDistributionForDays(ctx context.Context, siteID string, days int, limit int) ([]LocationDistributionStat, error)
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
	return provider.visitCount(ctx, siteID, time.Time{})
}

// VisitCountForDays returns page views within the same UTC day window used by VisitTrend.
func (provider *DatabaseSiteStatisticsProvider) VisitCountForDays(ctx context.Context, siteID string, days int) (int64, error) {
	normalizedDays := normalizeVisitTrendDays(days)
	return provider.visitCount(ctx, siteID, visitWindowStartDay(normalizedDays))
}

func (provider *DatabaseSiteStatisticsProvider) visitCount(ctx context.Context, siteID string, startDay time.Time) (int64, error) {
	if strings.TrimSpace(siteID) == "" {
		return 0, nil
	}
	var count int64
	query := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Where("site_id = ? AND is_bot = ?", siteID, false)
	if !startDay.IsZero() {
		query = query.Where("occurred_at >= ?", startDay)
	}
	err := query.Count(&count).Error
	return count, err
}

// UniqueVisitorCount returns distinct visitor ids for a site.
func (provider *DatabaseSiteStatisticsProvider) UniqueVisitorCount(ctx context.Context, siteID string) (int64, error) {
	return provider.uniqueVisitorCount(ctx, siteID, time.Time{})
}

// UniqueVisitorCountForDays returns distinct visitor ids within the same UTC day window used by VisitTrend.
func (provider *DatabaseSiteStatisticsProvider) UniqueVisitorCountForDays(ctx context.Context, siteID string, days int) (int64, error) {
	normalizedDays := normalizeVisitTrendDays(days)
	return provider.uniqueVisitorCount(ctx, siteID, visitWindowStartDay(normalizedDays))
}

func (provider *DatabaseSiteStatisticsProvider) uniqueVisitorCount(ctx context.Context, siteID string, startDay time.Time) (int64, error) {
	if strings.TrimSpace(siteID) == "" {
		return 0, nil
	}
	var count int64
	query := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Where("site_id = ? AND visitor_id <> '' AND is_bot = ?", siteID, false)
	if !startDay.IsZero() {
		query = query.Where("occurred_at >= ?", startDay)
	}
	err := query.
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
	return provider.topPages(ctx, siteID, limit, time.Time{})
}

// TopPagesForDays returns top pages by visit count within the same UTC day window used by VisitTrend.
func (provider *DatabaseSiteStatisticsProvider) TopPagesForDays(ctx context.Context, siteID string, days int, limit int) ([]TopPageStat, error) {
	normalizedDays := normalizeVisitTrendDays(days)
	return provider.topPages(ctx, siteID, limit, visitWindowStartDay(normalizedDays))
}

func (provider *DatabaseSiteStatisticsProvider) topPages(ctx context.Context, siteID string, limit int, startDay time.Time) ([]TopPageStat, error) {
	if strings.TrimSpace(siteID) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}
	var results []TopPageStat
	query := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select(topPagesSelectStatement).
		Where("site_id = ? AND path <> '' AND is_bot = ?", siteID, false)
	if !startDay.IsZero() {
		query = query.Where("occurred_at >= ?", startDay)
	}
	err := query.Group(topPagesCanonicalPathExpression).
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

// DeviceTypeStat captures visit counts per device category.
type DeviceTypeStat struct {
	DeviceType string
	VisitCount int64
}

// DeviceBreakdownStat aggregates device type, resolution, and viewport data.
type DeviceBreakdownStat struct {
	DeviceTypes    []DeviceTypeStat
	TopResolutions []AttributionStat
	TopViewports   []AttributionStat
}

// LocationDistributionStat captures visit counts by inferred visitor location.
type LocationDistributionStat struct {
	Label      string
	Source     string
	Signal     string
	Country    string
	Region     string
	City       string
	Latitude   float64
	Longitude  float64
	Confidence int
	VisitCount int64
}

type visitLocationSignals struct {
	Timezone     string
	Locale       string
	IP           string
	GeoSource    string
	GeoCountry   string
	GeoRegion    string
	GeoCity      string
	GeoLatitude  float64
	GeoLongitude float64
}

type visitLocationSignalCountRow struct {
	Timezone     string
	Locale       string
	IP           string
	GeoSource    string
	GeoCountry   string
	GeoRegion    string
	GeoCity      string
	GeoLatitude  float64
	GeoLongitude float64
	VisitCount   int64
}

func (row visitLocationSignalCountRow) signals() visitLocationSignals {
	return visitLocationSignals{
		Timezone:     row.Timezone,
		Locale:       row.Locale,
		IP:           row.IP,
		GeoSource:    row.GeoSource,
		GeoCountry:   row.GeoCountry,
		GeoRegion:    row.GeoRegion,
		GeoCity:      row.GeoCity,
		GeoLatitude:  row.GeoLatitude,
		GeoLongitude: row.GeoLongitude,
	}
}

func (provider *DatabaseSiteStatisticsProvider) VisitTrend(ctx context.Context, siteID string, days int) ([]DailyVisitTrendStat, error) {
	if strings.TrimSpace(siteID) == "" {
		return nil, nil
	}
	normalizedDays := normalizeVisitTrendDays(days)
	return provider.visitTrend(ctx, siteID, visitWindowStartDay(normalizedDays), normalizedDays)
}

// VisitTrendAll returns a day-filled trend from the first recorded human visit through today.
func (provider *DatabaseSiteStatisticsProvider) VisitTrendAll(ctx context.Context, siteID string) ([]DailyVisitTrendStat, error) {
	if strings.TrimSpace(siteID) == "" {
		return nil, nil
	}
	return provider.visitTrend(ctx, siteID, time.Time{}, 0)
}

func (provider *DatabaseSiteStatisticsProvider) visitTrend(ctx context.Context, siteID string, startDay time.Time, days int) ([]DailyVisitTrendStat, error) {
	var rows []dailyVisitTrendRow
	query := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select("DATE(occurred_at) as day, COUNT(*) as page_views, COUNT(DISTINCT CASE WHEN visitor_id <> '' THEN visitor_id END) as unique_visitors").
		Where("site_id = ? AND is_bot = ?", siteID, false)
	if !startDay.IsZero() {
		query = query.Where("occurred_at >= ?", startDay)
	}
	err := query.Group("DATE(occurred_at)").
		Order("day asc").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if startDay.IsZero() {
		if len(rows) == 0 {
			return nil, nil
		}
		_, firstDate, normalizeErr := normalizeVisitTrendMapKey(rows[0].Day)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		startDay = firstDate
		today := time.Now().UTC().Truncate(24 * time.Hour)
		if startDay.After(today) {
			today = startDay
		}
		days = int(today.Sub(startDay).Hours()/24) + 1
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

	trend := make([]DailyVisitTrendStat, 0, days)
	for dayIndex := 0; dayIndex < days; dayIndex++ {
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

func visitWindowStartDay(normalizedDays int) time.Time {
	return time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -(normalizedDays - 1))
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
	return provider.visitAttribution(ctx, siteID, limit, time.Time{})
}

// VisitAttributionForDays returns attribution within the same UTC day window used by VisitTrend.
func (provider *DatabaseSiteStatisticsProvider) VisitAttributionForDays(ctx context.Context, siteID string, days int, limit int) (VisitAttributionBreakdown, error) {
	normalizedDays := normalizeVisitTrendDays(days)
	return provider.visitAttribution(ctx, siteID, limit, visitWindowStartDay(normalizedDays))
}

func (provider *DatabaseSiteStatisticsProvider) visitAttribution(ctx context.Context, siteID string, limit int, startDay time.Time) (VisitAttributionBreakdown, error) {
	if strings.TrimSpace(siteID) == "" {
		return VisitAttributionBreakdown{}, nil
	}

	normalizedLimit := normalizeVisitAttributionLimit(limit)
	var rows []visitAttributionRow
	query := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select("url, referrer").
		Where("site_id = ? AND is_bot = ?", siteID, false)
	if !startDay.IsZero() {
		query = query.Where("occurred_at >= ?", startDay)
	}
	err := query.Scan(&rows).Error
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
	normalizedDays := normalizeVisitEngagementDays(days)
	return provider.visitEngagement(ctx, siteID, visitWindowStartDay(normalizedDays))
}

// VisitEngagementAll returns engagement metrics across all recorded human visits.
func (provider *DatabaseSiteStatisticsProvider) VisitEngagementAll(ctx context.Context, siteID string) (VisitEngagementStat, error) {
	return provider.visitEngagement(ctx, siteID, time.Time{})
}

func (provider *DatabaseSiteStatisticsProvider) visitEngagement(ctx context.Context, siteID string, startDay time.Time) (VisitEngagementStat, error) {
	if strings.TrimSpace(siteID) == "" {
		return VisitEngagementStat{}, nil
	}

	var rows []visitEngagementRow
	query := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select("visitor_id, occurred_at").
		Where("site_id = ? AND is_bot = ? AND visitor_id <> ''", siteID, false)
	if !startDay.IsZero() {
		query = query.Where("occurred_at >= ?", startDay)
	}
	err := query.Scan(&rows).Error
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

type visitDimensionCountRow struct {
	Value      string
	VisitCount int64
}

type viewportCountRow struct {
	Viewport   string
	VisitCount int64
}

func (provider *DatabaseSiteStatisticsProvider) DeviceBreakdown(ctx context.Context, siteID string, limit int) (DeviceBreakdownStat, error) {
	return provider.deviceBreakdown(ctx, siteID, limit, time.Time{})
}

// DeviceBreakdownForDays returns device breakdowns within the same UTC day window used by VisitTrend.
func (provider *DatabaseSiteStatisticsProvider) DeviceBreakdownForDays(ctx context.Context, siteID string, days int, limit int) (DeviceBreakdownStat, error) {
	normalizedDays := normalizeVisitTrendDays(days)
	return provider.deviceBreakdown(ctx, siteID, limit, visitWindowStartDay(normalizedDays))
}

func (provider *DatabaseSiteStatisticsProvider) deviceBreakdown(ctx context.Context, siteID string, limit int, startDay time.Time) (DeviceBreakdownStat, error) {
	if strings.TrimSpace(siteID) == "" {
		return DeviceBreakdownStat{}, nil
	}
	normalizedLimit := normalizeDeviceBreakdownLimit(limit)

	resolutionStats, err := provider.topScreenResolutionStats(ctx, siteID, normalizedLimit, startDay)
	if err != nil {
		return DeviceBreakdownStat{}, err
	}

	viewportRows, err := provider.viewportCounts(ctx, siteID, startDay)
	if err != nil {
		return DeviceBreakdownStat{}, err
	}

	deviceCounts := make(map[string]int64)
	var topViewportStats []AttributionStat
	if len(viewportRows) > 0 {
		topViewportStats = make([]AttributionStat, 0, minInt(len(viewportRows), normalizedLimit))
	}
	for index, row := range viewportRows {
		viewportValue := strings.TrimSpace(row.Viewport)
		if viewportValue == "" {
			continue
		}
		deviceType := classifyDeviceType(viewportValue)
		deviceCounts[deviceType] += row.VisitCount
		if index < normalizedLimit {
			topViewportStats = append(topViewportStats, AttributionStat{
				Value:      viewportValue,
				VisitCount: row.VisitCount,
			})
		}
	}

	return DeviceBreakdownStat{
		DeviceTypes:    topDeviceTypeStats(deviceCounts),
		TopResolutions: resolutionStats,
		TopViewports:   topViewportStats,
	}, nil
}

func (provider *DatabaseSiteStatisticsProvider) topScreenResolutionStats(ctx context.Context, siteID string, limit int, startDay time.Time) ([]AttributionStat, error) {
	var rows []visitDimensionCountRow
	query := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select("screen_resolution as value, COUNT(*) as visit_count").
		Where("site_id = ? AND is_bot = ? AND screen_resolution <> ''", siteID, false)
	if !startDay.IsZero() {
		query = query.Where("occurred_at >= ?", startDay)
	}
	err := query.Group("screen_resolution").
		Order("visit_count desc, screen_resolution asc").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return attributionStatsFromDimensionRows(rows), nil
}

func (provider *DatabaseSiteStatisticsProvider) viewportCounts(ctx context.Context, siteID string, startDay time.Time) ([]viewportCountRow, error) {
	var rows []viewportCountRow
	query := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select("viewport, COUNT(*) as visit_count").
		Where("site_id = ? AND is_bot = ? AND viewport <> ''", siteID, false)
	if !startDay.IsZero() {
		query = query.Where("occurred_at >= ?", startDay)
	}
	err := query.Group("viewport").
		Order("visit_count desc, viewport asc").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func attributionStatsFromDimensionRows(rows []visitDimensionCountRow) []AttributionStat {
	if len(rows) == 0 {
		return nil
	}
	stats := make([]AttributionStat, 0, len(rows))
	for _, row := range rows {
		value := strings.TrimSpace(row.Value)
		if value == "" {
			continue
		}
		stats = append(stats, AttributionStat{
			Value:      value,
			VisitCount: row.VisitCount,
		})
	}
	if len(stats) == 0 {
		return nil
	}
	return stats
}

func classifyDeviceType(viewport string) string {
	parts := strings.SplitN(viewport, "x", 2)
	if len(parts) < 2 {
		return deviceTypeDesktop
	}
	width, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return deviceTypeDesktop
	}
	switch {
	case width < 768:
		return deviceTypeMobile
	case width < 1024:
		return deviceTypeTablet
	default:
		return deviceTypeDesktop
	}
}

func normalizeDeviceBreakdownLimit(limit int) int {
	if limit <= 0 {
		return defaultDeviceBreakdownLimit
	}
	if limit > maxDeviceBreakdownLimit {
		return maxDeviceBreakdownLimit
	}
	return limit
}

func topDeviceTypeStats(counts map[string]int64) []DeviceTypeStat {
	if len(counts) == 0 {
		return nil
	}
	entries := make([]DeviceTypeStat, 0, len(counts))
	for deviceType, visitCount := range counts {
		entries = append(entries, DeviceTypeStat{DeviceType: deviceType, VisitCount: visitCount})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].VisitCount == entries[j].VisitCount {
			return entries[i].DeviceType < entries[j].DeviceType
		}
		return entries[i].VisitCount > entries[j].VisitCount
	})
	return entries
}

func (provider *DatabaseSiteStatisticsProvider) LocationDistribution(ctx context.Context, siteID string, limit int) ([]LocationDistributionStat, error) {
	return provider.locationDistribution(ctx, siteID, limit, time.Time{})
}

// LocationDistributionForDays returns inferred visitor locations within the same UTC day window used by VisitTrend.
func (provider *DatabaseSiteStatisticsProvider) LocationDistributionForDays(ctx context.Context, siteID string, days int, limit int) ([]LocationDistributionStat, error) {
	normalizedDays := normalizeVisitTrendDays(days)
	return provider.locationDistribution(ctx, siteID, limit, visitWindowStartDay(normalizedDays))
}

func (provider *DatabaseSiteStatisticsProvider) locationDistribution(ctx context.Context, siteID string, limit int, startDay time.Time) ([]LocationDistributionStat, error) {
	if strings.TrimSpace(siteID) == "" {
		return nil, nil
	}
	normalizedLimit := normalizeLocationDistributionLimit(limit)

	var rows []visitLocationSignalCountRow
	query := provider.database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select("timezone, locale, ip, geo_source, geo_country, geo_region, geo_city, geo_latitude, geo_longitude, COUNT(*) as visit_count").
		Where("site_id = ? AND is_bot = ?", siteID, false)
	if !startDay.IsZero() {
		query = query.Where("occurred_at >= ?", startDay)
	}
	err := query.
		Group("timezone, locale, ip, geo_source, geo_country, geo_region, geo_city, geo_latitude, geo_longitude").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	type locationDistributionAggregate struct {
		stat            LocationDistributionStat
		confidenceTotal int64
	}
	statsByKey := make(map[string]locationDistributionAggregate)
	for _, row := range rows {
		if row.VisitCount <= 0 {
			continue
		}
		location := inferVisitLocation(row.signals())
		key := location.Source + "\x00" + location.Signal + "\x00" + location.Label + "\x00" + location.Country + "\x00" + location.Region + "\x00" + location.City
		current := statsByKey[key]
		if current.stat.Label == "" {
			current.stat = location
		}
		current.stat.VisitCount += row.VisitCount
		current.confidenceTotal += int64(location.Confidence) * row.VisitCount
		statsByKey[key] = current
	}

	stats := make([]LocationDistributionStat, 0, len(statsByKey))
	for _, aggregate := range statsByKey {
		stat := aggregate.stat
		if stat.VisitCount > 0 {
			stat.Confidence = int((aggregate.confidenceTotal + stat.VisitCount/2) / stat.VisitCount)
		}
		stats = append(stats, stat)
	}
	sort.Slice(stats, func(leftIndex int, rightIndex int) bool {
		if stats[leftIndex].VisitCount == stats[rightIndex].VisitCount {
			if stats[leftIndex].Confidence != stats[rightIndex].Confidence {
				return stats[leftIndex].Confidence > stats[rightIndex].Confidence
			}
			if stats[leftIndex].Label == stats[rightIndex].Label {
				return stats[leftIndex].Source < stats[rightIndex].Source
			}
			return stats[leftIndex].Label < stats[rightIndex].Label
		}
		return stats[leftIndex].VisitCount > stats[rightIndex].VisitCount
	})
	if len(stats) > normalizedLimit {
		stats = stats[:normalizedLimit]
	}
	return stats, nil
}

func normalizeLocationDistributionLimit(limit int) int {
	if limit <= 0 {
		return defaultLocationDistributionLimit
	}
	if limit > maxLocationDistributionLimit {
		return maxLocationDistributionLimit
	}
	return limit
}

func inferVisitLocation(signals visitLocationSignals) LocationDistributionStat {
	edgeGeoLocation, edgeGeoOK := inferEdgeGeoLocation(signals)
	if edgeGeoOK {
		return edgeGeoLocation
	}
	timezoneLocation, timezoneOK := inferTimezoneLocation(signals)
	if timezoneOK {
		return timezoneLocation
	}
	localeLocation, localeOK := inferLocaleLocation(signals.Locale)
	if localeOK {
		return localeLocation
	}
	networkLocation, networkOK := inferNetworkLocation(signals.IP)
	if networkOK {
		return networkLocation
	}
	unknownSignal := locationUnknownSignal
	if strings.TrimSpace(signals.Timezone) != "" {
		unknownSignal = locationTimezoneUnknownSignal
	}
	return LocationDistributionStat{
		Label:      locationUnknownLabel,
		Source:     locationSourceUnknown,
		Signal:     unknownSignal,
		Latitude:   -53,
		Longitude:  152,
		Confidence: locationConfidenceUnknown,
	}
}

func inferEdgeGeoLocation(signals visitLocationSignals) (LocationDistributionStat, bool) {
	country := normalizeLocationCountry(signals.GeoCountry)
	region := normalizeLocationText(signals.GeoRegion)
	city := normalizeLocationText(signals.GeoCity)
	source := normalizeLocationText(signals.GeoSource)
	if source == "" {
		source = "edge"
	}
	hasCoordinates := edgeLocationHasCoordinates(signals.GeoLatitude, signals.GeoLongitude)
	if country == "" && region == "" && city == "" && !hasCoordinates {
		return LocationDistributionStat{}, false
	}

	anchor, hasCountryAnchor := locationLocaleAnchors[country]
	latitude := signals.GeoLatitude
	longitude := signals.GeoLongitude
	if !hasCoordinates {
		if hasCountryAnchor {
			latitude = anchor.Latitude
			longitude = anchor.Longitude
		} else {
			latitude = locationUnmappedCountryAnchor.Latitude
			longitude = locationUnmappedCountryAnchor.Longitude
		}
	}

	label := edgeLocationLabel(country, region, city, anchor, hasCountryAnchor)
	if label == "" && hasCoordinates {
		label = "Edge coordinates"
	}
	if label == "" {
		return LocationDistributionStat{}, false
	}

	confidence := locationConfidenceEdgeCountry
	if hasCoordinates {
		confidence = locationConfidenceEdgeCoordinate
	}
	if hasCoordinates && city != "" {
		confidence = locationConfidenceEdgeExact
	}
	if country != "" && country == normalizeLocaleRegion(signals.Locale) {
		confidence += 2
	}
	if country != "" && country == locationTimezoneCountry(strings.TrimSpace(signals.Timezone)) {
		confidence += 2
	}
	if confidence > 99 {
		confidence = 99
	}

	return LocationDistributionStat{
		Label:      label,
		Source:     locationSourceEdgeGeo,
		Signal:     edgeLocationSignal(source, country, region, city),
		Country:    country,
		Region:     region,
		City:       city,
		Latitude:   latitude,
		Longitude:  longitude,
		Confidence: confidence,
	}, true
}

func inferTimezoneLocation(signals visitLocationSignals) (LocationDistributionStat, bool) {
	normalizedTimezone := strings.TrimSpace(signals.Timezone)
	if normalizedTimezone == "" || normalizedTimezone == "Etc/Unknown" || strings.EqualFold(normalizedTimezone, "unknown") {
		return LocationDistributionStat{}, false
	}
	country := locationTimezoneCountry(normalizedTimezone)
	confidence := locationConfidenceTimezone
	if country != "" && country == normalizeLocaleRegion(signals.Locale) {
		confidence = locationConfidenceTimezoneAgree
	}
	if anchor, ok := locationTimezoneAnchors[normalizedTimezone]; ok {
		return locationStat(anchor, locationSourceTimezone, normalizedTimezone, country, confidence), true
	}
	parts := strings.Split(normalizedTimezone, "/")
	region := ""
	if len(parts) > 0 {
		region = strings.TrimSpace(parts[0])
	}
	anchor, ok := locationRegionAnchors[region]
	if !ok {
		return LocationDistributionStat{}, false
	}
	hash := hashLocationSignal(normalizedTimezone)
	label := normalizedTimezone
	if len(parts) > 0 {
		label = strings.ReplaceAll(strings.TrimSpace(parts[len(parts)-1]), "_", " ")
	}
	return LocationDistributionStat{
		Label:      label,
		Source:     locationSourceTimezone,
		Signal:     normalizedTimezone,
		Country:    country,
		Latitude:   clampFloat(anchor.Latitude+float64((hash%13)-6)*1.4, -74, 74),
		Longitude:  clampFloat(anchor.Longitude+float64(((hash>>3)%17)-8)*2.2, -176, 176),
		Confidence: confidence,
	}, true
}

func inferLocaleLocation(localeValue string) (LocationDistributionStat, bool) {
	region := normalizeLocaleRegion(localeValue)
	if region == "" {
		return LocationDistributionStat{}, false
	}
	anchor, ok := locationLocaleAnchors[region]
	if !ok {
		return LocationDistributionStat{}, false
	}
	return locationStat(anchor, locationSourceLocale, region, region, locationConfidenceLocale), true
}

func inferNetworkLocation(ipValue string) (LocationDistributionStat, bool) {
	parsedIP := net.ParseIP(strings.TrimSpace(ipValue))
	if parsedIP == nil || (!parsedIP.IsLoopback() && !parsedIP.IsPrivate() && !parsedIP.IsLinkLocalUnicast() && !parsedIP.IsLinkLocalMulticast()) {
		return LocationDistributionStat{}, false
	}
	return LocationDistributionStat{
		Label:      "Local network",
		Source:     locationSourceNetwork,
		Signal:     locationNetworkLocalSignal,
		Latitude:   0,
		Longitude:  0,
		Confidence: locationConfidenceNetwork,
	}, true
}

func locationStat(anchor locationAnchor, source string, signal string, country string, confidence int) LocationDistributionStat {
	return LocationDistributionStat{
		Label:      anchor.Label,
		Source:     source,
		Signal:     signal,
		Country:    country,
		Latitude:   anchor.Latitude,
		Longitude:  anchor.Longitude,
		Confidence: confidence,
	}
}

func edgeLocationHasCoordinates(latitude float64, longitude float64) bool {
	return latitude != 0 || longitude != 0
}

func edgeLocationLabel(country string, region string, city string, anchor locationAnchor, hasCountryAnchor bool) string {
	if city != "" && region != "" {
		return city + ", " + region
	}
	if city != "" {
		return city
	}
	if region != "" && country != "" {
		return region + ", " + country
	}
	if region != "" {
		return region
	}
	if hasCountryAnchor {
		return anchor.Label
	}
	return country
}

func edgeLocationSignal(source string, country string, region string, city string) string {
	parts := []string{source}
	for _, value := range []string{country, region, city} {
		normalizedValue := strings.TrimSpace(value)
		if normalizedValue != "" {
			parts = append(parts, normalizedValue)
		}
	}
	return strings.Join(parts, ":")
}

func normalizeLocationCountry(rawCountry string) string {
	country := strings.ToUpper(strings.TrimSpace(rawCountry))
	if country == "" || country == "XX" || country == "T1" || country == "A1" || country == "A2" {
		return ""
	}
	if len(country) != 2 && len(country) != 3 {
		return ""
	}
	for _, character := range country {
		if character < 'A' || character > 'Z' {
			return ""
		}
	}
	return country
}

func normalizeLocationText(rawValue string) string {
	trimmedValue := strings.TrimSpace(rawValue)
	if trimmedValue == "" {
		return ""
	}
	return strings.Map(func(character rune) rune {
		if character < 0x20 || character == 0x7f {
			return -1
		}
		return character
	}, trimmedValue)
}

func locationTimezoneCountry(timezoneValue string) string {
	switch strings.TrimSpace(timezoneValue) {
	case "America/Los_Angeles", "America/Denver", "America/Chicago", "America/New_York":
		return "US"
	case "America/Toronto":
		return "CA"
	case "America/Mexico_City":
		return "MX"
	case "America/Sao_Paulo":
		return "BR"
	case "America/Argentina/Buenos_Aires":
		return "AR"
	case "Europe/London":
		return "GB"
	case "Europe/Paris":
		return "FR"
	case "Europe/Berlin":
		return "DE"
	case "Europe/Moscow":
		return "RU"
	case "Africa/Cairo":
		return "EG"
	case "Africa/Johannesburg":
		return "ZA"
	case "Asia/Dubai":
		return "AE"
	case "Asia/Calcutta", "Asia/Kolkata":
		return "IN"
	case "Asia/Bangkok":
		return "TH"
	case "Asia/Singapore":
		return "SG"
	case "Asia/Hong_Kong":
		return "HK"
	case "Asia/Shanghai":
		return "CN"
	case "Asia/Tokyo":
		return "JP"
	case "Australia/Sydney":
		return "AU"
	case "Pacific/Auckland":
		return "NZ"
	default:
		return ""
	}
}

func normalizeLocaleRegion(localeValue string) string {
	primaryTag := strings.TrimSpace(strings.Split(strings.TrimSpace(localeValue), ",")[0])
	primaryTag = strings.TrimSpace(strings.Split(primaryTag, ";")[0])
	if primaryTag == "" {
		return ""
	}
	parts := strings.FieldsFunc(primaryTag, func(character rune) bool {
		return character == '-' || character == '_'
	})
	for _, part := range parts[1:] {
		normalizedPart := strings.ToUpper(strings.TrimSpace(part))
		if len(normalizedPart) == 2 || len(normalizedPart) == 3 {
			return normalizedPart
		}
	}
	return ""
}

func hashLocationSignal(signal string) int {
	hash := 0
	for _, character := range signal {
		hash = (hash*31 + int(character)) % 9973
	}
	return hash
}

func clampFloat(value float64, minimum float64, maximum float64) float64 {
	return math.Max(minimum, math.Min(maximum, value))
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
