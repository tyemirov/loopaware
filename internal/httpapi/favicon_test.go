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
