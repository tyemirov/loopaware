package favicon

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/net/html"
)

const (
	testResolverOrigin          = "https://example.com"
	testResolverIconPath        = "/favicon.ico"
	testResolverIconURL         = "https://example.com/favicon.ico"
	testResolverRelativeIconURL = "https://example.com/icon.png"
	testResolverIconContentType = "image/png"
	testResolverHTMLContentType = "text/html; charset=utf-8"
	testResolverCacheTTL        = time.Hour
	testResolverDataPayload     = "icon"
	testResolverUnsupportedType = "text/plain"
	testResolverDefaultType     = "application/octet-stream"
	testRelShortcutIcon         = "shortcut icon"
	testRelMaskIcon             = "mask-icon"
	testRelAppleTouchIcon       = "apple-touch-icon"
	testRelStylesheet           = "stylesheet"
	testFaviconCandidateIcon    = "/icon.svg"
	testFaviconCandidateMask    = "/mask.svg"
	testFaviconCandidateStyle   = "/style.css"
	testSupportedTypeIcon       = "application/x-icon"
	testSupportedTypeSVG        = "application/svg+xml"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTripper roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTripper(request)
}

func TestHTTPResolverResolveReturnsEmptyForInvalidOrigin(testingT *testing.T) {
	resolver := NewHTTPResolver(nil, zap.NewNop())

	testCases := []string{"", "   ", "not-a-url", "http://"}
	for _, origin := range testCases {
		resolved, resolveErr := resolver.Resolve(context.Background(), origin)
		require.NoError(testingT, resolveErr)
		require.Empty(testingT, resolved)
	}
}

func TestHTTPResolverResolveCachesResult(testingT *testing.T) {
	requestCount := 0
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requestCount++
			if request.URL.Path != testResolverIconPath {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("missing")),
					Header:     http.Header{},
				}, nil
			}
			header := make(http.Header)
			header.Set("Content-Type", testResolverIconContentType)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(testResolverDataPayload)),
				Header:     header,
			}, nil
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())
	resolver.cacheTTL = testResolverCacheTTL

	resolved, resolveErr := resolver.Resolve(context.Background(), testResolverOrigin)
	require.NoError(testingT, resolveErr)
	require.Equal(testingT, testResolverIconURL, resolved)

	resolvedAgain, resolveErr := resolver.Resolve(context.Background(), testResolverOrigin)
	require.NoError(testingT, resolveErr)
	require.Equal(testingT, resolved, resolvedAgain)
	require.Equal(testingT, 1, requestCount)
}

func TestHTTPResolverResolveAssetReturnsNilForEmptyOrigin(testingT *testing.T) {
	resolver := NewHTTPResolver(nil, zap.NewNop())

	asset, resolveErr := resolver.ResolveAsset(context.Background(), "   ")
	require.NoError(testingT, resolveErr)
	require.Nil(testingT, asset)
}

func TestHTTPResolverResolveAssetUsesDataURL(testingT *testing.T) {
	payload := []byte(testResolverDataPayload)
	encoded := base64.StdEncoding.EncodeToString(payload)
	htmlBody := `<html><head><link rel="icon" href="data:image/png;base64,` + encoded + `"></head></html>`

	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.URL.Path == testResolverIconPath {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("missing")),
					Header:     http.Header{},
				}, nil
			}
			header := make(http.Header)
			header.Set("Content-Type", testResolverHTMLContentType)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(htmlBody)),
				Header:     header,
			}, nil
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())

	asset, resolveErr := resolver.ResolveAsset(context.Background(), testResolverOrigin)
	require.NoError(testingT, resolveErr)
	require.NotNil(testingT, asset)
	require.Equal(testingT, testResolverIconContentType, asset.ContentType)
	require.Equal(testingT, payload, asset.Data)
}

func TestFetchRemoteIconAssetSkipsBadStatus(testingT *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("missing")),
				Header:     http.Header{},
			}, nil
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())

	asset, fetchErr := resolver.fetchRemoteIconAsset(context.Background(), testResolverIconURL)
	require.NoError(testingT, fetchErr)
	require.Nil(testingT, asset)
}

