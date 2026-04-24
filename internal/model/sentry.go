package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	SentryIssueStatusUnresolved = "unresolved"
	SentryIssueStatusResolved   = "resolved"
	SentryIssueStatusIgnored    = "ignored"

	SentryLevelDebug   = "debug"
	SentryLevelInfo    = "info"
	SentryLevelWarning = "warning"
	SentryLevelError   = "error"
	SentryLevelFatal   = "fatal"

	sentrySiteIDMaxLength        = 36
	sentryEventIDMaxLength       = 100
	sentryGroupingKeyMaxLength   = 64
	sentryTitleMaxLength         = 500
	sentryIssueStatusMaxLength   = 16
	sentryLevelMaxLength         = 20
	sentryPlatformMaxLength      = 50
	sentryEnvironmentMaxLength   = 100
	sentryReleaseMaxLength       = 200
	sentryMessageMaxLength       = 4000
	sentryExceptionTypeMaxLength = 300
	sentryUserHashMaxLength      = 200
	sentryFrameValueMaxLength    = 500
	sentryFrameModuleMaxLength   = 300
	sentryJSONMaxLength          = 12000
)

var (
	ErrInvalidSentrySiteID       = errors.New("invalid_sentry_site_id")
	ErrInvalidSentryEventID      = errors.New("invalid_sentry_event_id")
	ErrInvalidSentryTimestamp    = errors.New("invalid_sentry_timestamp")
	ErrInvalidSentryLevel        = errors.New("invalid_sentry_level")
	ErrInvalidSentryPlatform     = errors.New("invalid_sentry_platform")
	ErrInvalidSentryEnvironment  = errors.New("invalid_sentry_environment")
	ErrInvalidSentryMessage      = errors.New("invalid_sentry_message")
	ErrInvalidSentryFrame        = errors.New("invalid_sentry_frame")
	ErrInvalidSentryIssueStatus  = errors.New("invalid_sentry_issue_status")
	ErrInvalidSentryGrouping     = errors.New("invalid_sentry_grouping")
	ErrInvalidSentryOccurrenceID = errors.New("invalid_sentry_occurrence_id")
)

