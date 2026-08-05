package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	sentryRouteErrors                   = "/sentry/errors"
	sentryRouteBrowserErrors            = "/sentry/browser-errors"
	sentryHeaderAuthorization           = "Authorization"
	sentryHeaderIngestToken             = "X-LoopAware-Sentry-Token"
	sentryAuthorizationBearerPrefix     = "Bearer "
	sentryTokenByteLength               = 32
	sentryTokenPrefix                   = "las_"
	sentryBrowserRateWindowSeconds      = 30
	sentryBrowserMaxRequestsPerWindow   = 60
	sentryBrowserMaxRateCounterEntries  = 4096
	sentryPersistMaxAttempts            = 6
	sentryPersistRetryDelayMilliseconds = 25
	sentryPlatformJavaScript            = "javascript"
	sentryAlertKindFirstSeen            = "first_seen"
	sentryAlertKindRegressed            = "regressed"
	errorValueOriginForbidden           = "origin_forbidden"
	errorValueRateLimited               = "rate_limited"
	errorValueMissingSentryToken        = "missing_sentry_token"
	errorValueInvalidSentryToken        = "invalid_sentry_token"
	errorValueSentryTokenNotConfigured  = "sentry_token_not_configured"
	errorValueInvalidSentryEvent        = "invalid_sentry_event"
	errorValueDuplicateSentryEvent      = "duplicate_sentry_event"
	errorValueInvalidSentryIssueStatus  = "invalid_sentry_issue_status"
	errorValueUnknownSentryIssue        = "unknown_sentry_issue"
	errorValueSentryTokenRotationFailed = "sentry_token_rotation_failed"
)

// SentryHandlers owns developer monitoring ingest and dashboard APIs.
type SentryHandlers struct {
	database          *gorm.DB
	logger            *zap.Logger
	emailSender       EmailSender
	publicBaseURL     string
	rateWindow        time.Duration
	rateCountersByKey map[string]sentryBrowserRateCounter
	rateCountersMutex sync.Mutex
}

type sentryBrowserRateCounter struct {
	windowStartedAt time.Time
	count           int
}

type sentryErrorRequest struct {
	SiteID        string                    `json:"site_id"`
	EventID       string                    `json:"event_id"`
	Timestamp     string                    `json:"timestamp"`
	Platform      string                    `json:"platform"`
	Environment   string                    `json:"environment"`
	Release       string                    `json:"release"`
	Level         string                    `json:"level"`
	Message       string                    `json:"message"`
	ExceptionType string                    `json:"exception_type"`
	Stacktrace    []sentryStackFrameRequest `json:"stacktrace"`
	Request       map[string]any            `json:"request"`
	UserHash      string                    `json:"user_hash"`
	Tags          map[string]string         `json:"tags"`
	Extra         map[string]any            `json:"extra"`
}

