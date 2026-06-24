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
	FeedbackSourceWebWidget = "web_widget"
	FeedbackSourceMobileApp = "mobile_app"

	MobilePlatformAndroid = "android"
	MobilePlatformIOS     = "ios"

	mobileAppClientIDMaxLength      = 80
	mobileAppPlatformMaxLength      = 20
	mobileAppIdentifierMaxLength    = 200
	mobileAppDisplayNameMaxLength   = 200
	feedbackSourceKindMaxLength     = 20
	feedbackSourceURLMaxLength      = 500
	feedbackScreenNameMaxLength     = 120
	feedbackScreenPathMaxLength     = 300
	feedbackAppPlatformMaxLength    = 20
	feedbackAppIdentifierMaxLength  = 200
	feedbackAppVersionMaxLength     = 80
	feedbackAppBuildMaxLength       = 80
	feedbackAppEnvironmentMaxLength = 80
	feedbackContextJSONMaxLength    = 8192
)

var (
	ErrInvalidMobileAppSiteID      = errors.New("invalid_mobile_app_site_id")
	ErrInvalidMobileAppClientID    = errors.New("invalid_mobile_app_client_id")
	ErrInvalidMobileAppPlatform    = errors.New("invalid_mobile_app_platform")
	ErrInvalidMobileAppIdentifier  = errors.New("invalid_mobile_app_identifier")
	ErrInvalidMobileAppDisplayName = errors.New("invalid_mobile_app_display_name")
	ErrInvalidFeedbackSource       = errors.New("invalid_feedback_source")
	ErrInvalidFeedbackSourceURL    = errors.New("invalid_feedback_source_url")
	ErrInvalidFeedbackContext      = errors.New("invalid_feedback_context")
)