// SentryIssue groups developer error occurrences for a site.
type SentryIssue struct {
	ID              string `gorm:"primaryKey;size:36"`
	SiteID          string `gorm:"not null;size:36;uniqueIndex:idx_sentry_issues_site_group"`
	GroupingKey     string `gorm:"not null;size:64;uniqueIndex:idx_sentry_issues_site_group"`
	Title           string `gorm:"not null;size:500"`
	Status          string `gorm:"not null;size:16;index"`
	Level           string `gorm:"not null;size:20"`
	Platform        string `gorm:"not null;size:50"`
	Environment     string `gorm:"not null;size:100;index"`
	Release         string `gorm:"size:200"`
	FirstSeenAt     time.Time
	LastSeenAt      time.Time `gorm:"index"`
	OccurrenceCount int64     `gorm:"not null;default:1"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

// SentryOccurrence stores a raw developer error event.
type SentryOccurrence struct {
	ID            string    `gorm:"primaryKey;size:36"`
	SiteID        string    `gorm:"not null;size:36;index;uniqueIndex:idx_sentry_occurrences_site_event"`
	IssueID       string    `gorm:"not null;size:36;index"`
	EventID       string    `gorm:"not null;size:100;uniqueIndex:idx_sentry_occurrences_site_event"`
	RawMessage    string    `gorm:"not null;size:4000"`
	ExceptionType string    `gorm:"size:300"`
	StackFrames   string    `gorm:"type:text"`
	Request       string    `gorm:"type:text"`
	UserHash      string    `gorm:"size:200"`
	Tags          string    `gorm:"type:text"`
	Extra         string    `gorm:"type:text"`
	Platform      string    `gorm:"not null;size:50"`
	Environment   string    `gorm:"not null;size:100;index"`
	Release       string    `gorm:"size:200"`
	Level         string    `gorm:"not null;size:20"`
	ReceivedAt    time.Time `gorm:"not null;index"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

// SentryStackFrame describes one normalized stack frame.
type SentryStackFrame struct {
	Filename string `json:"filename"`
	Function string `json:"function"`
	Module   string `json:"module"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	InApp    bool   `json:"in_app"`
}

// SentryStackFrameInput captures raw stack frame values.
type SentryStackFrameInput struct {
	Filename string
	Function string
	Module   string
	Line     int
	Column   int
	InApp    bool
}

// SentryEventInput captures raw values for a Sentry developer error event.
type SentryEventInput struct {
	SiteID        string
	EventID       string
	OccurredAt    time.Time
	Platform      string
	Environment   string
	Release       string
	Level         string
	Message       string
	ExceptionType string
	StackFrames   []SentryStackFrameInput
	RequestJSON   string
	UserHash      string
	TagsJSON      string
	ExtraJSON     string
}

// SentryEvent combines a normalized occurrence with its grouping metadata.
type SentryEvent struct {
	Occurrence  SentryOccurrence
	GroupingKey string
	Title       string
}

// SentryIssueInput captures values required to create a grouped issue.
type SentryIssueInput struct {
	SiteID      string
	GroupingKey string
	Title       string
	Level       string
	Platform    string
	Environment string
	Release     string
	FirstSeenAt time.Time
	LastSeenAt  time.Time
}

// NewSentryEvent validates and normalizes a Sentry developer error event.
func NewSentryEvent(input SentryEventInput) (SentryEvent, error) {
	siteID := strings.TrimSpace(input.SiteID)
	if siteID == "" || len(siteID) > sentrySiteIDMaxLength {
		return SentryEvent{}, ErrInvalidSentrySiteID
	}

	eventID := strings.TrimSpace(input.EventID)
	if eventID == "" || len(eventID) > sentryEventIDMaxLength {
		return SentryEvent{}, ErrInvalidSentryEventID
	}

	if input.OccurredAt.IsZero() {
		return SentryEvent{}, ErrInvalidSentryTimestamp
	}
	occurredAt := input.OccurredAt.UTC()

	level, levelErr := NormalizeSentryLevel(input.Level)
	if levelErr != nil {
		return SentryEvent{}, levelErr
	}

	platform := strings.ToLower(strings.TrimSpace(input.Platform))
	if platform == "" || len(platform) > sentryPlatformMaxLength {
		return SentryEvent{}, ErrInvalidSentryPlatform
	}

	environment := strings.TrimSpace(input.Environment)
	if environment == "" || len(environment) > sentryEnvironmentMaxLength {
		return SentryEvent{}, ErrInvalidSentryEnvironment
	}

	message := strings.TrimSpace(input.Message)
	exceptionType := truncateString(strings.TrimSpace(input.ExceptionType), sentryExceptionTypeMaxLength)
	if message == "" && exceptionType == "" {
		return SentryEvent{}, ErrInvalidSentryMessage
	}
	message = truncateString(message, sentryMessageMaxLength)

	frames, framesErr := NewSentryStackFrames(input.StackFrames)
	if framesErr != nil {
		return SentryEvent{}, framesErr
	}
	framesJSON, framesJSONErr := marshalLimitedSentryJSON(frames)
	if framesJSONErr != nil {
		return SentryEvent{}, framesJSONErr
	}

	title := buildSentryTitle(exceptionType, message)
	groupingKey := buildSentryGroupingKey(exceptionType, message, frames)
	if groupingKey == "" {
		return SentryEvent{}, ErrInvalidSentryGrouping
	}

	occurrence := SentryOccurrence{
		ID:            uuid.NewString(),
		SiteID:        siteID,
		EventID:       eventID,
		RawMessage:    message,
		ExceptionType: exceptionType,
		StackFrames:   framesJSON,
		Request:       truncateString(strings.TrimSpace(input.RequestJSON), sentryJSONMaxLength),
		UserHash:      truncateString(strings.TrimSpace(input.UserHash), sentryUserHashMaxLength),
		Tags:          truncateString(strings.TrimSpace(input.TagsJSON), sentryJSONMaxLength),
		Extra:         truncateString(strings.TrimSpace(input.ExtraJSON), sentryJSONMaxLength),
		Platform:      platform,
		Environment:   environment,
		Release:       truncateString(strings.TrimSpace(input.Release), sentryReleaseMaxLength),
		Level:         level,
		ReceivedAt:    occurredAt,
	}

	return SentryEvent{
		Occurrence:  occurrence,
		GroupingKey: groupingKey,
		Title:       title,
	}, nil
}

// NewSentryIssue creates a grouped developer issue from validated event metadata.
func NewSentryIssue(input SentryIssueInput) (SentryIssue, error) {
	siteID := strings.TrimSpace(input.SiteID)
	if siteID == "" || len(siteID) > sentrySiteIDMaxLength {
		return SentryIssue{}, ErrInvalidSentrySiteID
	}
	groupingKey := strings.TrimSpace(input.GroupingKey)
	if groupingKey == "" || len(groupingKey) > sentryGroupingKeyMaxLength {
		return SentryIssue{}, ErrInvalidSentryGrouping
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return SentryIssue{}, ErrInvalidSentryMessage
	}
	level, levelErr := NormalizeSentryLevel(input.Level)
	if levelErr != nil {
		return SentryIssue{}, levelErr
	}
	platform := strings.TrimSpace(input.Platform)
	if platform == "" || len(platform) > sentryPlatformMaxLength {
		return SentryIssue{}, ErrInvalidSentryPlatform
	}
	environment := strings.TrimSpace(input.Environment)
	if environment == "" || len(environment) > sentryEnvironmentMaxLength {
		return SentryIssue{}, ErrInvalidSentryEnvironment
	}
	firstSeenAt := input.FirstSeenAt.UTC()
	lastSeenAt := input.LastSeenAt.UTC()
	if firstSeenAt.IsZero() || lastSeenAt.IsZero() {
		return SentryIssue{}, ErrInvalidSentryTimestamp
	}

	return SentryIssue{
		ID:              uuid.NewString(),
		SiteID:          siteID,
		GroupingKey:     groupingKey,
		Title:           truncateString(title, sentryTitleMaxLength),
		Status:          SentryIssueStatusUnresolved,
		Level:           level,
		Platform:        strings.ToLower(platform),
		Environment:     environment,
		Release:         truncateString(strings.TrimSpace(input.Release), sentryReleaseMaxLength),
		FirstSeenAt:     firstSeenAt,
		LastSeenAt:      lastSeenAt,
		OccurrenceCount: 1,
	}, nil
}

// NormalizeSentryLevel validates a Sentry severity level.
func NormalizeSentryLevel(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" || len(normalized) > sentryLevelMaxLength {
		return "", ErrInvalidSentryLevel
	}
	switch normalized {
	case SentryLevelDebug, SentryLevelInfo, SentryLevelWarning, SentryLevelError, SentryLevelFatal:
		return normalized, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidSentryLevel, normalized)
	}
}

// NormalizeSentryIssueStatus validates a grouped issue status.
func NormalizeSentryIssueStatus(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" || len(normalized) > sentryIssueStatusMaxLength {
		return "", ErrInvalidSentryIssueStatus
	}
	switch normalized {
	case SentryIssueStatusUnresolved, SentryIssueStatusResolved, SentryIssueStatusIgnored:
		return normalized, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidSentryIssueStatus, normalized)
	}
}

// NewSentryStackFrames validates and normalizes stack frames.
func NewSentryStackFrames(inputs []SentryStackFrameInput) ([]SentryStackFrame, error) {
	frames := make([]SentryStackFrame, 0, len(inputs))
	for _, frameInput := range inputs {
		filename := truncateString(strings.TrimSpace(frameInput.Filename), sentryFrameValueMaxLength)
		functionName := truncateString(strings.TrimSpace(frameInput.Function), sentryFrameValueMaxLength)
		moduleName := truncateString(strings.TrimSpace(frameInput.Module), sentryFrameModuleMaxLength)
		if filename == "" && functionName == "" && moduleName == "" {
			return nil, ErrInvalidSentryFrame
		}
		if frameInput.Line < 0 || frameInput.Column < 0 {
			return nil, ErrInvalidSentryFrame
		}
		frames = append(frames, SentryStackFrame{
			Filename: filename,
			Function: functionName,
			Module:   moduleName,
			Line:     frameInput.Line,
			Column:   frameInput.Column,
			InApp:    frameInput.InApp,
		})
	}
	return frames, nil
}

func marshalLimitedSentryJSON(value any) (string, error) {
	serialized, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		return "", marshalErr
	}
	return truncateString(string(serialized), sentryJSONMaxLength), nil
}

func buildSentryTitle(exceptionType string, message string) string {
	normalizedException := strings.TrimSpace(exceptionType)
	firstLine := firstSentryMessageLine(message)
	if normalizedException != "" && firstLine != "" {
		return truncateString(normalizedException+": "+firstLine, sentryTitleMaxLength)
	}
	if normalizedException != "" {
		return truncateString(normalizedException, sentryTitleMaxLength)
	}
	return truncateString(firstLine, sentryTitleMaxLength)
}

func firstSentryMessageLine(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ""
	}
	lines := strings.Split(trimmed, "\n")
	return strings.TrimSpace(lines[0])
}

func buildSentryGroupingKey(exceptionType string, message string, frames []SentryStackFrame) string {
	topFrame := resolveSentryTopFrame(frames)
	groupParts := []string{
		strings.ToLower(strings.TrimSpace(exceptionType)),
		strings.ToLower(firstSentryMessageLine(message)),
		strings.ToLower(topFrame.Filename),
		strings.ToLower(topFrame.Function),
		strconv.Itoa(topFrame.Line),
	}
	hash := sha256.Sum256([]byte(strings.Join(groupParts, "\x00")))
	return hex.EncodeToString(hash[:])
}

func resolveSentryTopFrame(frames []SentryStackFrame) SentryStackFrame {
	for frameIndex := len(frames) - 1; frameIndex >= 0; frameIndex -= 1 {
		if frames[frameIndex].InApp {
			return frames[frameIndex]
		}
	}
	if len(frames) > 0 {
		return frames[len(frames)-1]
	}
	return SentryStackFrame{}
}