type sentryStackFrameRequest struct {
	Filename string `json:"filename"`
	Function string `json:"function"`
	Module   string `json:"module"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	InApp    bool   `json:"in_app"`
}

type sentryCaptureResponse struct {
	Status       string `json:"status"`
	IssueID      string `json:"issue_id"`
	OccurrenceID string `json:"occurrence_id"`
	GroupingKey  string `json:"grouping_key"`
	Duplicate    bool   `json:"duplicate"`
}

type sentryIssuesResponse struct {
	SiteID string              `json:"site_id"`
	Issues []sentryIssueRecord `json:"issues"`
}

type sentryIssueRecord struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	Level           string `json:"level"`
	Platform        string `json:"platform"`
	Environment     string `json:"environment"`
	Release         string `json:"release"`
	FirstSeenAt     int64  `json:"first_seen_at"`
	LastSeenAt      int64  `json:"last_seen_at"`
	OccurrenceCount int64  `json:"occurrence_count"`
}

type sentryIssueDetailResponse struct {
	SiteID            string                   `json:"site_id"`
	Issue             sentryIssueRecord        `json:"issue"`
	LatestOccurrence  sentryOccurrenceRecord   `json:"latest_occurrence"`
	RecentOccurrences []sentryOccurrenceRecord `json:"recent_occurrences"`
}

type sentryOccurrenceRecord struct {
	ID            string                   `json:"id"`
	EventID       string                   `json:"event_id"`
	Message       string                   `json:"message"`
	ExceptionType string                   `json:"exception_type"`
	Stacktrace    []model.SentryStackFrame `json:"stacktrace"`
	Request       any                      `json:"request"`
	UserHash      string                   `json:"user_hash"`
	Tags          any                      `json:"tags"`
	Extra         any                      `json:"extra"`
	Platform      string                   `json:"platform"`
	Environment   string                   `json:"environment"`
	Release       string                   `json:"release"`
	Level         string                   `json:"level"`
	ReceivedAt    int64                    `json:"received_at"`
}

type sentryIssueStatusRequest struct {
	Status string `json:"status"`
}

type sentryTokenResponse struct {
	SiteID          string `json:"site_id"`
	IngestEndpoint  string `json:"ingest_endpoint"`
	IngestToken     string `json:"ingest_token"`
	TokenConfigured bool   `json:"token_configured"`
}

// NewSentryHandlers builds Sentry developer monitoring handlers.
func NewSentryHandlers(database *gorm.DB, logger *zap.Logger, emailSender EmailSender, publicBaseURL string) *SentryHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SentryHandlers{
		database:          database,
		logger:            logger,
		emailSender:       emailSender,
		publicBaseURL:     strings.TrimRight(strings.TrimSpace(publicBaseURL), "/"),
		rateWindow:        sentryBrowserRateWindowSeconds * time.Second,
		rateCountersByKey: make(map[string]sentryBrowserRateCounter),
	}
}

// CaptureError accepts protected developer error events.
func (handlers *SentryHandlers) CaptureError(context *gin.Context) {
	var payload sentryErrorRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}

	var site model.Site
	siteIdentifier := strings.TrimSpace(payload.SiteID)
	if siteIdentifier == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingSite})
		return
	}
	if err := handlers.database.First(&site, "id = ?", siteIdentifier).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSite})
		return
	}

	if tokenErr := handlers.validateSentryToken(context, site); tokenErr != nil {
		statusCode, errorCode := sentryTokenErrorResponse(tokenErr)
		context.JSON(statusCode, gin.H{jsonKeyError: errorCode})
		return
	}

	handlers.captureSentryEvent(context, site, payload)
}

// CaptureBrowserError accepts browser developer error events from configured site origins.
func (handlers *SentryHandlers) CaptureBrowserError(context *gin.Context) {
	var payload sentryErrorRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}

	var site model.Site
	siteIdentifier := strings.TrimSpace(payload.SiteID)
	if siteIdentifier == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingSite})
		return
	}
	if err := handlers.database.First(&site, "id = ?", siteIdentifier).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSite})
		return
	}

	originHeader := strings.TrimSpace(context.GetHeader("Origin"))
	refererHeader := strings.TrimSpace(context.GetHeader("Referer"))
	if !isOriginAllowed(site.AllowedOrigin, originHeader, refererHeader, "") {
		context.JSON(http.StatusForbidden, gin.H{jsonKeyError: errorValueOriginForbidden})
		return
	}
	if handlers.isBrowserRateLimited(publicRateKey(sentryRouteBrowserErrors, site.ID, context.ClientIP())) {
		context.JSON(http.StatusTooManyRequests, gin.H{jsonKeyError: errorValueRateLimited})
		return
	}

	payload.Platform = sentryPlatformJavaScript
	payload.Request = minimizeBrowserSentryRequest(payload.Request)
	handlers.captureSentryEvent(context, site, payload)
}

func (handlers *SentryHandlers) captureSentryEvent(context *gin.Context, site model.Site, payload sentryErrorRequest) {
	occurredAt, timestampErr := parseSentryTimestamp(payload.Timestamp)
	if timestampErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidSentryEvent})
		return
	}

	requestJSON, requestJSONErr := marshalSentryPayloadJSON(payload.Request)
	tagsJSON, tagsJSONErr := marshalSentryPayloadJSON(payload.Tags)
	extraJSON, extraJSONErr := marshalSentryPayloadJSON(payload.Extra)
	if errors.Join(requestJSONErr, tagsJSONErr, extraJSONErr) != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidSentryEvent})
		return
	}

	event, eventErr := model.NewSentryEvent(model.SentryEventInput{
		SiteID:        site.ID,
		EventID:       payload.EventID,
		OccurredAt:    occurredAt,
		Platform:      payload.Platform,
		Environment:   payload.Environment,
		Release:       payload.Release,
		Level:         payload.Level,
		Message:       payload.Message,
		ExceptionType: payload.ExceptionType,
		StackFrames:   toSentryStackFrameInputs(payload.Stacktrace),
		RequestJSON:   requestJSON,
		UserHash:      payload.UserHash,
		TagsJSON:      tagsJSON,
		ExtraJSON:     extraJSON,
	})
	if eventErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidSentryEvent})
		return
	}

	issue, occurrence, duplicate, alertKind, persistErr := handlers.persistSentryEvent(context.Request.Context(), site, event)
	if persistErr != nil {
		handlers.logger.Warn("sentry_event_persist_failed", zap.Error(persistErr), zap.String("site_id", site.ID))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}

	if alertKind != "" {
		handlers.sendSentryAlert(context.Request.Context(), site, issue, occurrence, alertKind)
	}

	context.JSON(http.StatusOK, sentryCaptureResponse{
		Status:       "ok",
		IssueID:      issue.ID,
		OccurrenceID: occurrence.ID,
		GroupingKey:  issue.GroupingKey,
		Duplicate:    duplicate,
	})
}

// ListIssues returns grouped Sentry issues for an authorized site.
func (handlers *SentryHandlers) ListIssues(context *gin.Context) {
	site, ok := handlers.resolveAuthorizedSite(context)
	if !ok {
		return
	}

	var issues []model.SentryIssue
	if err := handlers.database.WithContext(context.Request.Context()).
		Where("site_id = ?", site.ID).
		Order("last_seen_at desc").
		Find(&issues).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}

	records := make([]sentryIssueRecord, 0, len(issues))
	for _, issue := range issues {
		records = append(records, toSentryIssueRecord(issue))
	}
	context.JSON(http.StatusOK, sentryIssuesResponse{SiteID: site.ID, Issues: records})
}

// IssueDetail returns the latest and recent occurrences for a grouped Sentry issue.
func (handlers *SentryHandlers) IssueDetail(context *gin.Context) {
	site, ok := handlers.resolveAuthorizedSite(context)
	if !ok {
		return
	}
	issue, issueOK := handlers.resolveSentryIssue(context, site.ID)
	if !issueOK {
		return
	}

	var occurrences []model.SentryOccurrence
	if err := handlers.database.WithContext(context.Request.Context()).
		Where("site_id = ? AND issue_id = ?", site.ID, issue.ID).
		Order("received_at desc").
		Limit(10).
		Find(&occurrences).Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}

	records := make([]sentryOccurrenceRecord, 0, len(occurrences))
	for _, occurrence := range occurrences {
		records = append(records, toSentryOccurrenceRecord(occurrence))
	}
	latest := sentryOccurrenceRecord{}
	if len(records) > 0 {
		latest = records[0]
	}
	context.JSON(http.StatusOK, sentryIssueDetailResponse{
		SiteID:            site.ID,
		Issue:             toSentryIssueRecord(issue),
		LatestOccurrence:  latest,
		RecentOccurrences: records,
	})
}

// UpdateIssueStatus changes a grouped Sentry issue status.
func (handlers *SentryHandlers) UpdateIssueStatus(context *gin.Context) {
	site, ok := handlers.resolveManagedSite(context)
	if !ok {
		return
	}
	issue, issueOK := handlers.resolveSentryIssue(context, site.ID)
	if !issueOK {
		return
	}

	var payload sentryIssueStatusRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}
	status, statusErr := model.NormalizeSentryIssueStatus(payload.Status)
	if statusErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidSentryIssueStatus})
		return
	}

	issue.Status = status
	if saveErr := handlers.database.WithContext(context.Request.Context()).Save(&issue).Error; saveErr != nil {
		handlers.logger.Warn("sentry_issue_status_save_failed", zap.Error(saveErr), zap.String("site_id", site.ID), zap.String("issue_id", issue.ID))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}

	context.JSON(http.StatusOK, toSentryIssueRecord(issue))
}

// RotateToken creates a new per-site Sentry ingest token and returns it once.
func (handlers *SentryHandlers) RotateToken(context *gin.Context) {
	site, ok := handlers.resolveManagedSite(context)
	if !ok {
		return
	}

	token, tokenHash, tokenErr := newSentryIngestToken()
	if tokenErr != nil {
		handlers.logger.Warn("sentry_token_generate_failed", zap.Error(tokenErr), zap.String("site_id", site.ID))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSentryTokenRotationFailed})
		return
	}

	if updateErr := handlers.database.WithContext(context.Request.Context()).Model(&model.Site{}).Where("id = ?", site.ID).Update("sentry_ingest_token_hash", tokenHash).Error; updateErr != nil {
		handlers.logger.Warn("sentry_token_rotate_failed", zap.Error(updateErr), zap.String("site_id", site.ID))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSentryTokenRotationFailed})
		return
	}

	context.JSON(http.StatusOK, sentryTokenResponse{
		SiteID:          site.ID,
		IngestEndpoint:  handlers.sentryIngestEndpoint(context),
		IngestToken:     token,
		TokenConfigured: true,
	})
}

func (handlers *SentryHandlers) persistSentryEvent(ctx context.Context, site model.Site, event model.SentryEvent) (model.SentryIssue, model.SentryOccurrence, bool, string, error) {
	for attempt := 1; attempt <= sentryPersistMaxAttempts; attempt += 1 {
		issue, occurrence, duplicate, alertKind, persistErr := handlers.persistSentryEventOnce(ctx, site, event)
		if persistErr == nil {
			return issue, occurrence, duplicate, alertKind, nil
		}
		if !isRetryableSentryPersistError(persistErr) || attempt == sentryPersistMaxAttempts {
			return model.SentryIssue{}, model.SentryOccurrence{}, false, "", persistErr
		}
		retryDelay := time.Duration(attempt*sentryPersistRetryDelayMilliseconds) * time.Millisecond
		retryTimer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			retryTimer.Stop()
			return model.SentryIssue{}, model.SentryOccurrence{}, false, "", ctx.Err()
		case <-retryTimer.C:
		}
	}
	return model.SentryIssue{}, model.SentryOccurrence{}, false, "", errors.New("sentry persist attempts exhausted")
}

func (handlers *SentryHandlers) persistSentryEventOnce(ctx context.Context, site model.Site, event model.SentryEvent) (model.SentryIssue, model.SentryOccurrence, bool, string, error) {
	var duplicateOccurrence model.SentryOccurrence
	duplicateErr := handlers.database.WithContext(ctx).
		Where("site_id = ? AND event_id = ?", site.ID, event.Occurrence.EventID).
		First(&duplicateOccurrence).Error
	if duplicateErr == nil {
		issue := model.SentryIssue{}
		if issueErr := handlers.database.WithContext(ctx).First(&issue, "id = ?", duplicateOccurrence.IssueID).Error; issueErr != nil {
			return model.SentryIssue{}, model.SentryOccurrence{}, true, "", issueErr
		}
		return issue, duplicateOccurrence, true, "", nil
	}
	if !errors.Is(duplicateErr, gorm.ErrRecordNotFound) {
		return model.SentryIssue{}, model.SentryOccurrence{}, false, "", duplicateErr
	}

	var savedIssue model.SentryIssue
	savedOccurrence := event.Occurrence
	alertKind := ""

	transactionErr := handlers.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		findErr := transaction.Where("site_id = ? AND grouping_key = ?", site.ID, event.GroupingKey).First(&savedIssue).Error
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			newIssue, issueErr := model.NewSentryIssue(model.SentryIssueInput{
				SiteID:      site.ID,
				GroupingKey: event.GroupingKey,
				Title:       event.Title,
				Level:       savedOccurrence.Level,
				Platform:    savedOccurrence.Platform,
				Environment: savedOccurrence.Environment,
				Release:     savedOccurrence.Release,
				FirstSeenAt: savedOccurrence.ReceivedAt,
				LastSeenAt:  savedOccurrence.ReceivedAt,
			})
			if issueErr != nil {
				return issueErr
			}
			createResult := transaction.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "site_id"}, {Name: "grouping_key"}},
				DoNothing: true,
			}).Create(&newIssue)
			if createResult.Error != nil {
				return createResult.Error
			}
			if createResult.RowsAffected == 0 {
				if refetchErr := transaction.Where("site_id = ? AND grouping_key = ?", site.ID, event.GroupingKey).First(&savedIssue).Error; refetchErr != nil {
					return refetchErr
				}
				if updateErr := updateExistingSentryIssue(transaction, &savedIssue, event, &alertKind); updateErr != nil {
					return updateErr
				}
			} else {
				savedIssue = newIssue
				alertKind = sentryAlertKindFirstSeen
			}
		} else if findErr != nil {
			return findErr
		} else {
			if updateErr := updateExistingSentryIssue(transaction, &savedIssue, event, &alertKind); updateErr != nil {
				return updateErr
			}
		}

		savedOccurrence.IssueID = savedIssue.ID
		if createErr := transaction.Create(&savedOccurrence).Error; createErr != nil {
			return createErr
		}
		return nil
	})
	if transactionErr != nil {
		return model.SentryIssue{}, model.SentryOccurrence{}, false, "", transactionErr
	}

	return savedIssue, savedOccurrence, false, alertKind, nil
}

func updateExistingSentryIssue(transaction *gorm.DB, savedIssue *model.SentryIssue, event model.SentryEvent, alertKind *string) error {
	savedOccurrence := event.Occurrence
	updates := map[string]any{
		"title":            event.Title,
		"level":            savedOccurrence.Level,
		"platform":         savedOccurrence.Platform,
		"environment":      savedOccurrence.Environment,
		"release":          savedOccurrence.Release,
		"last_seen_at":     savedOccurrence.ReceivedAt,
		"occurrence_count": gorm.Expr("occurrence_count + ?", 1),
	}
	if savedIssue.Status == model.SentryIssueStatusResolved {
		updates["status"] = model.SentryIssueStatusUnresolved
		*alertKind = sentryAlertKindRegressed
	}
	if updateErr := transaction.Model(savedIssue).Updates(updates).Error; updateErr != nil {
		return updateErr
	}
	return transaction.First(savedIssue, "id = ?", savedIssue.ID).Error
}

func isRetryableSentryPersistError(err error) bool {
	if err == nil {
		return false
	}
	normalizedMessage := strings.ToLower(err.Error())
	return strings.Contains(normalizedMessage, "database is locked") ||
		strings.Contains(normalizedMessage, "database table is locked") ||
		strings.Contains(normalizedMessage, "database is busy") ||
		strings.Contains(normalizedMessage, "database table is busy") ||
		strings.Contains(normalizedMessage, "sqlite_busy") ||
		strings.Contains(normalizedMessage, "sqlite_locked")
}

func (handlers *SentryHandlers) validateSentryToken(context *gin.Context, site model.Site) error {
	tokenHash := strings.TrimSpace(site.SentryIngestTokenHash)
	if tokenHash == "" {
		return errSentryTokenNotConfigured
	}
	token := readSentryIngestToken(context)
	if token == "" {
		return errMissingSentryToken
	}
	presentedHash := hashSentryIngestToken(token)
	if subtle.ConstantTimeCompare([]byte(presentedHash), []byte(tokenHash)) != 1 {
		return errInvalidSentryToken
	}
	return nil
}

var (
	errMissingSentryToken       = errors.New("missing sentry token")
	errInvalidSentryToken       = errors.New("invalid sentry token")
	errSentryTokenNotConfigured = errors.New("sentry token not configured")
)

func sentryTokenErrorResponse(err error) (int, string) {
	if errors.Is(err, errMissingSentryToken) {
		return http.StatusUnauthorized, errorValueMissingSentryToken
	}
	if errors.Is(err, errSentryTokenNotConfigured) {
		return http.StatusForbidden, errorValueSentryTokenNotConfigured
	}
	return http.StatusForbidden, errorValueInvalidSentryToken
}

func readSentryIngestToken(context *gin.Context) string {
	headerValue := strings.TrimSpace(context.GetHeader(sentryHeaderAuthorization))
	if strings.HasPrefix(headerValue, sentryAuthorizationBearerPrefix) {
		return strings.TrimSpace(strings.TrimPrefix(headerValue, sentryAuthorizationBearerPrefix))
	}
	return strings.TrimSpace(context.GetHeader(sentryHeaderIngestToken))
}

func newSentryIngestToken() (string, string, error) {
	tokenBytes := make([]byte, sentryTokenByteLength)
	if _, readErr := io.ReadFull(rand.Reader, tokenBytes); readErr != nil {
		return "", "", readErr
	}
	token := sentryTokenPrefix + base64.RawURLEncoding.EncodeToString(tokenBytes)
	return token, hashSentryIngestToken(token), nil
}

func hashSentryIngestToken(token string) string {
	hash := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(hash[:])
}

func parseSentryTimestamp(rawValue string) (time.Time, error) {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return time.Time{}, model.ErrInvalidSentryTimestamp
	}
	parsed, parseErr := time.Parse(time.RFC3339Nano, trimmed)
	if parseErr != nil {
		return time.Time{}, parseErr
	}
	return parsed.UTC(), nil
}

func marshalSentryPayloadJSON(value any) (string, error) {
	if value == nil {
		return "", nil
	}
	serialized, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		return "", marshalErr
	}
	return string(serialized), nil
}

func toSentryStackFrameInputs(frames []sentryStackFrameRequest) []model.SentryStackFrameInput {
	inputs := make([]model.SentryStackFrameInput, 0, len(frames))
	for _, frame := range frames {
		inputs = append(inputs, model.SentryStackFrameInput{
			Filename: frame.Filename,
			Function: frame.Function,
			Module:   frame.Module,
			Line:     frame.Line,
			Column:   frame.Column,
			InApp:    frame.InApp,
		})
	}
	return inputs
}

func minimizeBrowserSentryRequest(requestPayload map[string]any) map[string]any {
	if len(requestPayload) == 0 {
		return nil
	}
	minimized := make(map[string]any)
	for _, key := range []string{"url", "referrer", "user_agent"} {
		value, ok := requestPayload[key].(string)
		if !ok {
			continue
		}
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			continue
		}
		if key == "url" || key == "referrer" {
			trimmedValue = sanitizeBrowserSentryURL(trimmedValue)
		}
		if trimmedValue != "" {
			minimized[key] = trimmedValue
		}
	}
	if len(minimized) == 0 {
		return nil
	}
	return minimized
}

func sanitizeBrowserSentryURL(rawValue string) string {
	parsedURL, parseErr := url.Parse(strings.TrimSpace(rawValue))
	if parseErr != nil || parsedURL == nil {
		return ""
	}
	scheme := strings.ToLower(strings.TrimSpace(parsedURL.Scheme))
	if scheme != urlSchemeHTTP && scheme != urlSchemeHTTPS {
		return ""
	}
	if strings.TrimSpace(parsedURL.Host) == "" {
		return ""
	}
	return (&url.URL{Scheme: scheme, Host: parsedURL.Host, Path: parsedURL.Path}).String()
}

func (handlers *SentryHandlers) isBrowserRateLimited(key string) bool {
	now := time.Now()

	handlers.rateCountersMutex.Lock()
	defer handlers.rateCountersMutex.Unlock()

	handlers.pruneBrowserRateCounters(now)
	rateCounter, exists := handlers.rateCountersByKey[key]
	if !exists && len(handlers.rateCountersByKey) >= sentryBrowserMaxRateCounterEntries {
		return true
	}
	if !exists || now.Sub(rateCounter.windowStartedAt) >= handlers.rateWindow {
		rateCounter = sentryBrowserRateCounter{windowStartedAt: now}
	}
	rateCounter.count += 1
	handlers.rateCountersByKey[key] = rateCounter
	return rateCounter.count > sentryBrowserMaxRequestsPerWindow
}

func (handlers *SentryHandlers) pruneBrowserRateCounters(now time.Time) {
	for key, rateCounter := range handlers.rateCountersByKey {
		if now.Sub(rateCounter.windowStartedAt) >= handlers.rateWindow {
			delete(handlers.rateCountersByKey, key)
		}
	}
}

func (handlers *SentryHandlers) resolveAuthorizedSite(context *gin.Context) (model.Site, bool) {
	siteIdentifier := strings.TrimSpace(context.Param("id"))
	if siteIdentifier == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingSite})
		return model.Site{}, false
	}
	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return model.Site{}, false
	}
	var site model.Site
	if err := handlers.database.First(&site, "id = ?", siteIdentifier).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSite})
		return model.Site{}, false
	}
	if !currentUserCanViewSite(context.Request.Context(), handlers.database, currentUser, site) {
		context.JSON(http.StatusForbidden, gin.H{jsonKeyError: errorValueNotAuthorized})
		return model.Site{}, false
	}
	return site, true
}

func (handlers *SentryHandlers) resolveManagedSite(context *gin.Context) (model.Site, bool) {
	siteIdentifier := strings.TrimSpace(context.Param("id"))
	if siteIdentifier == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingSite})
		return model.Site{}, false
	}
	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return model.Site{}, false
	}
	var site model.Site
	if err := handlers.database.First(&site, "id = ?", siteIdentifier).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSite})
		return model.Site{}, false
	}
	if !currentUser.canManageSite(site) {
		context.JSON(http.StatusForbidden, gin.H{jsonKeyError: errorValueNotAuthorized})
		return model.Site{}, false
	}
	return site, true
}

func (handlers *SentryHandlers) resolveSentryIssue(context *gin.Context, siteID string) (model.SentryIssue, bool) {
	issueIdentifier := strings.TrimSpace(context.Param("issue_id"))
	if issueIdentifier == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueUnknownSentryIssue})
		return model.SentryIssue{}, false
	}
	var issue model.SentryIssue
	if err := handlers.database.First(&issue, "site_id = ? AND id = ?", siteID, issueIdentifier).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSentryIssue})
		return model.SentryIssue{}, false
	}
	return issue, true
}

func toSentryIssueRecord(issue model.SentryIssue) sentryIssueRecord {
	return sentryIssueRecord{
		ID:              issue.ID,
		Title:           issue.Title,
		Status:          issue.Status,
		Level:           issue.Level,
		Platform:        issue.Platform,
		Environment:     issue.Environment,
		Release:         issue.Release,
		FirstSeenAt:     unixSeconds(issue.FirstSeenAt),
		LastSeenAt:      unixSeconds(issue.LastSeenAt),
		OccurrenceCount: issue.OccurrenceCount,
	}
}

func toSentryOccurrenceRecord(occurrence model.SentryOccurrence) sentryOccurrenceRecord {
	var frames []model.SentryStackFrame
	decodeJSON(occurrence.StackFrames, &frames)
	return sentryOccurrenceRecord{
		ID:            occurrence.ID,
		EventID:       occurrence.EventID,
		Message:       occurrence.RawMessage,
		ExceptionType: occurrence.ExceptionType,
		Stacktrace:    frames,
		Request:       decodeJSONAny(occurrence.Request),
		UserHash:      occurrence.UserHash,
		Tags:          decodeJSONAny(occurrence.Tags),
		Extra:         decodeJSONAny(occurrence.Extra),
		Platform:      occurrence.Platform,
		Environment:   occurrence.Environment,
		Release:       occurrence.Release,
		Level:         occurrence.Level,
		ReceivedAt:    unixSeconds(occurrence.ReceivedAt),
	}
}

func decodeJSONAny(rawValue string) any {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return nil
	}
	var decoded any
	if decodeErr := json.Unmarshal([]byte(trimmed), &decoded); decodeErr != nil {
		return nil
	}
	return decoded
}

func decodeJSON(rawValue string, target any) {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return
	}
	_ = json.Unmarshal([]byte(trimmed), target)
}

func (handlers *SentryHandlers) sentryIngestEndpoint(context *gin.Context) string {
	baseURL := handlers.publicBaseURL
	if baseURL == "" {
		baseURL = resolveRequestOrigin(context, "")
	}
	if baseURL == "" {
		return sentryRouteErrors
	}
	return strings.TrimRight(baseURL, "/") + sentryRouteErrors
}

func (handlers *SentryHandlers) sendSentryAlert(ctx context.Context, site model.Site, issue model.SentryIssue, occurrence model.SentryOccurrence, alertKind string) {
	if handlers.emailSender == nil {
		return
	}
	recipient := strings.TrimSpace(site.OwnerEmail)
	if !strings.Contains(recipient, "@") {
		return
	}
	subjectPrefix := "New developer error"
	if alertKind == sentryAlertKindRegressed {
		subjectPrefix = "Regressed developer error"
	}
	subject := fmt.Sprintf("%s for %s", subjectPrefix, strings.TrimSpace(site.Name))
	messageBuilder := &strings.Builder{}
	_, _ = fmt.Fprintf(messageBuilder, "%s\n\n", issue.Title)
	_, _ = fmt.Fprintf(messageBuilder, "Site: %s\n", strings.TrimSpace(site.Name))
	_, _ = fmt.Fprintf(messageBuilder, "Level: %s\n", issue.Level)
	_, _ = fmt.Fprintf(messageBuilder, "Environment: %s\n", issue.Environment)
	if issue.Release != "" {
		_, _ = fmt.Fprintf(messageBuilder, "Release: %s\n", issue.Release)
	}
	_, _ = fmt.Fprintf(messageBuilder, "Occurrences: %d\n", issue.OccurrenceCount)
	if occurrence.RawMessage != "" {
		_, _ = fmt.Fprintf(messageBuilder, "\nMessage:\n%s\n", occurrence.RawMessage)
	}
	if sendErr := handlers.emailSender.SendEmail(ctx, recipient, subject, messageBuilder.String()); sendErr != nil {
		handlers.logger.Warn("sentry_alert_failed", zap.Error(sendErr), zap.String("site_id", site.ID), zap.String("issue_id", issue.ID))
	}
}
