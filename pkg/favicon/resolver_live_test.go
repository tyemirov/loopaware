package favicon

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	liveFaviconTestEnvironmentKey = "LOOPAWARE_LIVE_FAVICON_TESTS"
	liveFaviconTestEnabledValue   = "1"
	liveFaviconLookupTimeout      = 15 * time.Second
)

func TestHTTPResolverLiveKnownSitesReturnFavicons(testingT *testing.T) {
	if strings.TrimSpace(os.Getenv(liveFaviconTestEnvironmentKey)) != liveFaviconTestEnabledValue {
		testingT.Skip("set LOOPAWARE_LIVE_FAVICON_TESTS=1 to run live favicon checks")
	}

	testCases := []struct {
		name          string
		allowedOrigin string
	}{
		{name: "google", allowedOrigin: "https://www.google.com/"},
		{name: "wikipedia", allowedOrigin: "https://www.wikipedia.org/"},
		{name: "github", allowedOrigin: "https://github.com/"},
		{name: "apple", allowedOrigin: "https://www.apple.com/"},
		{name: "microsoft", allowedOrigin: "https://www.microsoft.com/"},
		{name: "reddit", allowedOrigin: "https://www.reddit.com/"},
	}

	resolver := NewHTTPResolver(nil, zap.NewNop())

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), liveFaviconLookupTimeout)
			testingT.Cleanup(cancel)

			asset, resolveErr := resolver.ResolveAsset(ctx, testCase.allowedOrigin)
			require.NoError(testingT, resolveErr)
			require.NotNil(testingT, asset)
			require.NotEmpty(testingT, asset.Data)
			require.True(testingT, isSupportedContentType(asset.ContentType), asset.ContentType)
		})
	}
}
