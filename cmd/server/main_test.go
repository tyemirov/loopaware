package main_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	servercmd "github.com/MarkoPoloResearchLab/feedback_svc/cmd/server"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
	"gorm.io/gorm"
)

const (
	testEnvironmentKeyDatabaseDriverName     = "DB_DRIVER"
	testEnvironmentKeyDatabaseDataSource     = "DB_DSN"
	testEnvironmentKeyAdminBearerToken       = "ADMIN_BEARER_TOKEN"
	testAlternateDatabaseDriverName          = "alternate-driver"
	testAlternateDatabaseDataSource          = "db://example.com/database"
	testPlaceholderAdminBearerToken          = "very-secret-token"
	testMissingConfigurationMessage          = "missing required configuration"
	testFlagNameDatabaseDriver               = "db-driver"
	testFlagNameDatabaseDataSource           = "db-dsn"
	testFlagNameAdminBearerToken             = "admin-bearer-token"
	testFlagIndicator                        = "--"
	testUsagePrefix                          = "Usage:"
	testSQLiteDefaultFileName                = "loopaware.sqlite"
	testSQLiteDataSourceNamePattern          = "file:%s?_foreign_keys=on"
	testMissingDatabaseDriverDescription     = "missing database driver"
	testMissingDatabaseDataSourceDescription = "missing database data source"
	testMissingAdminBearerDescription        = "missing admin bearer token"
)

func TestServerCommandMissingConfigurationShowsHelp(t *testing.T) {
	sqliteDefaultDataSource := fmt.Sprintf(testSQLiteDataSourceNamePattern, testSQLiteDefaultFileName)

	testCases := []struct {
		name                string
		environment         map[string]string
		expectedMissingFlag string
	}{
		{
			name: testMissingDatabaseDriverDescription,
			environment: map[string]string{
				testEnvironmentKeyDatabaseDriverName: "",
				testEnvironmentKeyDatabaseDataSource: testAlternateDatabaseDataSource,
				testEnvironmentKeyAdminBearerToken:   testPlaceholderAdminBearerToken,
			},
			expectedMissingFlag: testFlagNameDatabaseDriver,
		},
		{
			name: testMissingDatabaseDataSourceDescription,
			environment: map[string]string{
				testEnvironmentKeyDatabaseDriverName: testAlternateDatabaseDriverName,
				testEnvironmentKeyDatabaseDataSource: "",
				testEnvironmentKeyAdminBearerToken:   testPlaceholderAdminBearerToken,
			},
			expectedMissingFlag: testFlagNameDatabaseDataSource,
		},
		{
			name: testMissingAdminBearerDescription,
			environment: map[string]string{
				testEnvironmentKeyDatabaseDriverName: storage.DriverNameSQLite,
				testEnvironmentKeyDatabaseDataSource: sqliteDefaultDataSource,
				testEnvironmentKeyAdminBearerToken:   "",
			},
			expectedMissingFlag: testFlagNameAdminBearerToken,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(testingT *testing.T) {
			for key, value := range testCase.environment {
				testingT.Setenv(key, value)
			}

			databaseOpenerStub := func(configuration storage.Config) (*gorm.DB, error) {
				testingT.Fatalf("database opener invoked with %+v", configuration)
				return nil, nil
			}

			application := servercmd.NewServerApplication().WithDatabaseOpener(databaseOpenerStub)
			command, commandErr := application.Command()
			require.NoError(testingT, commandErr)

			commandOutput := &bytes.Buffer{}
			command.SetOut(commandOutput)
			command.SetErr(commandOutput)

			executionErr := command.Execute()
			if executionErr == nil {
				testingT.Fatalf("expected error for missing configuration")
			}

			combinedOutput := commandOutput.String()
			if !strings.Contains(combinedOutput, testMissingConfigurationMessage) {
				testingT.Fatalf("expected combined output to mention missing configuration: %s", combinedOutput)
			}

			if !strings.Contains(combinedOutput, testUsagePrefix) {
				testingT.Fatalf("expected combined output to include usage instructions: %s", combinedOutput)
			}

			expectedFlagIndicator := testFlagIndicator + testCase.expectedMissingFlag
			if !strings.Contains(combinedOutput, expectedFlagIndicator) {
				testingT.Fatalf("expected help output to include flag %s, actual output: %s", expectedFlagIndicator, combinedOutput)
			}
		})
	}
}

func TestServerCommandFlagDefaults(t *testing.T) {
	expectedSQLiteDataSource := fmt.Sprintf(testSQLiteDataSourceNamePattern, testSQLiteDefaultFileName)

	application := servercmd.NewServerApplication()
	command, commandErr := application.Command()
	require.NoError(t, commandErr)

	driverFlag := command.Flag(testFlagNameDatabaseDriver)
	require.NotNil(t, driverFlag)
	require.Equal(t, storage.DriverNameSQLite, driverFlag.DefValue)

	dataSourceFlag := command.Flag(testFlagNameDatabaseDataSource)
	require.NotNil(t, dataSourceFlag)
	require.Equal(t, expectedSQLiteDataSource, dataSourceFlag.DefValue)
}
