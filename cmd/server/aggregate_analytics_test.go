package main

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestServerRetainsOnlyCurrentAggregateCounts(testingT *testing.T) {
	var recoveryLog bytes.Buffer
	originalWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &recoveryLog
	testingT.Cleanup(func() { gin.DefaultErrorWriter = originalWriter })
	sqlite := testutil.NewSQLiteTestDatabase(testingT)
	database, err := storage.OpenDatabase(sqlite.Configuration())
	require.NoError(testingT, err)
	require.NoError(testingT, storage.AutoMigrate(database))
	site := model.Site{ID: storage.NewID(), Name: "Game", AllowedOrigin: "https://game.example", TrafficProfile: model.TrafficProfileAggregate}
	require.NoError(testingT, database.Create(&site).Error)
	rows := []model.SiteVisitCount{
		{SiteID: site.ID, Date: time.Now().UTC().AddDate(0, 0, -180).Format("2006-01-02"), Count: 12},
		{SiteID: site.ID, Date: time.Now().UTC().Format("2006-01-02"), Count: 3},
	}
	require.NoError(testingT, database.Create(&rows).Error)
	listener := startPinguinServer(testingT)
	application := NewServerApplication().WithDatabaseOpener(func(storage.Config) (*gorm.DB, error) { return database, nil }).WithPinguinDialer(createPinguinDialer(listener))
	application.WithServerRunner(func(server *http.Server) error {
		httpServer := httptest.NewServer(server.Handler)
		defer httpServer.Close()
		request, err := http.NewRequest(http.MethodPost, httpServer.URL+"/public/sites/"+site.ID+"/visit-counts", strings.NewReader("{}"))
		require.NoError(testingT, err)
		request.Header.Set("Origin", site.AllowedOrigin)
		request.Header.Set("Content-Type", "application/json")
		response, err := httpServer.Client().Do(request)
		require.NoError(testingT, err)
		response.Body.Close()
		require.Equal(testingT, http.StatusNoContent, response.StatusCode)
		var remaining []model.SiteVisitCount
		require.NoError(testingT, database.Find(&remaining).Error)
		require.Equal(testingT, []model.SiteVisitCount{{SiteID: site.ID, Date: rows[1].Date, Count: 4}}, remaining)
		require.NoError(testingT, database.Callback().Raw().Before("gorm:raw").Register("panic_count", func(*gorm.DB) { panic("private-transport-marker") }))
		defer database.Callback().Raw().Remove("panic_count")
		request, err = http.NewRequest(http.MethodPost, httpServer.URL+"/public/sites/"+site.ID+"/visit-counts", strings.NewReader("{}"))
		require.NoError(testingT, err)
		request.Header.Set("Origin", site.AllowedOrigin)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("User-Agent", "private-device-marker")
		response, err = httpServer.Client().Do(request)
		require.NoError(testingT, err)
		response.Body.Close()
		require.Equal(testingT, http.StatusInternalServerError, response.StatusCode)
		require.NotContains(testingT, recoveryLog.String(), "private-")
		return http.ErrServerClosed
	})
	command, err := application.Command()
	require.NoError(testingT, err)
	require.NoError(testingT, command.Flags().Set(flagNameConfigFile, writeServerConfig(testingT, validServerConfigYAML(testPinguinAddress))))
	require.NoError(testingT, command.Execute())
}
