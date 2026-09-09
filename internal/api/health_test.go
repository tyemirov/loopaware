package api_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MarkoPoloResearchLab/loopaware/internal/api"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestHealthReadinessAndQuietSuccess(testingT *testing.T) {
	fixture := testutil.NewSQLiteTestDatabase(testingT)
	database, err := storage.OpenDatabase(fixture.Configuration())
	require.NoError(testingT, err)
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	handlers := api.NewPublicHandlers(database, logger, nil, nil, nil, nil, false, "http://localhost", "health-test", nil)
	router := gin.New()
	router.Use(api.RequestLogger(logger))
	router.GET(api.HealthPath, handlers.Health)
	server := httptest.NewServer(router)
	defer server.Close()
	check := func(status int, body string) {
		response, requestErr := server.Client().Get(server.URL + api.HealthPath)
		require.NoError(testingT, requestErr)
		defer response.Body.Close()
		payload, readErr := io.ReadAll(response.Body)
		require.NoError(testingT, readErr)
		require.Equal(testingT, status, response.StatusCode)
		require.Equal(testingT, "no-store", response.Header.Get("Cache-Control"))
		require.JSONEq(testingT, body, string(payload))
	}
	check(http.StatusOK, `{"status":"ok"}`)
	require.Zero(testingT, logs.Len())
	connection, err := database.DB()
	require.NoError(testingT, err)
	require.NoError(testingT, connection.Close())
	check(http.StatusServiceUnavailable, `{"status":"unavailable"}`)
	require.Equal(testingT, 1, logs.FilterMessage("health_probe_failed").Len())
	require.Equal(testingT, 1, logs.FilterMessage("http").Len())
}