func TestFetchRemoteIconAssetRejectsUnsupportedContentType(testingT *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			header := make(http.Header)
			header.Set("Content-Type", testResolverUnsupportedType)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(testResolverDataPayload)),
				Header:     header,
			}, nil
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())

	asset, fetchErr := resolver.fetchRemoteIconAsset(context.Background(), testResolverIconURL)
	require.NoError(testingT, fetchErr)
	require.Nil(testingT, asset)
}

func TestFetchRemoteIconAssetDefaultsContentType(testingT *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(testResolverDataPayload)),
				Header:     http.Header{},
			}, nil
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())

	asset, fetchErr := resolver.fetchRemoteIconAsset(context.Background(), testResolverIconURL)
	require.NoError(testingT, fetchErr)
	require.NotNil(testingT, asset)
	require.Equal(testingT, testResolverDefaultType, asset.ContentType)
	require.Equal(testingT, []byte(testResolverDataPayload), asset.Data)
}

func TestFetchRemoteIconAssetRejectsEmptyPayload(testingT *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			header := make(http.Header)
			header.Set("Content-Type", testResolverIconContentType)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     header,
			}, nil
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())

	asset, fetchErr := resolver.fetchRemoteIconAsset(context.Background(), testResolverIconURL)
	require.NoError(testingT, fetchErr)
	require.Nil(testingT, asset)
}

func TestFetchRemoteIconAssetRejectsOversizedPayload(testingT *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			header := make(http.Header)
			header.Set("Content-Type", testResolverIconContentType)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("toolong")),
				Header:     header,
			}, nil
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())
	resolver.maxIconBytes = 3

	asset, fetchErr := resolver.fetchRemoteIconAsset(context.Background(), testResolverIconURL)
	require.Error(testingT, fetchErr)
	require.Nil(testingT, asset)
}

func TestFetchHTMLDeclaredFaviconResolvesRemote(testingT *testing.T) {
	htmlBody := `<html><head><link rel="icon" href="/icon.png"></head></html>`
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			header := make(http.Header)
			if request.URL.Path == "/" {
				header.Set("Content-Type", testResolverHTMLContentType)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(htmlBody)),
					Header:     header,
				}, nil
			}
			if request.URL.Path == "/icon.png" {
				header.Set("Content-Type", testResolverIconContentType)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(testResolverDataPayload)),
					Header:     header,
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("missing")),
				Header:     header,
			}, nil
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())
	root, parseErr := url.Parse(testResolverOrigin)
	require.NoError(testingT, parseErr)

	candidate, fetchErr := resolver.fetchHTMLDeclaredFavicon(context.Background(), root, []string{"/"})
	require.NoError(testingT, fetchErr)
	require.NotNil(testingT, candidate)
	require.Equal(testingT, testResolverRelativeIconURL, candidate.remoteURL)
	require.NotNil(testingT, candidate.asset)
}

func TestFetchHTMLDeclaredFaviconReturnsParseError(testingT *testing.T) {
	htmlBody := `<html><head><link rel="icon" href="data:text/plain,hello"></head></html>`
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			header := make(http.Header)
			header.Set("Content-Type", testResolverHTMLContentType)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(htmlBody)),
				Header:     header,
			}, nil
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())
	root, parseErr := url.Parse(testResolverOrigin)
	require.NoError(testingT, parseErr)

	candidate, fetchErr := resolver.fetchHTMLDeclaredFavicon(context.Background(), root, []string{"/"})
	require.Error(testingT, fetchErr)
	require.Nil(testingT, candidate)
}

func TestParseDataURLRejectsInvalidPayload(testingT *testing.T) {
	resolver := NewHTTPResolver(nil, zap.NewNop())
	resolver.maxIconBytes = 4

	testCases := []struct {
		name  string
		value string
	}{
		{name: "empty", value: ""},
		{name: "missing prefix", value: "image/png,abcd"},
		{name: "missing comma", value: "data:image/png;base64abcd"},
		{name: "unsupported type", value: "data:text/plain,hello"},
		{name: "invalid base64", value: "data:image/png;base64,@@@"},
		{name: "payload too large", value: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("toolong"))},
	}

	for _, testCase := range testCases {
		asset, parseErr := resolver.parseDataURL(testCase.value)
		require.Error(testingT, parseErr, testCase.name)
		require.Nil(testingT, asset)
	}
}

