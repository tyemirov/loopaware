package favicon_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/feedback_svc/pkg/favicon"
)

type stubResolver struct {
	asset      *favicon.Asset
	resolveErr error
}

func (resolver *stubResolver) Resolve(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (resolver *stubResolver) ResolveAsset(_ context.Context, _ string) (*favicon.Asset, error) {
	return resolver.asset, resolver.resolveErr
}

func TestServiceCollect(testingT *testing.T) {
	testTimestamp := time.Date(2024, time.January, 15, 12, 0, 0, 0, time.UTC)
	existingSite := favicon.Site{
		FaviconData:        []byte{0x01, 0x02},
		FaviconContentType: "image/png",
		FaviconFetchedAt:   testTimestamp.Add(-time.Hour),
	}

	testCases := []struct {
		name             string
		resolver         *stubResolver
		site             favicon.Site
		notify           bool
		expectedKeys     []string
		expectDataUpdate bool
		expectNotify     bool
		expectError      error
	}{
		{
			name: "updatesWhenAssetChanges",
			resolver: &stubResolver{
				asset: &favicon.Asset{ContentType: "image/png", Data: []byte{0x0A}},
			},
			site:             existingSite,
			expectedKeys:     []string{"favicon_origin", "favicon_last_attempt_at", "favicon_data", "favicon_content_type", "favicon_fetched_at"},
			expectDataUpdate: true,
			expectNotify:     true,
		},
		{
			name: "skipsWhenAssetMissing",
			resolver: &stubResolver{
				asset: nil,
			},
			site:         existingSite,
			expectedKeys: []string{"favicon_origin", "favicon_last_attempt_at"},
		},
		{
			name: "notifiesWhenRequested",
			resolver: &stubResolver{
				asset: &favicon.Asset{ContentType: "image/png", Data: existingSite.FaviconData},
			},
			site:         existingSite,
			notify:       true,
			expectedKeys: []string{"favicon_origin", "favicon_last_attempt_at", "favicon_fetched_at"},
			expectNotify: true,
		},
		{
			name: "returnsErrorAndUpdates",
			resolver: &stubResolver{
				asset:      nil,
				resolveErr: errors.New("lookup failed"),
			},
			site:         existingSite,
			expectedKeys: []string{"favicon_origin", "favicon_last_attempt_at"},
			expectError:  errors.New("lookup failed"),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		testingT.Run(testCase.name, func(nestedT *testing.T) {
			service := favicon.NewService(testCase.resolver)
			result, collectErr := service.Collect(
				context.Background(),
				testCase.site,
				" https://example.com ",
				testCase.notify,
				testTimestamp,
			)

			if testCase.expectError != nil {
				require.EqualError(nestedT, collectErr, testCase.expectError.Error())
			} else {
				require.NoError(nestedT, collectErr)
			}

			require.NotNil(nestedT, result.Updates)
			require.Len(nestedT, result.Updates, len(testCase.expectedKeys))
			for _, key := range testCase.expectedKeys {
				_, exists := result.Updates[key]
				require.True(nestedT, exists, "expected key %s in updates", key)
			}

			if testCase.expectDataUpdate {
				require.Equal(nestedT, []byte{0x0A}, result.Updates["favicon_data"])
				require.Equal(nestedT, "image/png", result.Updates["favicon_content_type"])
			} else {
				_, hasData := result.Updates["favicon_data"]
				require.False(nestedT, hasData)
			}

			if testCase.expectNotify {
				require.True(nestedT, result.ShouldNotify)
				require.True(nestedT, result.EventTimestamp.Equal(testTimestamp))
			} else {
				require.False(nestedT, result.ShouldNotify)
				require.True(nestedT, result.EventTimestamp.IsZero())
			}

			require.Equal(nestedT, "https://example.com", result.Updates["favicon_origin"])
			require.Equal(nestedT, testTimestamp, result.Updates["favicon_last_attempt_at"])
		})
	}
}

func TestServiceCollectReturnsEmptyResultForBlankOrigin(testingT *testing.T) {
	service := favicon.NewService(&stubResolver{})
	result, err := service.Collect(context.Background(), favicon.Site{}, "   ", false, time.Now())
	require.NoError(testingT, err)
	require.Nil(testingT, result.Updates)
	require.False(testingT, result.ShouldNotify)
	require.True(testingT, result.EventTimestamp.IsZero())
}

func TestServiceCollectValidatesResolverPresence(testingT *testing.T) {
	service := favicon.NewService(nil)
	_, err := service.Collect(context.Background(), favicon.Site{}, "https://example.com", false, time.Now())
	require.EqualError(testingT, err, "favicon resolver is not configured")
}
