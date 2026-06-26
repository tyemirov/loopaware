package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/google/uuid"
)

const (
	TrafficReportFrequencyDaily   = "daily"
	TrafficReportFrequencyWeekly  = "weekly"
	TrafficReportFrequencyMonthly = "monthly"

	TrafficReportRecipientModeManager  = SiteRecipientModeManager
	TrafficReportRecipientModeTeam     = SiteRecipientModeTeam
	TrafficReportRecipientModeSelected = SiteRecipientModeSelected

	TrafficReportStatusPending = "pending"
	TrafficReportStatusSent    = "sent"
	TrafficReportStatusFailed  = "failed"

	DefaultTrafficReportTimezone   = "UTC"
	DefaultTrafficReportSendHour   = 9
	DefaultTrafficReportSendMinute = 0
	DefaultTrafficReportWeekday    = int(time.Monday)
	DefaultTrafficReportMonthDay   = 1

	trafficReportTimezoneMaxLength = 100
)

var ErrInvalidTrafficReportSchedule = errors.New("invalid_traffic_report_schedule")

// TrafficReportSchedule captures one site's recurring traffic report settings.
type TrafficReportSchedule struct {
	ID                string    `gorm:"primaryKey;size:36"`
	SiteID            string    `gorm:"not null;size:36;uniqueIndex"`
	Enabled           bool      `gorm:"not null;default:false;index"`
	Frequency         string    `gorm:"not null;size:16"`
	RecipientEmail    string    `gorm:"not null;size:320"`
	RecipientMode     string    `gorm:"not null;size:16;default:manager"`
	RecipientEmails   string    `gorm:"type:text"`
	Timezone          string    `gorm:"not null;size:100"`
	SendHour          int       `gorm:"not null"`
	SendMinute        int       `gorm:"not null"`
	Weekday           int       `gorm:"not null"`
	MonthDay          int       `gorm:"not null"`
	NextSendAt        time.Time `gorm:"index"`
	LastSentAt        time.Time
	LastAttemptedAt   time.Time `gorm:"index"`
	RetryCount        int       `gorm:"not null;default:0;index"`
	LastStatus        string    `gorm:"not null;size:32;default:pending"`
	LastError         string    `gorm:"size:500"`
	ProviderMessageID string    `gorm:"size:120"`
	CreatedAt         time.Time `gorm:"autoCreateTime"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime"`
}

// PortfolioTrafficReportSchedule captures recurring cross-site reports for one user.
type PortfolioTrafficReportSchedule struct {
	ID                string    `gorm:"primaryKey;size:36"`
	UserEmail         string    `gorm:"not null;size:320;index;uniqueIndex:idx_portfolio_report_schedule_user_report,priority:1"`
	ReportID          string    `gorm:"not null;size:36;default:all-sites-traffic;uniqueIndex:idx_portfolio_report_schedule_user_report,priority:2"`
	Enabled           bool      `gorm:"not null;default:false;index"`
	Frequency         string    `gorm:"not null;size:16"`
	RecipientEmail    string    `gorm:"not null;size:320"`
	Timezone          string    `gorm:"not null;size:100"`
	SendHour          int       `gorm:"not null"`
	SendMinute        int       `gorm:"not null"`
	Weekday           int       `gorm:"not null"`
	MonthDay          int       `gorm:"not null"`
	NextSendAt        time.Time `gorm:"index"`
	LastSentAt        time.Time
	LastAttemptedAt   time.Time `gorm:"index"`
	RetryCount        int       `gorm:"not null;default:0;index"`
	LastStatus        string    `gorm:"not null;size:32;default:pending"`
	LastError         string    `gorm:"size:500"`
	ProviderMessageID string    `gorm:"size:120"`
	CreatedAt         time.Time `gorm:"autoCreateTime"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime"`
}

// TrafficReportScheduleInput holds incoming schedule values from API/configuration edges.
type TrafficReportScheduleInput struct {
	SiteID          string
	Enabled         bool
	Frequency       string
	RecipientEmail  string
	RecipientMode   string
	RecipientEmails []string
	Timezone        string
	SendHour        int
	SendMinute      int
	Weekday         int
	MonthDay        int
	ReferenceTime   time.Time
}

// PortfolioTrafficReportScheduleInput holds recurring all-sites schedule values.
type PortfolioTrafficReportScheduleInput struct {
	UserEmail      string
	ReportID       string
	Enabled        bool
	Frequency      string
	RecipientEmail string
	Timezone       string
	SendHour       int
	SendMinute     int
	Weekday        int
	MonthDay       int
	ReferenceTime  time.Time
}