func TestParseDataURLAcceptsBase64Payload(testingT *testing.T) {
	resolver := NewHTTPResolver(nil, zap.NewNop())
	value := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(testResolverDataPayload))

	asset, parseErr := resolver.parseDataURL(value)
	require.NoError(testingT, parseErr)
	require.Equal(testingT, testResolverIconContentType, asset.ContentType)
	require.Equal(testingT, []byte(testResolverDataPayload), asset.Data)
}

func TestHTMLProbePathsIncludesBasePath(testingT *testing.T) {
	resolver := NewHTTPResolver(nil, zap.NewNop())
	baseURL, parseErr := url.Parse("https://example.com/app")
	require.NoError(testingT, parseErr)

	paths := resolver.htmlProbePaths(baseURL)
	require.Contains(testingT, paths, "/app")
	require.Contains(testingT, paths, "/app/")
	require.Contains(testingT, paths, "/app/index.html")
	require.Contains(testingT, paths, "/")
}

func TestGetReturnsNilForBadStatus(testingT *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("missing")),
				Header:     http.Header{},
			}, nil
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())

	body, fetchErr := resolver.get(context.Background(), testResolverIconURL, 10)
	require.NoError(testingT, fetchErr)
	require.Nil(testingT, body)
}

func TestDoRequestRejectsInvalidTarget(testingT *testing.T) {
	resolver := NewHTTPResolver(nil, zap.NewNop())

	response, requestErr := resolver.doRequest(context.Background(), http.MethodGet, "://invalid")
	require.Error(testingT, requestErr)
	require.Nil(testingT, response)
}

func TestRequestContextUsesBackgroundForNil(testingT *testing.T) {
	resolver := NewHTTPResolver(nil, zap.NewNop())
	var optionalContext context.Context
	requestContext := resolver.requestContext(optionalContext)
	require.NotNil(testingT, requestContext)
	require.NoError(testingT, requestContext.Err())
}

func TestDecodeBase64HandlesStandardAndRaw(testingT *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(testResolverDataPayload))
	decoded, decodeErr := decodeBase64(encoded)
	require.NoError(testingT, decodeErr)
	require.Equal(testingT, []byte(testResolverDataPayload), decoded)

	rawEncoded := strings.TrimRight(encoded, "=")
	rawDecoded, rawDecodeErr := decodeBase64(rawEncoded)
	require.NoError(testingT, rawDecodeErr)
	require.Equal(testingT, []byte(testResolverDataPayload), rawDecoded)
}

func TestDecodeBase64RejectsEmptyPayload(testingT *testing.T) {
	decoded, decodeErr := decodeBase64("")
	require.Error(testingT, decodeErr)
	require.Nil(testingT, decoded)
}

func TestIsSupportedContentTypeValidatesInput(testingT *testing.T) {
	require.True(testingT, isSupportedContentType("image/png"))
	require.True(testingT, isSupportedContentType("application/octet-stream"))
	require.True(testingT, isSupportedContentType("image/x-icon"))
	require.True(testingT, isSupportedContentType("image/svg+xml"))
	require.False(testingT, isSupportedContentType(testResolverUnsupportedType))
}

func TestAbsoluteURLRejectsInvalidHref(testingT *testing.T) {
	resolver := NewHTTPResolver(nil, zap.NewNop())
	root, parseErr := url.Parse(testResolverOrigin)
	require.NoError(testingT, parseErr)

	absolute, resolveErr := resolver.absoluteURL(root, "http://%zz")
	require.Error(testingT, resolveErr)
	require.Empty(testingT, absolute)
}

func TestRelContainsIconRejectsNonIcon(testingT *testing.T) {
	require.False(testingT, relContainsIcon(""))
	require.False(testingT, relContainsIcon("stylesheet"))
}

func TestFindFaviconCandidatesExtractsLinks(testingT *testing.T) {
	htmlBody := `<html><head>
<link rel="icon" href="/favicon.ico">
<link rel="apple-touch-icon" href="/apple.png">
<link rel="mask-icon" href="/mask.svg">
</head></html>`
	document, parseErr := html.Parse(strings.NewReader(htmlBody))
	require.NoError(testingT, parseErr)

	candidates := findFaviconCandidates(document)
	require.Len(testingT, candidates, 3)
}

