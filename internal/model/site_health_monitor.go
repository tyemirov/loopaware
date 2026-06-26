package model

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/MarkoPoloResearchLab/loopaware/pkg/outbound"
)

const (
	SiteHealthStatusUnknown = "unknown"
	SiteHealthStatusUp      = "up"
	SiteHealthStatusDown    = "down"

	SiteHealthEventKindDown      = "down"
	SiteHealthEventKindRecovered = "recovered"

	SiteHealthErrorNone          = ""
	SiteHealthErrorHTTP5xx       = "http_5xx"
	SiteHealthErrorTimeout       = "timeout"
	SiteHealthErrorDNS           = "dns"
	SiteHealthErrorTLS           = "tls"
	SiteHealthErrorRedirect      = "redirect"
	SiteHealthErrorNetwork       = "network"
	SiteHealthErrorInvalidTarget = "invalid_target"

	DefaultSiteHealthIntervalSeconds  = 300
	DefaultSiteHealthTimeoutSeconds   = 10
	DefaultSiteHealthFailureThreshold = 2

	MinSiteHealthIntervalSeconds    = 60
	MaxSiteHealthIntervalSeconds    = 86400
	MinSiteHealthTimeoutSeconds     = 1
	MaxSiteHealthTimeoutSeconds     = 30
	MinSiteHealthFailureThreshold   = 1
	MaxSiteHealthFailureThreshold   = 10
	siteHealthSiteIDMaxLength       = 36
	siteHealthTargetURLMaxLength    = 500
	siteHealthErrorCodeMaxLength    = 64
	siteHealthErrorMessageMaxLength = 500
)

var ErrInvalidSiteHealthMonitor = errors.New("invalid_site_health_monitor")

// SiteHealthMonitor stores one site's current uptime monitor configuration and state.
type SiteHealthMonitor struct {
	ID                  string    `gorm:"primaryKey;size:36"`
	SiteID              string    `gorm:"not null;size:36;uniqueIndex"`
	Enabled             bool      `gorm:"not null;default:false;index"`
	TargetURL           string    `gorm:"not null;size:500"`
	IntervalSeconds     int       `gorm:"not null"`
	TimeoutSeconds      int       `gorm:"not null"`
	FailureThreshold    int       `gorm:"not null"`
	RecipientEmail      string    `gorm:"not null;size:320"`
	RecipientMode       string    `gorm:"not null;size:16;default:manager"`
	RecipientEmails     string    `gorm:"type:text"`
	Status              string    `gorm:"not null;size:16;default:unknown;index"`
	ConsecutiveFailures int       `gorm:"not null;default:0"`
	NextCheckAt         time.Time `gorm:"index"`
	LastCheckedAt       time.Time
	LastSuccessAt       time.Time
	LastFailureAt       time.Time
	LastStatusCode      int    `gorm:"not null;default:0"`
	LastErrorCode       string `gorm:"size:64"`
	LastErrorMessage    string `gorm:"size:500"`
	LastDurationMs      int    `gorm:"not null;default:0"`
	LastAlertedStatus   string `gorm:"size:16"`
	LastAlertedAt       time.Time
	CreatedAt           time.Time `gorm:"autoCreateTime"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime"`
}

// SiteHealthEvent records health status transitions for a site monitor.
type SiteHealthEvent struct {
	ID               string    `gorm:"primaryKey;size:36"`
	SiteID           string    `gorm:"not null;size:36;index"`
	MonitorID        string    `gorm:"not null;size:36;index"`
	Kind             string    `gorm:"not null;size:32;index"`
	Status           string    `gorm:"not null;size:16"`
	TargetURL        string    `gorm:"not null;size:500"`
	HTTPStatus       int       `gorm:"not null;default:0"`
	ErrorCode        string    `gorm:"size:64"`
	ErrorMessage     string    `gorm:"size:500"`
	DurationMs       int       `gorm:"not null;default:0"`
	ConsecutiveFails int       `gorm:"not null;default:0"`
	CreatedAt        time.Time `gorm:"autoCreateTime;index"`
}

// SiteHealthMonitorInput captures raw monitor values from configuration/API edges.
type SiteHealthMonitorInput struct {
	SiteID           string
	Enabled          bool
	TargetURL        string
	IntervalSeconds  int
	TimeoutSeconds   int
	FailureThreshold int
	RecipientEmail   string
	RecipientMode    string
	RecipientEmails  []string
	ReferenceTime    time.Time
}