// NewTrafficReportSchedule constructs a validated TrafficReportSchedule.
func NewTrafficReportSchedule(input TrafficReportScheduleInput) (TrafficReportSchedule, error) {
	siteID := strings.TrimSpace(input.SiteID)
	if siteID == "" {
		return TrafficReportSchedule{}, fmt.Errorf("%w: missing site_id", ErrInvalidTrafficReportSchedule)
	}

	frequency := normalizeTrafficReportFrequency(input.Frequency)
	if frequency == "" {
		return TrafficReportSchedule{}, fmt.Errorf("%w: invalid frequency", ErrInvalidTrafficReportSchedule)
	}

	recipientEmail, recipientErr := normalizeTrafficReportRecipient(input.RecipientEmail)
	if recipientErr != nil {
		return TrafficReportSchedule{}, recipientErr
	}

	recipientMode, recipientModeErr := normalizeTrafficReportRecipientMode(input.RecipientMode)
	if recipientModeErr != nil {
		return TrafficReportSchedule{}, recipientModeErr
	}
	recipientEmails, recipientEmailsErr := normalizeTrafficReportRecipientList(input.RecipientEmails)
	if recipientEmailsErr != nil {
		return TrafficReportSchedule{}, recipientEmailsErr
	}
	if recipientMode == TrafficReportRecipientModeSelected && len(recipientEmails) == 0 {
		return TrafficReportSchedule{}, fmt.Errorf("%w: missing recipient_emails", ErrInvalidTrafficReportSchedule)
	}
	if recipientMode != TrafficReportRecipientModeSelected {
		recipientEmails = []string{}
	}
	encodedRecipientEmails, encodeRecipientEmailsErr := encodeTrafficReportRecipientEmails(recipientEmails)
	if encodeRecipientEmailsErr != nil {
		return TrafficReportSchedule{}, encodeRecipientEmailsErr
	}

	timezoneName, _, timezoneErr := normalizeTrafficReportTimezone(input.Timezone)
	if timezoneErr != nil {
		return TrafficReportSchedule{}, timezoneErr
	}

	if input.SendHour < 0 || input.SendHour > 23 {
		return TrafficReportSchedule{}, fmt.Errorf("%w: invalid send_hour", ErrInvalidTrafficReportSchedule)
	}
	if input.SendMinute < 0 || input.SendMinute > 59 {
		return TrafficReportSchedule{}, fmt.Errorf("%w: invalid send_minute", ErrInvalidTrafficReportSchedule)
	}
	if input.Weekday < int(time.Sunday) || input.Weekday > int(time.Saturday) {
		return TrafficReportSchedule{}, fmt.Errorf("%w: invalid weekday", ErrInvalidTrafficReportSchedule)
	}
	if input.MonthDay < 1 || input.MonthDay > 28 {
		return TrafficReportSchedule{}, fmt.Errorf("%w: invalid month_day", ErrInvalidTrafficReportSchedule)
	}

	referenceTime := input.ReferenceTime
	if referenceTime.IsZero() {
		referenceTime = time.Now().UTC()
	}

	schedule := TrafficReportSchedule{
		ID:              uuid.NewString(),
		SiteID:          siteID,
		Enabled:         input.Enabled,
		Frequency:       frequency,
		RecipientEmail:  recipientEmail,
		RecipientMode:   recipientMode,
		RecipientEmails: encodedRecipientEmails,
		Timezone:        timezoneName,
		SendHour:        input.SendHour,
		SendMinute:      input.SendMinute,
		Weekday:         input.Weekday,
		MonthDay:        input.MonthDay,
		LastStatus:      TrafficReportStatusPending,
	}
	nextSendAt, nextErr := schedule.NextAfter(referenceTime)
	if nextErr != nil {
		return TrafficReportSchedule{}, nextErr
	}
	schedule.NextSendAt = nextSendAt
	return schedule, nil
}

