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

	visitURLMaxLength       = 500
	visitPathMaxLength      = 300
	visitIPMaxLength        = 64
	visitUserAgentMaxLength = 400
)

var (
	ErrInvalidVisitSiteID = errors.New("invalid_visit_site_id")
	ErrInvalidVisitURL    = errors.New("invalid_visit_url")
	ErrInvalidVisitID     = errors.New("invalid_visit_id")
)

// SiteVisit captures a single page view.
type SiteVisit struct {
	ID         string    `gorm:"primaryKey;size:36"`
	SiteID     string    `gorm:"not null;size:36;index"`
	URL        string    `gorm:"size:500"`
	Path       string    `gorm:"size:300;index"`
	VisitorID  string    `gorm:"size:36;index"`
	IP         string    `gorm:"size:64"`
	UserAgent  string    `gorm:"size:400"`
	Referrer   string    `gorm:"size:500"`
	IsBot      bool      `gorm:"not null;default:false;index"`
	Status     string    `gorm:"size:20"`
	OccurredAt time.Time `gorm:"not null;index"`
}

// SiteVisitInput holds incoming visit data.
type SiteVisitInput struct {
	SiteID    string
	URL       string
	VisitorID string
	IP        string
	UserAgent string
	Referrer  string
	IsBot     bool
	Occurred  time.Time
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

	return SiteVisit{
		ID:         uuid.NewString(),
		SiteID:     siteID,
		URL:        normalizedURL,
		Path:       path,
		VisitorID:  visitorID,
		IP:         ip,
		UserAgent:  userAgent,
		Referrer:   referrer,
		IsBot:      input.IsBot,
		Status:     VisitStatusRecorded,
		OccurredAt: occurred,
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
