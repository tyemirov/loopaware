package favicon_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/pkg/favicon"
)

type roundTripperFunc func(request *http.Request) (*http.Response, error)

func (roundTripper roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTripper(request)
}

type stubHTTPResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

func newStubHTTPClient(responseByPath map[string]stubHTTPResponse) *http.Client {
	return &http.Client{
		Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			response, ok := responseByPath[request.URL.Path]
			if !ok {
				response = stubHTTPResponse{StatusCode: http.StatusNotFound}
			}
			headers := make(http.Header)
			if response.ContentType != "" {
				headers.Set("Content-Type", response.ContentType)
			}
			body := io.NopCloser(bytes.NewReader(response.Body))
			return &http.Response{
				StatusCode: response.StatusCode,
				Header:     headers,
				Body:       body,
				Request:    request,
			}, nil
		}),
	}
}

func TestHTTPResolverPrefersDefaultIcon(testingT *testing.T) {
	allowedOrigin := "http://icon.test"
	httpClient := newStubHTTPClient(map[string]stubHTTPResponse{
		"/favicon.ico": {
			StatusCode:  http.StatusOK,
			ContentType: "image/x-icon",
			Body:        []byte{0x00, 0x01},
		},
	})

	resolver := favicon.NewHTTPResolver(httpClient, zap.NewNop())

	faviconURL, resolveErr := resolver.Resolve(context.Background(), allowedOrigin)
	require.NoError(testingT, resolveErr)
	require.Equal(testingT, allowedOrigin+"/favicon.ico", faviconURL)
}

func TestHTTPResolverParsesHTMLLinks(testingT *testing.T) {
	iconPath := "/assets/icon.png"
	htmlResponse := "<!doctype html><html><head><link rel=\"icon\" href=\"" + iconPath + "\"></head><body></body></html>"
	allowedOrigin := "http://html.test"
	httpClient := newStubHTTPClient(map[string]stubHTTPResponse{
		"/favicon.ico": {
			StatusCode: http.StatusNotFound,
		},
		"/": {
			StatusCode:  http.StatusOK,
			ContentType: "text/html",
			Body:        []byte(htmlResponse),
		},
		iconPath: {
			StatusCode:  http.StatusOK,
			ContentType: "image/png",
			Body:        []byte{0x89, 0x50, 0x4e, 0x47},
		},
	})

	resolver := favicon.NewHTTPResolver(httpClient, zap.NewNop())

	faviconURL, resolveErr := resolver.Resolve(context.Background(), allowedOrigin)
	require.NoError(testingT, resolveErr)
	require.Equal(testingT, allowedOrigin+iconPath, faviconURL)
}

func TestHTTPResolverSupportsInlineData(testingT *testing.T) {
	inlineData := "data:image/png;base64,iVBORw0KGgo="
	htmlResponse := "<!doctype html><html><head><link rel=\"icon\" href=\"" + inlineData + "\"></head></html>"
	allowedOrigin := "http://inline.test"
	httpClient := newStubHTTPClient(map[string]stubHTTPResponse{
		"/favicon.ico": {
			StatusCode: http.StatusNotFound,
		},
		"/": {
			StatusCode:  http.StatusOK,
			ContentType: "text/html",
			Body:        []byte(htmlResponse),
		},
	})

	resolver := favicon.NewHTTPResolver(httpClient, zap.NewNop())

	faviconURL, resolveErr := resolver.Resolve(context.Background(), allowedOrigin)
	require.NoError(testingT, resolveErr)
	require.Equal(testingT, inlineData, faviconURL)
}

func TestHTTPResolverResolveAssetReturnsBinaryData(testingT *testing.T) {
	iconBytes := []byte{0x00, 0x11, 0x22}
	allowedOrigin := "http://asset.test"
	httpClient := newStubHTTPClient(map[string]stubHTTPResponse{
		"/favicon.ico": {
			StatusCode:  http.StatusOK,
			ContentType: "image/x-icon",
			Body:        iconBytes,
		},
	})

	resolver := favicon.NewHTTPResolver(httpClient, zap.NewNop())

	asset, resolveErr := resolver.ResolveAsset(context.Background(), allowedOrigin)
	require.NoError(testingT, resolveErr)
	require.NotNil(testingT, asset)
	require.Equal(testingT, "image/x-icon", asset.ContentType)
	require.Equal(testingT, iconBytes, asset.Data)
}

func TestHTTPResolverResolveAssetParsesInlineData(testingT *testing.T) {
	inlineData := "data:image/svg+xml;base64,PHN2Zy8+"
	htmlResponse := "<!doctype html><html><head><link rel=\"icon\" href=\"" + inlineData + "\"></head></html>"
	allowedOrigin := "http://inline-asset.test"
	httpClient := newStubHTTPClient(map[string]stubHTTPResponse{
		"/favicon.ico": {
			StatusCode: http.StatusNotFound,
		},
		"/": {
			StatusCode:  http.StatusOK,
			ContentType: "text/html",
			Body:        []byte(htmlResponse),
		},
	})

	resolver := favicon.NewHTTPResolver(httpClient, zap.NewNop())

	asset, resolveErr := resolver.ResolveAsset(context.Background(), allowedOrigin)
	require.NoError(testingT, resolveErr)
	require.NotNil(testingT, asset)
	require.Equal(testingT, "image/svg+xml", asset.ContentType)
	require.Equal(testingT, []byte("<svg/>"), asset.Data)
}

func TestHTTPResolverResolveAssetReturnsNilForUnsupportedContentType(testingT *testing.T) {
	htmlResponse := "<!doctype html><html><head><link rel=\"icon\" href=\"/icon\"></head></html>"
	allowedOrigin := "http://unsupported.test"
	httpClient := newStubHTTPClient(map[string]stubHTTPResponse{
		"/favicon.ico": {
			StatusCode: http.StatusNotFound,
		},
		"/": {
			StatusCode:  http.StatusOK,
			ContentType: "text/html",
			Body:        []byte(htmlResponse),
		},
		"/icon": {
			StatusCode:  http.StatusOK,
			ContentType: "text/plain",
			Body:        []byte("not an icon"),
		},
	})

	resolver := favicon.NewHTTPResolver(httpClient, zap.NewNop())

	asset, resolveErr := resolver.ResolveAsset(context.Background(), allowedOrigin)
	require.NoError(testingT, resolveErr)
	require.Nil(testingT, asset)
}

func TestHTTPResolverFallsBackToAppPath(testingT *testing.T) {
	inlineData := "data:image/svg+xml;utf8,<svg/>"
	allowedOrigin := "http://fallback.test/app"
	httpClient := newStubHTTPClient(map[string]stubHTTPResponse{
		"/favicon.ico": {
			StatusCode: http.StatusNotFound,
		},
		"/": {
			StatusCode: http.StatusNotFound,
		},
		"/app": {
			StatusCode:  http.StatusOK,
			ContentType: "text/html",
			Body:        []byte("<!doctype html><html><head><link rel=\"icon\" href=\"" + inlineData + "\"></head></html>"),
		},
	})

	resolver := favicon.NewHTTPResolver(httpClient, zap.NewNop())

	asset, resolveErr := resolver.ResolveAsset(context.Background(), allowedOrigin)
	require.NoError(testingT, resolveErr)
	require.NotNil(testingT, asset)
	require.Equal(testingT, "image/svg+xml", asset.ContentType)
	require.Equal(testingT, []byte("<svg/>"), asset.Data)
}
