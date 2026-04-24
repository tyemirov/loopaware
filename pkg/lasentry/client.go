package lasentry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultPlatform         = "go"
	defaultEnvironment      = "production"
	defaultLevel            = "error"
	defaultHTTPClientTimout = 5 * time.Second
	maxCapturedFrames       = 32
)

// HTTPClient executes outbound HTTP requests.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Config describes a LoopAware LA Sentry client.
type Config struct {
	SiteID      string
	Endpoint    string
	Token       string
	Platform    string
	Environment string
	Release     string
	HTTPClient  HTTPClient
}

// Client submits developer error events to LoopAware.
type Client struct {
	siteID      string
	endpoint    string
	token       string
	platform    string
	environment string
	release     string
	httpClient  HTTPClient
}

// Attributes captures optional event context.
type Attributes struct {
	Level         string
	Message       string
	ExceptionType string
	Stacktrace    []StackFrame
	Request       map[string]any
	UserHash      string
	Tags          map[string]string
	Extra         map[string]any
}

// StackFrame describes one client-side stack frame.
type StackFrame struct {
	Filename string `json:"filename"`
	Function string `json:"function"`
	Module   string `json:"module"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	InApp    bool   `json:"in_app"`
}

type eventPayload struct {
	SiteID        string            `json:"site_id"`
	EventID       string            `json:"event_id"`
	Timestamp     string            `json:"timestamp"`
	Platform      string            `json:"platform"`
	Environment   string            `json:"environment"`
	Release       string            `json:"release"`
	Level         string            `json:"level"`
	Message       string            `json:"message"`
	ExceptionType string            `json:"exception_type"`
	Stacktrace    []StackFrame      `json:"stacktrace"`
	Request       map[string]any    `json:"request"`
	UserHash      string            `json:"user_hash"`
	Tags          map[string]string `json:"tags"`
	Extra         map[string]any    `json:"extra"`
}

// NewClient constructs a LoopAware LA Sentry client.
func NewClient(config Config) (*Client, error) {
	siteID := strings.TrimSpace(config.SiteID)
	if siteID == "" {
		return nil, errors.New("la sentry site id is required")
	}
	endpoint := strings.TrimSpace(config.Endpoint)
	if endpoint == "" {
		return nil, errors.New("la sentry endpoint is required")
	}
	token := strings.TrimSpace(config.Token)
	if token == "" {
		return nil, errors.New("la sentry token is required")
	}
	platform := strings.TrimSpace(config.Platform)
	if platform == "" {
		platform = defaultPlatform
	}
	environment := strings.TrimSpace(config.Environment)
	if environment == "" {
		environment = defaultEnvironment
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPClientTimout}
	}
	return &Client{
		siteID:      siteID,
		endpoint:    endpoint,
		token:       token,
		platform:    platform,
		environment: environment,
		release:     strings.TrimSpace(config.Release),
		httpClient:  httpClient,
	}, nil
}

// CaptureError submits an explicit error event.
func (client *Client) CaptureError(ctx context.Context, capturedErr error, attributes Attributes) error {
	if client == nil {
		return errors.New("la sentry client is nil")
	}
	if capturedErr == nil {
		return errors.New("la sentry error is required")
	}
	level := strings.TrimSpace(attributes.Level)
	if level == "" {
		level = defaultLevel
	}
	message := strings.TrimSpace(attributes.Message)
	if message == "" {
		message = capturedErr.Error()
	}
	exceptionType := strings.TrimSpace(attributes.ExceptionType)
	if exceptionType == "" {
		exceptionType = fmt.Sprintf("%T", capturedErr)
	}
	stacktrace := attributes.Stacktrace
	if len(stacktrace) == 0 {
		stacktrace = captureRuntimeStackFrames(3)
	}
	payload := eventPayload{
		SiteID:        client.siteID,
		EventID:       uuid.NewString(),
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Platform:      client.platform,
		Environment:   client.environment,
		Release:       client.release,
		Level:         level,
		Message:       message,
		ExceptionType: exceptionType,
		Stacktrace:    stacktrace,
		Request:       attributes.Request,
		UserHash:      strings.TrimSpace(attributes.UserHash),
		Tags:          attributes.Tags,
		Extra:         attributes.Extra,
	}
	return client.submit(ctx, payload)
}

// Middleware captures panics from an HTTP handler and returns a 500 response.
func (client *Client) Middleware(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		defer func() {
			recoveredValue := recover()
			if recoveredValue == nil {
				return
			}
			_ = client.CaptureError(request.Context(), fmt.Errorf("panic: %v", recoveredValue), Attributes{
				Level:         "fatal",
				ExceptionType: "panic",
				Stacktrace:    captureRuntimeStackFrames(4),
				Request:       RequestMetadata(request),
			})
			http.Error(responseWriter, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}()
		next.ServeHTTP(responseWriter, request)
	})
}

// RequestMetadata extracts non-secret request metadata for an error event.
func RequestMetadata(request *http.Request) map[string]any {
	if request == nil {
		return nil
	}
	requestURL := ""
	if request.URL != nil {
		sanitizedURL := *request.URL
		sanitizedURL.RawQuery = ""
		sanitizedURL.ForceQuery = false
		sanitizedURL.Fragment = ""
		sanitizedURL.RawFragment = ""
		requestURL = sanitizedURL.String()
	}
	return map[string]any{
		"method":     request.Method,
		"url":        requestURL,
		"host":       request.Host,
		"user_agent": request.UserAgent(),
		"remote":     request.RemoteAddr,
	}
}

func (client *Client) submit(ctx context.Context, payload eventPayload) error {
	serializedPayload, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return fmt.Errorf("marshal la sentry event: %w", marshalErr)
	}
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(serializedPayload))
	if requestErr != nil {
		return fmt.Errorf("create la sentry request: %w", requestErr)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+client.token)
	response, responseErr := client.httpClient.Do(request)
	if responseErr != nil {
		return fmt.Errorf("submit la sentry event: %w", responseErr)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("submit la sentry event: status %d", response.StatusCode)
	}
	return nil
}

func captureRuntimeStackFrames(skip int) []StackFrame {
	programCounters := make([]uintptr, maxCapturedFrames)
	frameCount := runtime.Callers(skip, programCounters)
	runtimeFrames := runtime.CallersFrames(programCounters[:frameCount])
	stackFrames := make([]StackFrame, 0, frameCount)
	for {
		runtimeFrame, more := runtimeFrames.Next()
		stackFrames = append(stackFrames, StackFrame{
			Filename: runtimeFrame.File,
			Function: runtimeFrame.Function,
			Line:     runtimeFrame.Line,
			InApp:    true,
		})
		if !more {
			break
		}
	}
	return stackFrames
}