// NewSiteHealthMonitor validates and normalizes one site's monitor.
func NewSiteHealthMonitor(input SiteHealthMonitorInput) (SiteHealthMonitor, error) {
	siteID := strings.TrimSpace(input.SiteID)
	if siteID == "" || len(siteID) > siteHealthSiteIDMaxLength {
		return SiteHealthMonitor{}, fmt.Errorf("%w: invalid site_id", ErrInvalidSiteHealthMonitor)
	}

	targetURL, targetErr := NormalizeSiteHealthTargetURL(input.TargetURL)
	if targetErr != nil {
		return SiteHealthMonitor{}, targetErr
	}

	intervalSeconds := input.IntervalSeconds
	if intervalSeconds == 0 {
		intervalSeconds = DefaultSiteHealthIntervalSeconds
	}
	if intervalSeconds < MinSiteHealthIntervalSeconds || intervalSeconds > MaxSiteHealthIntervalSeconds {
		return SiteHealthMonitor{}, fmt.Errorf("%w: invalid interval_seconds", ErrInvalidSiteHealthMonitor)
	}

	timeoutSeconds := input.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = DefaultSiteHealthTimeoutSeconds
	}
	if timeoutSeconds < MinSiteHealthTimeoutSeconds || timeoutSeconds > MaxSiteHealthTimeoutSeconds {
		return SiteHealthMonitor{}, fmt.Errorf("%w: invalid timeout_seconds", ErrInvalidSiteHealthMonitor)
	}

	failureThreshold := input.FailureThreshold
	if failureThreshold == 0 {
		failureThreshold = DefaultSiteHealthFailureThreshold
	}
	if failureThreshold < MinSiteHealthFailureThreshold || failureThreshold > MaxSiteHealthFailureThreshold {
		return SiteHealthMonitor{}, fmt.Errorf("%w: invalid failure_threshold", ErrInvalidSiteHealthMonitor)
	}

	recipientEmail, recipientErr := NormalizeSiteRecipientEmail(input.RecipientEmail)
	if recipientErr != nil {
		return SiteHealthMonitor{}, fmt.Errorf("%w: %v", ErrInvalidSiteHealthMonitor, recipientErr)
	}
	recipientMode, recipientModeErr := NormalizeSiteRecipientMode(input.RecipientMode)
	if recipientModeErr != nil {
		return SiteHealthMonitor{}, fmt.Errorf("%w: %v", ErrInvalidSiteHealthMonitor, recipientModeErr)
	}
	recipientEmails, recipientEmailsErr := NormalizeSiteRecipientEmailList(input.RecipientEmails)
	if recipientEmailsErr != nil {
		return SiteHealthMonitor{}, fmt.Errorf("%w: %v", ErrInvalidSiteHealthMonitor, recipientEmailsErr)
	}
	if recipientMode == SiteRecipientModeSelected && len(recipientEmails) == 0 {
		return SiteHealthMonitor{}, fmt.Errorf("%w: missing recipient_emails", ErrInvalidSiteHealthMonitor)
	}
	if recipientMode != SiteRecipientModeSelected {
		recipientEmails = []string{}
	}
	encodedRecipientEmails, encodeErr := EncodeSiteRecipientEmails(recipientEmails)
	if encodeErr != nil {
		return SiteHealthMonitor{}, fmt.Errorf("%w: %v", ErrInvalidSiteHealthMonitor, encodeErr)
	}

	referenceTime := input.ReferenceTime
	if referenceTime.IsZero() {
		referenceTime = time.Now().UTC()
	}

	monitor := SiteHealthMonitor{
		ID:               uuid.NewString(),
		SiteID:           siteID,
		Enabled:          input.Enabled,
		TargetURL:        targetURL,
		IntervalSeconds:  intervalSeconds,
		TimeoutSeconds:   timeoutSeconds,
		FailureThreshold: failureThreshold,
		RecipientEmail:   recipientEmail,
		RecipientMode:    recipientMode,
		RecipientEmails:  encodedRecipientEmails,
		Status:           SiteHealthStatusUnknown,
	}
	if monitor.Enabled {
		monitor.NextCheckAt = referenceTime.UTC()
	}
	return monitor, nil
}

// NormalizeSiteHealthTargetURL returns the canonical health-check target URL.
func NormalizeSiteHealthTargetURL(rawTarget string) (string, error) {
	target := strings.TrimSpace(rawTarget)
	if target == "" || len(target) > siteHealthTargetURLMaxLength {
		return "", fmt.Errorf("%w: invalid target_url", ErrInvalidSiteHealthMonitor)
	}
	parsedURL, parseErr := url.Parse(target)
	if parseErr != nil || parsedURL == nil {
		return "", fmt.Errorf("%w: invalid target_url", ErrInvalidSiteHealthMonitor)
	}
	parsedURL.Fragment = ""
	if outboundErr := outbound.ValidatePublicHTTPURL(parsedURL); outboundErr != nil {
		return "", fmt.Errorf("%w: invalid target_url", ErrInvalidSiteHealthMonitor)
	}
	return parsedURL.String(), nil
}

// RecipientModeValue returns the normalized monitor recipient mode.
func (monitor SiteHealthMonitor) RecipientModeValue() string {
	recipientMode, recipientModeErr := NormalizeSiteRecipientMode(monitor.RecipientMode)
	if recipientModeErr != nil {
		return SiteRecipientModeManager
	}
	return recipientMode
}

// SelectedRecipientEmails returns the normalized selected-member recipient list.
func (monitor SiteHealthMonitor) SelectedRecipientEmails() []string {
	return DecodeSiteRecipientEmails(monitor.RecipientEmails)
}

// CheckInterval returns the monitor check interval.
func (monitor SiteHealthMonitor) CheckInterval() time.Duration {
	intervalSeconds := monitor.IntervalSeconds
	if intervalSeconds <= 0 {
		intervalSeconds = DefaultSiteHealthIntervalSeconds
	}
	return time.Duration(intervalSeconds) * time.Second
}

// CheckTimeout returns the monitor check timeout.
func (monitor SiteHealthMonitor) CheckTimeout() time.Duration {
	timeoutSeconds := monitor.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = DefaultSiteHealthTimeoutSeconds
	}
	return time.Duration(timeoutSeconds) * time.Second
}

// TruncateSiteHealthErrorMessage bounds persisted provider/network error detail.
func TruncateSiteHealthErrorMessage(rawMessage string) string {
	message := strings.TrimSpace(rawMessage)
	if len(message) <= siteHealthErrorMessageMaxLength {
		return message
	}
	return message[:siteHealthErrorMessageMaxLength]
}

// TruncateSiteHealthErrorCode bounds persisted health error codes.
func TruncateSiteHealthErrorCode(rawCode string) string {
	code := strings.TrimSpace(rawCode)
	if len(code) <= siteHealthErrorCodeMaxLength {
		return code
	}
	return code[:siteHealthErrorCodeMaxLength]
}
