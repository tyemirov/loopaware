package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/httpapi"
)

func TestHTTPFaviconResolverPrefersDefaultIcon(t *testing.T) {
	iconServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/favicon.ico":
			writer.Header().Set("Content-Type", "image/x-icon")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte{0x00, 0x01})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(iconServer.Close)

	resolver := httpapi.NewHTTPFaviconResolver(iconServer.Client(), zap.NewNop())

	faviconURL, err := resolver.Resolve(context.Background(), iconServer.URL)
	require.NoError(t, err)
	require.Equal(t, iconServer.URL+"/favicon.ico", faviconURL)
}

func TestHTTPFaviconResolverParsesHTMLLinks(t *testing.T) {
	iconPath := "/assets/icon.png"
	htmlResponse := "<!doctype html><html><head><link rel=\"icon\" href=\"" + iconPath + "\"></head><body></body></html>"
	iconServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/favicon.ico":
			writer.WriteHeader(http.StatusNotFound)
		case iconPath:
			writer.Header().Set("Content-Type", "image/png")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte{0x89, 0x50, 0x4e, 0x47})
		default:
			writer.Header().Set("Content-Type", "text/html")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(htmlResponse))
		}
	}))
	t.Cleanup(iconServer.Close)

	resolver := httpapi.NewHTTPFaviconResolver(iconServer.Client(), zap.NewNop())

	faviconURL, err := resolver.Resolve(context.Background(), iconServer.URL)
	require.NoError(t, err)
	require.Equal(t, iconServer.URL+iconPath, faviconURL)
}

func TestHTTPFaviconResolverSupportsInlineData(t *testing.T) {
	inlineData := "data:image/png;base64,iVBORw0KGgo="
	htmlResponse := "<!doctype html><html><head><link rel=\"icon\" href=\"" + inlineData + "\"></head></html>"
	iconServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/" {
			writer.Header().Set("Content-Type", "text/html")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(htmlResponse))
			return
		}
		writer.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(iconServer.Close)

	resolver := httpapi.NewHTTPFaviconResolver(iconServer.Client(), zap.NewNop())

	faviconURL, err := resolver.Resolve(context.Background(), iconServer.URL)
	require.NoError(t, err)
	require.Equal(t, inlineData, faviconURL)
}

func TestHTTPFaviconResolverResolveAssetReturnsBinaryData(t *testing.T) {
	iconBytes := []byte{0x00, 0x11, 0x22}
	iconServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/favicon.ico" {
			writer.Header().Set("Content-Type", "image/x-icon")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write(iconBytes)
			return
		}
		writer.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(iconServer.Close)

	resolver := httpapi.NewHTTPFaviconResolver(iconServer.Client(), zap.NewNop())

	asset, err := resolver.ResolveAsset(context.Background(), iconServer.URL)
	require.NoError(t, err)
	require.NotNil(t, asset)
	require.Equal(t, "image/x-icon", asset.ContentType)
	require.Equal(t, iconBytes, asset.Data)
}

func TestHTTPFaviconResolverResolveAssetParsesInlineData(t *testing.T) {
	inlineData := "data:image/svg+xml;base64,PHN2Zy8+"
	htmlResponse := "<!doctype html><html><head><link rel=\"icon\" href=\"" + inlineData + "\"></head></html>"
	iconServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/" {
			writer.Header().Set("Content-Type", "text/html")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(htmlResponse))
			return
		}
		writer.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(iconServer.Close)

	resolver := httpapi.NewHTTPFaviconResolver(iconServer.Client(), zap.NewNop())

	asset, err := resolver.ResolveAsset(context.Background(), iconServer.URL)
	require.NoError(t, err)
	require.NotNil(t, asset)
	require.Equal(t, "image/svg+xml", asset.ContentType)
	require.Equal(t, []byte("<svg/>"), asset.Data)
}

func TestHTTPFaviconResolverResolveAssetReturnsNilForUnsupportedContentType(t *testing.T) {
	htmlResponse := "<!doctype html><html><head><link rel=\"icon\" href=\"/icon\"></head></html>"
	iconServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/":
			writer.Header().Set("Content-Type", "text/html")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(htmlResponse))
		case "/icon":
			writer.Header().Set("Content-Type", "text/plain")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte("not an icon"))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(iconServer.Close)

	resolver := httpapi.NewHTTPFaviconResolver(iconServer.Client(), zap.NewNop())

	asset, err := resolver.ResolveAsset(context.Background(), iconServer.URL)
	require.NoError(t, err)
	require.Nil(t, asset)
}

func TestHTTPFaviconResolverFallsBackToAppPath(t *testing.T) {
	inlineData := "data:image/svg+xml;utf8,<svg/>"
	iconServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/favicon.ico":
			writer.WriteHeader(http.StatusNotFound)
		case "/":
			writer.WriteHeader(http.StatusNotFound)
		case "/app":
			writer.Header().Set("Content-Type", "text/html")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte("<!doctype html><html><head><link rel=\"icon\" href=\"" + inlineData + "\"></head></html>"))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(iconServer.Close)

	resolver := httpapi.NewHTTPFaviconResolver(iconServer.Client(), zap.NewNop())

	asset, err := resolver.ResolveAsset(context.Background(), iconServer.URL+"/app")
	require.NoError(t, err)
	require.NotNil(t, asset)
	require.Equal(t, "image/svg+xml", asset.ContentType)
	require.Equal(t, []byte("<svg/>"), asset.Data)
}
