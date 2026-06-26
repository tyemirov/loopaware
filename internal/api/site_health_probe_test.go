package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

type siteHealthRoundTripper func(*http.Request) (*http.Response, error)

func (roundTripper siteHealthRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTripper(request)
}

func TestHTTPHealthProberClassifiesResponses(testingT *testing.T) {
	probeTime := time.Date(2026, time.June, 25, 18, 0, 0, 0, time.UTC)
	var capturedUserAgent string
	prober := NewHTTPHealthProber(&http.Client{
		Transport: siteHealthRoundTripper(func(request *http.Request) (*http.Response, error) {
			capturedUserAgent = request.Header.Get("User-Agent")
			statusCode := http.StatusNotFound
			if request.URL.Path == "/down" {
				statusCode = http.StatusBadGateway
			}
			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	})
	prober.now = func() time.Time {
		return probeTime
	}

	downResult := prober.Probe(context.Background(), "https://health-probe.example.com/down", 2*time.Second)
	require.False(testingT, downResult.Success)
	require.Equal(testingT, http.StatusBadGateway, downResult.StatusCode)
	require.Equal(testingT, model.SiteHealthErrorHTTP5xx, downResult.ErrorCode)
	require.Equal(testingT, "HTTP 502", downResult.ErrorMessage)
	require.Equal(testingT, probeTime, downResult.CheckedAt)
	require.Equal(testingT, siteHealthUserAgent, capturedUserAgent)

	upResult := prober.Probe(context.Background(), "https://health-probe.example.com/missing", 2*time.Second)
	require.True(testingT, upResult.Success)
	require.Equal(testingT, http.StatusNotFound, upResult.StatusCode)
	require.Empty(testingT, upResult.ErrorCode)
}

func TestHTTPHealthProberClassifiesInvalidTargetAndTimeout(testingT *testing.T) {
	probeTime := time.Date(2026, time.June, 25, 18, 30, 0, 0, time.UTC)
	prober := NewHTTPHealthProber(&http.Client{
		Transport: siteHealthRoundTripper(func(*http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		}),
	})
	prober.now = func() time.Time {
		return probeTime
	}

	invalidTarget := prober.Probe(context.Background(), "http://127.0.0.1/healthz", 2*time.Second)
	require.False(testingT, invalidTarget.Success)
	require.Equal(testingT, model.SiteHealthErrorInvalidTarget, invalidTarget.ErrorCode)

	timeoutResult := prober.Probe(context.Background(), "https://health-probe.example.com/timeout", 2*time.Second)
	require.False(testingT, timeoutResult.Success)
	require.Equal(testingT, model.SiteHealthErrorTimeout, timeoutResult.ErrorCode)
}
