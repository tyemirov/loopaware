package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTrafficIdentityServerAppliesPolicyAndHistoricalMigration(testingT *testing.T) {
	for _, excludePrivate := range []bool{true, false} {
		testingT.Run(fmt.Sprint(excludePrivate), func(testingT *testing.T) {
			notifier := startPinguinServer(testingT)
			origin := httptest.NewServer(http.NotFoundHandler())
			defer origin.Close()
			database, err := storage.OpenDatabase(storage.Config{DriverName: storage.DriverNameSQLite, DataSourceName: filepath.Join(testingT.TempDir(), "traffic.sqlite")})
			require.NoError(testingT, err)
			sqlDatabase, err := database.DB()
			require.NoError(testingT, err)
			defer sqlDatabase.Close()
			require.NoError(testingT, storage.AutoMigrate(database))
			site := model.Site{ID: storage.NewID(), Name: "Startup traffic", AllowedOrigin: origin.URL, TrafficProfile: model.TrafficProfileDetailed}
			require.NoError(testingT, database.Create(&site).Error)
			historical, err := model.NewSiteVisit(model.SiteVisitInput{SiteID: site.ID, URL: origin.URL, VisitorID: storage.NewID(), IP: "192.168.96.5", UserAgent: "HeadlessChrome/143.0", Occurred: time.Now().UTC()})
			require.NoError(testingT, err)
			require.NoError(testingT, database.Create(&historical).Error)
			config := strings.Replace(validServerConfigYAML(notifier.Addr().String()), "exclude_private_ips: false", fmt.Sprintf("exclude_private_ips: %t", excludePrivate), 1)
			application := NewServerApplication().WithDatabaseOpener(func(storage.Config) (*gorm.DB, error) { return database, nil })
			application.WithServerRunner(func(configured *http.Server) error {
				var corrected model.SiteVisit
				require.NoError(testingT, database.First(&corrected, "id = ?", historical.ID).Error)
				require.True(testingT, corrected.IsBot, "historical correction must finish before serving")
				server := httptest.NewServer(configured.Handler)
				defer server.Close()
				request, err := http.NewRequest(http.MethodGet, server.URL+"/public/visits?site_id="+site.ID+"&url="+origin.URL+"&visitor_id="+storage.NewID(), nil)
				require.NoError(testingT, err)
				request.Header.Set("Origin", origin.URL)
				response, err := server.Client().Do(request)
				require.NoError(testingT, err)
				require.NoError(testingT, response.Body.Close())
				require.Equal(testingT, http.StatusOK, response.StatusCode)
				var count int64
				require.NoError(testingT, database.Model(&model.SiteVisit{}).Where("is_bot = ?", false).Count(&count).Error)
				expected := int64(1)
				if excludePrivate {
					expected = 0
				}
				require.Equal(testingT, expected, count)
				return http.ErrServerClosed
			})
			command, err := application.Command()
			require.NoError(testingT, err)
			require.NoError(testingT, command.Flags().Set(flagNameConfigFile, writeServerConfig(testingT, config)))
			require.NoError(testingT, application.runCommand(command, nil))
		})
	}
}