func TestNewLimitedReadCloserRespectsLimits(testingT *testing.T) {
	reader := io.NopCloser(bytes.NewBufferString("payload"))
	limited := newLimitedReadCloser(reader, 3)
	buffer := make([]byte, 10)
	readCount, readErr := limited.Read(buffer)
	require.NoError(testingT, readErr)
	require.Equal(testingT, 3, readCount)
	require.NoError(testingT, limited.Close())

	unlimitedReader := io.NopCloser(bytes.NewBufferString("payload"))
	unlimited := newLimitedReadCloser(unlimitedReader, 0)
	unlimitedBytes, readErr := io.ReadAll(unlimited)
	require.NoError(testingT, readErr)
	require.Equal(testingT, "payload", string(unlimitedBytes))
	require.NoError(testingT, unlimited.Close())
}

func TestHTTPResolverResolveAssetReturnsNilForInvalidOrigin(testingT *testing.T) {
	resolver := NewHTTPResolver(nil, zap.NewNop())

	asset, resolveErr := resolver.ResolveAsset(context.Background(), "http://")
	require.NoError(testingT, resolveErr)
	require.Nil(testingT, asset)
}

func TestHTTPResolverResolveAssetReturnsNilWhenNoCandidate(testingT *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("missing")),
				Header:     http.Header{},
			}, nil
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())

	asset, resolveErr := resolver.ResolveAsset(context.Background(), testResolverOrigin)
	require.NoError(testingT, resolveErr)
	require.Nil(testingT, asset)
}

func TestHTTPResolverResolveAssetReturnsErrorOnLookupFailure(testingT *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())

	asset, resolveErr := resolver.ResolveAsset(context.Background(), testResolverOrigin)
	require.Error(testingT, resolveErr)
	require.Nil(testingT, asset)
}

func TestFetchHTMLDeclaredFaviconReportsInvalidHref(testingT *testing.T) {
	htmlBody := `<html><head><link rel="icon" href="http://%zz"></head></html>`
	client := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			header := make(http.Header)
			header.Set("Content-Type", testResolverHTMLContentType)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(htmlBody)),
				Header:     header,
			}, nil
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())
	root, parseErr := url.Parse(testResolverOrigin)
	require.NoError(testingT, parseErr)

	candidate, fetchErr := resolver.fetchHTMLDeclaredFavicon(context.Background(), root, []string{"/"})
	require.Error(testingT, fetchErr)
	require.Nil(testingT, candidate)
}

func TestGetReturnsBodyForOKStatus(testingT *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("content")),
				Header:     http.Header{},
			}, nil
		}),
	}
	resolver := NewHTTPResolver(client, zap.NewNop())

	body, fetchErr := resolver.get(context.Background(), testResolverIconURL, 3)
	require.NoError(testingT, fetchErr)
	require.NotNil(testingT, body)
	defer body.Close()

	payload, readErr := io.ReadAll(body)
	require.NoError(testingT, readErr)
	require.Equal(testingT, "con", string(payload))
}

func TestIsSupportedContentTypeAcceptsEmptyAndBinary(testingT *testing.T) {
	require.True(testingT, isSupportedContentType(""))
	require.True(testingT, isSupportedContentType("binary/octet-stream"))
}

func TestRelContainsIconAcceptsIconValues(testingT *testing.T) {
	require.True(testingT, relContainsIcon("icon"))
	require.True(testingT, relContainsIcon("apple-touch-icon"))
}

func TestIsSupportedContentTypeAcceptsIconAndSVG(testingT *testing.T) {
	require.True(testingT, isSupportedContentType(testSupportedTypeIcon))
	require.True(testingT, isSupportedContentType(testSupportedTypeSVG))
	require.False(testingT, isSupportedContentType(testRelStylesheet))
}

func TestFindFaviconCandidatesFiltersLinks(testingT *testing.T) {
	htmlBody := fmt.Sprintf(`<!doctype html><html><head><link rel="%s" href="%s"><link rel="%s" href="%s"><link rel="%s" href="%s"></head><body></body></html>`, testRelShortcutIcon, testFaviconCandidateIcon, testRelMaskIcon, testFaviconCandidateMask, testRelStylesheet, testFaviconCandidateStyle)

	root, parseErr := html.Parse(strings.NewReader(htmlBody))
	require.NoError(testingT, parseErr)

	candidates := findFaviconCandidates(root)
	require.Contains(testingT, candidates, testFaviconCandidateIcon)
	require.Contains(testingT, candidates, testFaviconCandidateMask)
	require.NotContains(testingT, candidates, testFaviconCandidateStyle)
}