// NewPortfolioTrafficReportSchedule constructs a validated portfolio report schedule.
func NewPortfolioTrafficReportSchedule(input PortfolioTrafficReportScheduleInput) (PortfolioTrafficReportSchedule, error) {
	userEmail, userEmailErr := normalizeTrafficReportRecipient(input.UserEmail)
	if userEmailErr != nil {
		return PortfolioTrafficReportSchedule{}, userEmailErr
	}

	frequency := normalizeTrafficReportFrequency(input.Frequency)
	if frequency == "" {
		return PortfolioTrafficReportSchedule{}, fmt.Errorf("%w: invalid frequency", ErrInvalidTrafficReportSchedule)
	}

	recipientEmail, recipientErr := normalizeTrafficReportRecipient(input.RecipientEmail)
	if recipientErr != nil {
		return PortfolioTrafficReportSchedule{}, recipientErr
	}

	timezoneName, _, timezoneErr := normalizeTrafficReportTimezone(input.Timezone)
	if timezoneErr != nil {
		return PortfolioTrafficReportSchedule{}, timezoneErr
	}

	if input.SendHour < 0 || input.SendHour > 23 {
		return PortfolioTrafficReportSchedule{}, fmt.Errorf("%w: invalid send_hour", ErrInvalidTrafficReportSchedule)
	}
	if input.SendMinute < 0 || input.SendMinute > 59 {
		return PortfolioTrafficReportSchedule{}, fmt.Errorf("%w: invalid send_minute", ErrInvalidTrafficReportSchedule)
	}
	if input.Weekday < int(time.Sunday) || input.Weekday > int(time.Saturday) {
		return PortfolioTrafficReportSchedule{}, fmt.Errorf("%w: invalid weekday", ErrInvalidTrafficReportSchedule)
	}
	if input.MonthDay < 1 || input.MonthDay > 28 {
		return PortfolioTrafficReportSchedule{}, fmt.Errorf("%w: invalid month_day", ErrInvalidTrafficReportSchedule)
	}

	referenceTime := input.ReferenceTime
	if referenceTime.IsZero() {
		referenceTime = time.Now().UTC()
	}

	schedule := PortfolioTrafficReportSchedule{
		ID:             uuid.NewString(),
		UserEmail:      userEmail,
		ReportID:       normalizePortfolioTrafficReportID(input.ReportID),
		Enabled:        input.Enabled,
		Frequency:      frequency,
		RecipientEmail: recipientEmail,
		Timezone:       timezoneName,
		SendHour:       input.SendHour,
		SendMinute:     input.SendMinute,
		Weekday:        input.Weekday,
		MonthDay:       input.MonthDay,
		LastStatus:     TrafficReportStatusPending,
	}
	nextSendAt, nextErr := schedule.NextAfter(referenceTime)
	if nextErr != nil {
		return PortfolioTrafficReportSchedule{}, nextErr
	}
	schedule.NextSendAt = nextSendAt
	return schedule, nil
}

func normalizePortfolioTrafficReportID(rawReportID string) string {
	reportID := strings.TrimSpace(rawReportID)
	if reportID == "" {
		return PortfolioTrafficReportDefaultID
	}
	return reportID
}

// NextAfter returns the next scheduled send time strictly after referenceTime.
func (schedule TrafficReportSchedule) NextAfter(referenceTime time.Time) (time.Time, error) {
	_, location, timezoneErr := normalizeTrafficReportTimezone(schedule.Timezone)
	if timezoneErr != nil {
		return time.Time{}, timezoneErr
	}
	if referenceTime.IsZero() {
		referenceTime = time.Now().UTC()
	}
	localReference := referenceTime.In(location)

	var candidate time.Time
	switch schedule.Frequency {
	case TrafficReportFrequencyDaily:
		candidate = time.Date(localReference.Year(), localReference.Month(), localReference.Day(), schedule.SendHour, schedule.SendMinute, 0, 0, location)
		if !candidate.After(localReference) {
			candidate = candidate.AddDate(0, 0, 1)
		}
	case TrafficReportFrequencyWeekly:
		candidate = time.Date(localReference.Year(), localReference.Month(), localReference.Day(), schedule.SendHour, schedule.SendMinute, 0, 0, location)
		daysUntil := (schedule.Weekday - int(localReference.Weekday()) + 7) % 7
		candidate = candidate.AddDate(0, 0, daysUntil)
		if !candidate.After(localReference) {
			candidate = candidate.AddDate(0, 0, 7)
		}
	case TrafficReportFrequencyMonthly:
		candidate = time.Date(localReference.Year(), localReference.Month(), schedule.MonthDay, schedule.SendHour, schedule.SendMinute, 0, 0, location)
		if !candidate.After(localReference) {
			candidate = candidate.AddDate(0, 1, 0)
		}
	default:
		return time.Time{}, fmt.Errorf("%w: invalid frequency", ErrInvalidTrafficReportSchedule)
	}

	return candidate.UTC(), nil
}

