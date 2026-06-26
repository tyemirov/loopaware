package api

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/pkg/outbound"
)

const (
	siteHealthUserAgent       = "LoopAwareHealthCheck/1.0"
	siteHealthAcceptHeader    = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
	siteHealthRedirectLimit   = 10
	siteHealthHTTPErrorFormat = "HTTP %d"
)

var errSiteHealthRedirectLimit = errors.New("site health redirect limit exceeded")

// SiteHealthProbeResult captures one synthetic health-check attempt.
type SiteHealthProbeResult struct {
	TargetURL    string
	Success      bool
	StatusCode   int
	ErrorCode    string
	ErrorMessage string
	Duration     time.Duration
	CheckedAt    time.Time
}

// SiteHealthProber executes a synthetic site health check.
type SiteHealthProber interface {
	Probe(ctx context.Context, targetURL string, timeout time.Duration) SiteHealthProbeResult
}

// HTTPHealthProber probes sites through a public-network-only HTTP client.
type HTTPHealthProber struct {
	httpClient *http.Client
	now        func() time.Time
}

// NewHTTPHealthProber builds the default health prober.
func NewHTTPHealthProber(httpClient *http.Client) *HTTPHealthProber {
	return &HTTPHealthProber{
		httpClient: outbound.NewSafeHTTPClient(httpClient, 0),
		now:        time.Now,
	}
}

// Probe performs one GET health check against targetURL.
func (prober *HTTPHealthProber) Probe(ctx context.Context, targetURL string, timeout time.Duration) SiteHealthProbeResult {
	if prober == nil {
		prober = NewHTTPHealthProber(nil)
	}
	checkedAt := prober.now().UTC()
	normalizedTarget, targetErr := model.NormalizeSiteHealthTargetURL(targetURL)
	if targetErr != nil {
		return siteHealthProbeFailure(normalizedTarget, checkedAt, 0, model.SiteHealthErrorInvalidTarget, targetErr)
	}
	parsedTarget, parseErr := url.Parse(normalizedTarget)
	if parseErr != nil {
		return siteHealthProbeFailure(normalizedTarget, checkedAt, 0, model.SiteHealthErrorInvalidTarget, parseErr)
	}
	if validateErr := outbound.ValidatePublicHTTPURL(parsedTarget); validateErr != nil {
		return siteHealthProbeFailure(normalizedTarget, checkedAt, 0, model.SiteHealthErrorInvalidTarget, validateErr)
	}

	requestCtx := ctx
	if requestCtx == nil {
		requestCtx = context.Background()
	}
	if timeout <= 0 {
		timeout = time.Duration(model.DefaultSiteHealthTimeoutSeconds) * time.Second
	}
	requestCtx, cancel := context.WithTimeout(requestCtx, timeout)
	defer cancel()

	startedAt := prober.now()
	request, requestErr := http.NewRequestWithContext(requestCtx, http.MethodGet, normalizedTarget, nil)
	if requestErr != nil {
		return siteHealthProbeFailure(normalizedTarget, checkedAt, prober.now().Sub(startedAt), model.SiteHealthErrorInvalidTarget, requestErr)
	}
	request.Header.Set("User-Agent", siteHealthUserAgent)
	request.Header.Set("Accept", siteHealthAcceptHeader)

	httpClient := prober.clientForProbe()
	response, requestErr := httpClient.Do(request)
	duration := prober.now().Sub(startedAt)
	if requestErr != nil {
		errorCode := classifySiteHealthRequestError(requestErr)
		return siteHealthProbeFailure(normalizedTarget, checkedAt, duration, errorCode, requestErr)
	}
	if response == nil {
		return siteHealthProbeFailure(normalizedTarget, checkedAt, duration, model.SiteHealthErrorNetwork, errors.New("missing response"))
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusInternalServerError {
		message := fmt.Sprintf(siteHealthHTTPErrorFormat, response.StatusCode)
		return SiteHealthProbeResult{
			TargetURL:    normalizedTarget,
			Success:      false,
			StatusCode:   response.StatusCode,
			ErrorCode:    model.SiteHealthErrorHTTP5xx,
			ErrorMessage: message,
			Duration:     duration,
			CheckedAt:    checkedAt,
		}
	}

	return SiteHealthProbeResult{
		TargetURL:  normalizedTarget,
		Success:    true,
		StatusCode: response.StatusCode,
		ErrorCode:  model.SiteHealthErrorNone,
		Duration:   duration,
		CheckedAt:  checkedAt,
	}
}

func (prober *HTTPHealthProber) clientForProbe() *http.Client {
	httpClient := prober.httpClient
	if httpClient == nil {
		httpClient = outbound.NewSafeHTTPClient(nil, 0)
	}
	clientCopy := *httpClient
	clientCopy.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= siteHealthRedirectLimit {
			return errSiteHealthRedirectLimit
		}
		return outbound.ValidatePublicHTTPURL(request.URL)
	}
	return &clientCopy
}

func siteHealthProbeFailure(targetURL string, checkedAt time.Time, duration time.Duration, errorCode string, probeErr error) SiteHealthProbeResult {
	return SiteHealthProbeResult{
		TargetURL:    strings.TrimSpace(targetURL),
		Success:      false,
		ErrorCode:    model.TruncateSiteHealthErrorCode(errorCode),
		ErrorMessage: model.TruncateSiteHealthErrorMessage(probeErr.Error()),
		Duration:     duration,
		CheckedAt:    checkedAt,
	}
}

func classifySiteHealthRequestError(requestErr error) string {
	if requestErr == nil {
		return model.SiteHealthErrorNetwork
	}
	if errors.Is(requestErr, errSiteHealthRedirectLimit) {
		return model.SiteHealthErrorRedirect
	}
	if errors.Is(requestErr, context.DeadlineExceeded) {
		return model.SiteHealthErrorTimeout
	}
	var urlErr *url.Error
	if errors.As(requestErr, &urlErr) {
		if urlErr.Timeout() {
			return model.SiteHealthErrorTimeout
		}
		if errors.Is(urlErr.Err, errSiteHealthRedirectLimit) {
			return model.SiteHealthErrorRedirect
		}
		if strings.Contains(strings.ToLower(urlErr.Err.Error()), "outbound") {
			return model.SiteHealthErrorInvalidTarget
		}
	}
	var netErr net.Error
	if errors.As(requestErr, &netErr) && netErr.Timeout() {
		return model.SiteHealthErrorTimeout
	}
	var dnsErr *net.DNSError
	if errors.As(requestErr, &dnsErr) {
		return model.SiteHealthErrorDNS
	}
	var unknownAuthorityErr x509.UnknownAuthorityError
	if errors.As(requestErr, &unknownAuthorityErr) {
		return model.SiteHealthErrorTLS
	}
	var hostnameErr x509.HostnameError
	if errors.As(requestErr, &hostnameErr) {
		return model.SiteHealthErrorTLS
	}
	var certificateInvalidErr x509.CertificateInvalidError
	if errors.As(requestErr, &certificateInvalidErr) {
		return model.SiteHealthErrorTLS
	}
	lowerMessage := strings.ToLower(requestErr.Error())
	if strings.Contains(lowerMessage, "certificate") || strings.Contains(lowerMessage, "tls") {
		return model.SiteHealthErrorTLS
	}
	if strings.Contains(lowerMessage, "redirect") {
		return model.SiteHealthErrorRedirect
	}
	if strings.Contains(lowerMessage, "outbound") || strings.Contains(lowerMessage, "non-public") {
		return model.SiteHealthErrorInvalidTarget
	}
	return model.SiteHealthErrorNetwork
}