// SiteMobileApp identifies one native mobile app allowed to submit feedback for a site.
type SiteMobileApp struct {
	ID            string `gorm:"primaryKey;size:36"`
	SiteID        string `gorm:"not null;size:36;index;uniqueIndex:idx_site_mobile_apps_site_client,priority:1;index:idx_site_mobile_apps_site_enabled,priority:1"`
	ClientID      string `gorm:"not null;size:80;uniqueIndex:idx_site_mobile_apps_site_client,priority:2"`
	Platform      string `gorm:"not null;size:20"`
	AppIdentifier string `gorm:"not null;size:200"`
	DisplayName   string `gorm:"not null;size:200"`
	Enabled       bool   `gorm:"not null;default:true;index:idx_site_mobile_apps_site_enabled,priority:2"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SiteMobileAppInput holds raw values used to register a mobile app.
type SiteMobileAppInput struct {
	SiteID        string
	ClientID      string
	Platform      string
	AppIdentifier string
	DisplayName   string
}

// NewSiteMobileApp constructs a validated mobile app registration.
func NewSiteMobileApp(input SiteMobileAppInput) (SiteMobileApp, error) {
	siteID := strings.TrimSpace(input.SiteID)
	if siteID == "" {
		return SiteMobileApp{}, ErrInvalidMobileAppSiteID
	}

	clientID := strings.TrimSpace(input.ClientID)
	if clientID == "" {
		clientID = uuid.NewString()
	}
	if len(clientID) > mobileAppClientIDMaxLength {
		return SiteMobileApp{}, fmt.Errorf("%w: too long", ErrInvalidMobileAppClientID)
	}

	platform, platformErr := NormalizeMobilePlatform(input.Platform)
	if platformErr != nil {
		return SiteMobileApp{}, platformErr
	}

	appIdentifier := strings.TrimSpace(input.AppIdentifier)
	if appIdentifier == "" || len(appIdentifier) > mobileAppIdentifierMaxLength {
		return SiteMobileApp{}, fmt.Errorf("%w: empty or too long", ErrInvalidMobileAppIdentifier)
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = appIdentifier
	}
	if len(displayName) > mobileAppDisplayNameMaxLength {
		return SiteMobileApp{}, fmt.Errorf("%w: too long", ErrInvalidMobileAppDisplayName)
	}

	return SiteMobileApp{
		ID:            uuid.NewString(),
		SiteID:        siteID,
		ClientID:      clientID,
		Platform:      platform,
		AppIdentifier: appIdentifier,
		DisplayName:   displayName,
		Enabled:       true,
	}, nil
}

// NormalizeMobilePlatform returns the canonical mobile platform value.
func NormalizeMobilePlatform(rawInput string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(rawInput))
	if len(normalized) > mobileAppPlatformMaxLength {
		return "", fmt.Errorf("%w: too long", ErrInvalidMobileAppPlatform)
	}
	switch normalized {
	case MobilePlatformAndroid, MobilePlatformIOS:
		return normalized, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidMobileAppPlatform, normalized)
	}
}

// NormalizeFeedbackSource returns the canonical feedback source kind.
func NormalizeFeedbackSource(rawInput string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(rawInput))
	if normalized == "" {
		return FeedbackSourceWebWidget, nil
	}
	if len(normalized) > feedbackSourceKindMaxLength {
		return "", fmt.Errorf("%w: too long", ErrInvalidFeedbackSource)
	}
	switch normalized {
	case FeedbackSourceWebWidget, FeedbackSourceMobileApp:
		return normalized, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidFeedbackSource, normalized)
	}
}

// NormalizeFeedbackSourceURL returns a canonical source page URL for web feedback.
func NormalizeFeedbackSourceURL(rawInput string) (string, error) {
	normalized := strings.TrimSpace(rawInput)
	if normalized == "" {
		return "", nil
	}
	if len(normalized) > feedbackSourceURLMaxLength {
		return "", fmt.Errorf("%w: too long", ErrInvalidFeedbackSourceURL)
	}
	parsedURL, parseErr := url.Parse(normalized)
	if parseErr != nil || parsedURL == nil {
		return "", fmt.Errorf("%w: invalid", ErrInvalidFeedbackSourceURL)
	}
	scheme := strings.ToLower(strings.TrimSpace(parsedURL.Scheme))
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("%w: unsupported scheme", ErrInvalidFeedbackSourceURL)
	}
	if strings.TrimSpace(parsedURL.Host) == "" {
		return "", fmt.Errorf("%w: missing host", ErrInvalidFeedbackSourceURL)
	}
	if parsedURL.User != nil {
		return "", fmt.Errorf("%w: credentials not allowed", ErrInvalidFeedbackSourceURL)
	}
	return normalized, nil
}

// TruncateFeedbackScreenName limits screen names stored with feedback records.
func TruncateFeedbackScreenName(value string) string {
	return truncateString(strings.TrimSpace(value), feedbackScreenNameMaxLength)
}

// TruncateFeedbackScreenPath limits screen paths stored with feedback records.
func TruncateFeedbackScreenPath(value string) string {
	return truncateString(strings.TrimSpace(value), feedbackScreenPathMaxLength)
}

// TruncateFeedbackAppPlatform limits app platform values stored with feedback records.
func TruncateFeedbackAppPlatform(value string) string {
	return truncateString(strings.TrimSpace(value), feedbackAppPlatformMaxLength)
}

// TruncateFeedbackAppIdentifier limits app identifiers stored with feedback records.
func TruncateFeedbackAppIdentifier(value string) string {
	return truncateString(strings.TrimSpace(value), feedbackAppIdentifierMaxLength)
}

// TruncateFeedbackAppVersion limits app versions stored with feedback records.
func TruncateFeedbackAppVersion(value string) string {
	return truncateString(strings.TrimSpace(value), feedbackAppVersionMaxLength)
}

// TruncateFeedbackAppBuild limits app build values stored with feedback records.
func TruncateFeedbackAppBuild(value string) string {
	return truncateString(strings.TrimSpace(value), feedbackAppBuildMaxLength)
}

// TruncateFeedbackAppEnvironment limits app environment values stored with feedback records.
func TruncateFeedbackAppEnvironment(value string) string {
	return truncateString(strings.TrimSpace(value), feedbackAppEnvironmentMaxLength)
}

// ValidateFeedbackContextJSON validates the bounded mobile feedback context payload.
func ValidateFeedbackContextJSON(value string) error {
	if len(value) > feedbackContextJSONMaxLength {
		return fmt.Errorf("%w: too long", ErrInvalidFeedbackContext)
	}
	return nil
}