// NextAfter returns the next scheduled portfolio send time strictly after referenceTime.
func (schedule PortfolioTrafficReportSchedule) NextAfter(referenceTime time.Time) (time.Time, error) {
	siteSchedule := TrafficReportSchedule{
		Frequency:  schedule.Frequency,
		Timezone:   schedule.Timezone,
		SendHour:   schedule.SendHour,
		SendMinute: schedule.SendMinute,
		Weekday:    schedule.Weekday,
		MonthDay:   schedule.MonthDay,
	}
	return siteSchedule.NextAfter(referenceTime)
}

// ReportWindowDays returns the number of days summarized by the report frequency.
func (schedule TrafficReportSchedule) ReportWindowDays() int {
	switch schedule.Frequency {
	case TrafficReportFrequencyDaily:
		return 1
	case TrafficReportFrequencyWeekly:
		return 7
	case TrafficReportFrequencyMonthly:
		return 30
	default:
		return 1
	}
}

// ReportWindowDays returns the number of days summarized by the portfolio report frequency.
func (schedule PortfolioTrafficReportSchedule) ReportWindowDays() int {
	siteSchedule := TrafficReportSchedule{Frequency: schedule.Frequency}
	return siteSchedule.ReportWindowDays()
}

// SelectedRecipientEmails returns the normalized selected-member recipient list.
func (schedule TrafficReportSchedule) SelectedRecipientEmails() []string {
	return DecodeSiteRecipientEmails(schedule.RecipientEmails)
}

// RecipientModeValue returns the normalized schedule recipient mode.
func (schedule TrafficReportSchedule) RecipientModeValue() string {
	recipientMode, recipientModeErr := normalizeTrafficReportRecipientMode(schedule.RecipientMode)
	if recipientModeErr != nil {
		return TrafficReportRecipientModeManager
	}
	return recipientMode
}

func normalizeTrafficReportFrequency(rawFrequency string) string {
	switch strings.ToLower(strings.TrimSpace(rawFrequency)) {
	case TrafficReportFrequencyDaily:
		return TrafficReportFrequencyDaily
	case TrafficReportFrequencyWeekly:
		return TrafficReportFrequencyWeekly
	case TrafficReportFrequencyMonthly:
		return TrafficReportFrequencyMonthly
	default:
		return ""
	}
}

func normalizeTrafficReportRecipientMode(rawMode string) (string, error) {
	recipientMode, recipientModeErr := NormalizeSiteRecipientMode(rawMode)
	if recipientModeErr != nil {
		return "", fmt.Errorf("%w: invalid recipient_mode", ErrInvalidTrafficReportSchedule)
	}
	return recipientMode, nil
}

func normalizeTrafficReportRecipient(rawRecipient string) (string, error) {
	recipient, recipientErr := NormalizeSiteRecipientEmail(rawRecipient)
	if recipientErr != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidTrafficReportSchedule, recipientErr)
	}
	return recipient, nil
}

func normalizeTrafficReportRecipientList(rawRecipients []string) ([]string, error) {
	recipients, recipientErr := NormalizeSiteRecipientEmailList(rawRecipients)
	if recipientErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTrafficReportSchedule, recipientErr)
	}
	return recipients, nil
}

func encodeTrafficReportRecipientEmails(recipientEmails []string) (string, error) {
	encoded, encodeErr := EncodeSiteRecipientEmails(recipientEmails)
	if encodeErr != nil {
		return "", fmt.Errorf("%w: invalid recipient_emails", ErrInvalidTrafficReportSchedule)
	}
	return encoded, nil
}

func normalizeTrafficReportTimezone(rawTimezone string) (string, *time.Location, error) {
	timezoneName := strings.TrimSpace(rawTimezone)
	if timezoneName == "" {
		timezoneName = DefaultTrafficReportTimezone
	}
	if len(timezoneName) > trafficReportTimezoneMaxLength {
		return "", nil, fmt.Errorf("%w: timezone too long", ErrInvalidTrafficReportSchedule)
	}
	location, locationErr := time.LoadLocation(timezoneName)
	if locationErr != nil {
		return "", nil, fmt.Errorf("%w: invalid timezone", ErrInvalidTrafficReportSchedule)
	}
	return timezoneName, location, nil
}
