package httpapi

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewSiteTrafficTestHandlersDefaultsLogger(testingT *testing.T) {
	handlers := NewSiteTrafficTestHandlers(nil, nil, AuthClientConfig{})
	require.NotNil(testingT, handlers.logger)
}

func TestNewSiteWidgetTestHandlersDefaultsLogger(testingT *testing.T) {
	handlers := NewSiteWidgetTestHandlers(nil, nil, "https://widget.example", nil, nil, AuthClientConfig{})
	require.NotNil(testingT, handlers.logger)
}

func TestNewSiteSubscribeTestHandlersDefaultsLogger(testingT *testing.T) {
	handlers := NewSiteSubscribeTestHandlers(nil, nil, nil, nil, false, "http://example.test", "secret", nil, AuthClientConfig{})
	require.NotNil(testingT, handlers.logger)
	require.NotEmpty(testingT, handlers.subscriptionTokenTTL)
}

func TestNewDashboardWebHandlersDefaultsLandingPath(testingT *testing.T) {
	handlers := NewDashboardWebHandlers(zap.NewNop(), "", AuthClientConfig{})
	require.Equal(testingT, "/", handlers.landingPath)
}
