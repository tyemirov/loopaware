package model

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	SubscriberStatusPending      = "pending"
	SubscriberStatusConfirmed    = "confirmed"
	SubscriberStatusUnsubscribed = "unsubscribed"

	subscriberEmailMaxLength     = 320
	subscriberNameMaxLength      = 200
	subscriberSourceURLMaxLength = 500
	subscriberIPMaxLength        = 64
	subscriberUserAgentMaxLength = 400
	subscriberStatusMaxLength    = 16
)

var (
	ErrInvalidSubscriberSiteID  = errors.New("invalid_subscriber_site_id")
	ErrInvalidSubscriberEmail   = errors.New("invalid_subscriber_email")
	ErrInvalidSubscriberStatus  = errors.New("invalid_subscriber_status")
	ErrInvalidSubscriberContact = errors.New("invalid_subscriber_contact")
)

// Subscriber captures newsletter/announcement subscriptions for a site.
type Subscriber struct {
	ID             string `gorm:"primaryKey;size:36"`
	SiteID         string `gorm:"not null;size:36;uniqueIndex:idx_subscribers_site_email"`
	Email          string `gorm:"not null;size:320;uniqueIndex:idx_subscribers_site_email"`
	Name           string `gorm:"size:200"`
	SourceURL      string `gorm:"size:500"`
	IP             string `gorm:"size:64"`
	UserAgent      string `gorm:"size:400"`
	Status         string `gorm:"not null;size:16;index"`
	ConsentAt      time.Time
	ConfirmedAt    time.Time
	UnsubscribedAt time.Time
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

// SubscriberInput holds the raw values used to construct a Subscriber.
type SubscriberInput struct {
	SiteID         string
	Email          string
	Name           string
	SourceURL      string
	IP             string
	UserAgent      string
	Status         string
	ConsentAt      time.Time
	ConfirmedAt    time.Time
	UnsubscribedAt time.Time
}

// NewSubscriber constructs a Subscriber with validated, normalized fields.
func NewSubscriber(input SubscriberInput) (Subscriber, error) {
	siteID := strings.TrimSpace(input.SiteID)
	if siteID == "" {
		return Subscriber{}, ErrInvalidSubscriberSiteID
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	if err := validateSubscriberEmail(email); err != nil {
		return Subscriber{}, err
	}

	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = SubscriberStatusPending
	}
	if err := validateSubscriberStatus(status); err != nil {
		return Subscriber{}, err
	}

	name := strings.TrimSpace(input.Name)
	if len(name) > subscriberNameMaxLength {
		return Subscriber{}, fmt.Errorf("%w: name too long", ErrInvalidSubscriberContact)
	}

	sourceURL := strings.TrimSpace(input.SourceURL)
	if len(sourceURL) > subscriberSourceURLMaxLength {
		return Subscriber{}, fmt.Errorf("%w: source_url too long", ErrInvalidSubscriberContact)
	}

	ip := strings.TrimSpace(input.IP)
	if len(ip) > subscriberIPMaxLength {
		return Subscriber{}, fmt.Errorf("%w: ip too long", ErrInvalidSubscriberContact)
	}

	userAgent := strings.TrimSpace(input.UserAgent)
	if len(userAgent) > subscriberUserAgentMaxLength {
		return Subscriber{}, fmt.Errorf("%w: user_agent too long", ErrInvalidSubscriberContact)
	}

	return Subscriber{
		ID:             uuid.NewString(),
		SiteID:         siteID,
		Email:          email,
		Name:           name,
		SourceURL:      sourceURL,
		IP:             ip,
		UserAgent:      userAgent,
		Status:         status,
		ConsentAt:      input.ConsentAt,
		ConfirmedAt:    input.ConfirmedAt,
		UnsubscribedAt: input.UnsubscribedAt,
	}, nil
}

func validateSubscriberEmail(email string) error {
	if email == "" || len(email) > subscriberEmailMaxLength {
		return fmt.Errorf("%w: empty or too long", ErrInvalidSubscriberEmail)
	}
	_, parseErr := mail.ParseAddress(email)
	if parseErr != nil {
		return fmt.Errorf("%w: %v", ErrInvalidSubscriberEmail, parseErr)
	}
	return nil
}

func validateSubscriberStatus(status string) error {
	if len(status) > subscriberStatusMaxLength {
		return fmt.Errorf("%w: too long", ErrInvalidSubscriberStatus)
	}
	switch status {
	case SubscriberStatusPending, SubscriberStatusConfirmed, SubscriberStatusUnsubscribed:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidSubscriberStatus, status)
	}
}
