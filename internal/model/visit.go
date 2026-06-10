package model

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	VisitStatusRecorded = "recorded"

	visitURLMaxLength              = 500
	visitPathMaxLength             = 300
	visitIPMaxLength               = 64
	visitUserAgentMaxLength        = 400
	visitScreenResolutionMaxLength = 20
	visitViewportMaxLength         = 20
	visitTimezoneMaxLength         = 60
	visitLocaleMaxLength           = 35
	visitGeoSourceMaxLength        = 30
	visitGeoCountryMaxLength       = 3
	visitGeoRegionMaxLength        = 80
	visitGeoCityMaxLength          = 120
	visitPageTitleMaxLength        = 200
)

var (
	ErrInvalidVisitSiteID = errors.New("invalid_visit_site_id")
	ErrInvalidVisitURL    = errors.New("invalid_visit_url")
	ErrInvalidVisitID     = errors.New("invalid_visit_id")
)

// SiteVisit captures a single page view.
type SiteVisit struct {
	ID               string `gorm:"primaryKey;size:36"`
	SiteID           string `gorm:"not null;size:36;index;index:idx_site_visits_site_bot_occurred,priority:1;index:idx_site_visits_site_bot_path,priority:1;index:idx_site_visits_site_bot_visitor,priority:1;index:idx_site_visits_site_bot_resolution,priority:1;index:idx_site_visits_site_bot_viewport,priority:1;index:idx_site_visits_site_bot_timezone,priority:1;index:idx_site_visits_site_bot_locale,priority:1;index:idx_site_visits_site_bot_geo_source,priority:1;index:idx_site_visits_site_bot_geo_country,priority:1"`
	URL              string `gorm:"size:500"`
	Path             string `gorm:"size:300;index;index:idx_site_visits_site_bot_path,priority:3"`
	VisitorID        string `gorm:"size:36;index;index:idx_site_visits_site_bot_visitor,priority:3"`
	IP               string `gorm:"size:64"`
	UserAgent        string `gorm:"size:400"`
	Referrer         string `gorm:"size:500"`
	IsBot            bool   `gorm:"not null;default:false;index;index:idx_site_visits_site_bot_occurred,priority:2;index:idx_site_visits_site_bot_path,priority:2;index:idx_site_visits_site_bot_visitor,priority:2;index:idx_site_visits_site_bot_resolution,priority:2;index:idx_site_visits_site_bot_viewport,priority:2;index:idx_site_visits_site_bot_timezone,priority:2;index:idx_site_visits_site_bot_locale,priority:2;index:idx_site_visits_site_bot_geo_source,priority:2;index:idx_site_visits_site_bot_geo_country,priority:2"`
	ScreenResolution string `gorm:"size:20;index:idx_site_visits_site_bot_resolution,priority:3"`
	Viewport         string `gorm:"size:20;index:idx_site_visits_site_bot_viewport,priority:3"`
	Timezone         string `gorm:"size:60;index:idx_site_visits_site_bot_timezone,priority:3"`
	Locale           string `gorm:"size:35;index:idx_site_visits_site_bot_locale,priority:3"`
	GeoSource        string `gorm:"size:30;index:idx_site_visits_site_bot_geo_source,priority:3"`
	GeoCountry       string `gorm:"size:3;index:idx_site_visits_site_bot_geo_country,priority:3"`
	GeoRegion        string `gorm:"size:80"`
	GeoCity          string `gorm:"size:120"`
	GeoLatitude      float64
	GeoLongitude     float64
	PageTitle        string    `gorm:"size:200"`
	Status           string    `gorm:"size:20"`
	OccurredAt       time.Time `gorm:"not null;index;index:idx_site_visits_site_bot_occurred,priority:3"`
}

// SiteVisitInput holds incoming visit data.
type SiteVisitInput struct {
	SiteID           string
	URL              string
	VisitorID        string
	IP               string
	UserAgent        string
	Referrer         string
	IsBot            bool
	ScreenResolution string
	Viewport         string
	Timezone         string
	Locale           string
	GeoSource        string
	GeoCountry       string
	GeoRegion        string
	GeoCity          string
	GeoLatitude      float64
	GeoLongitude     float64
	PageTitle        string
	Occurred         time.Time
}

// NewSiteVisit constructs a validated SiteVisit.
func NewSiteVisit(input SiteVisitInput) (SiteVisit, error) {
	siteID := strings.TrimSpace(input.SiteID)
	if siteID == "" {
		return SiteVisit{}, ErrInvalidVisitSiteID
	}
	occurred := input.Occurred
	if occurred.IsZero() {
		occurred = time.Now().UTC()
	}

	normalizedURL, path, urlErr := normalizeVisitURL(input.URL)
	if urlErr != nil {
		return SiteVisit{}, urlErr
	}

	visitorID := strings.TrimSpace(input.VisitorID)
	if len(visitorID) > 0 && len(visitorID) != 36 {
		return SiteVisit{}, ErrInvalidVisitID
	}

	ip := truncateString(input.IP, visitIPMaxLength)
	userAgent := truncateString(input.UserAgent, visitUserAgentMaxLength)
	referrer := truncateString(strings.TrimSpace(input.Referrer), visitURLMaxLength)
	screenResolution := truncateString(strings.TrimSpace(input.ScreenResolution), visitScreenResolutionMaxLength)
	viewport := truncateString(strings.TrimSpace(input.Viewport), visitViewportMaxLength)
	timezone := truncateString(strings.TrimSpace(input.Timezone), visitTimezoneMaxLength)
	locale := truncateString(strings.TrimSpace(input.Locale), visitLocaleMaxLength)
	geoSource := truncateString(strings.TrimSpace(input.GeoSource), visitGeoSourceMaxLength)
	geoCountry := truncateString(strings.ToUpper(strings.TrimSpace(input.GeoCountry)), visitGeoCountryMaxLength)
	geoRegion := truncateString(strings.TrimSpace(input.GeoRegion), visitGeoRegionMaxLength)
	geoCity := truncateString(strings.TrimSpace(input.GeoCity), visitGeoCityMaxLength)
	pageTitle := truncateString(strings.TrimSpace(input.PageTitle), visitPageTitleMaxLength)

	return SiteVisit{
		ID:               uuid.NewString(),
		SiteID:           siteID,
		URL:              normalizedURL,
		Path:             path,
		VisitorID:        visitorID,
		IP:               ip,
		UserAgent:        userAgent,
		Referrer:         referrer,
		IsBot:            input.IsBot,
		ScreenResolution: screenResolution,
		Viewport:         viewport,
		Timezone:         timezone,
		Locale:           locale,
		GeoSource:        geoSource,
		GeoCountry:       geoCountry,
		GeoRegion:        geoRegion,
		GeoCity:          geoCity,
		GeoLatitude:      input.GeoLatitude,
		GeoLongitude:     input.GeoLongitude,
		PageTitle:        pageTitle,
		Status:           VisitStatusRecorded,
		OccurredAt:       occurred,
	}, nil
}

func normalizeVisitURL(raw string) (string, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", ErrInvalidVisitURL
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", fmt.Errorf("%w: %v", ErrInvalidVisitURL, err)
	}
	parsed.Fragment = ""
	normalized := parsed.String()
	if len(normalized) > visitURLMaxLength {
		normalized = normalized[:visitURLMaxLength]
	}
	path := parsed.Path
	if len(path) > visitPathMaxLength {
		path = path[:visitPathMaxLength]
	}
	return normalized, path, nil
}

func truncateString(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
