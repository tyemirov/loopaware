package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
)

const (
	testFlagNameDatabaseDriver       = "db-driver"
	testFlagNameDatabaseDataSource   = "db-dsn"
	testFlagNamePublicBaseURL        = "public-base-url"
	testSQLiteDefaultFileName        = "loopaware.sqlite"
	testSQLiteDataSourcePattern      = "file:%s?_foreign_keys=on"
	testDefaultPublicBaseURL         = "http://localhost:8080"
	testAdminEmail                   = "admin@example.com"
	testGoogleClientID               = "client-id"
	testGoogleClientSecret           = "client-secret"
	testSessionSecret                = "session-secret"
	testAdministratorsEnvironmentKey = "ADMINS"
	testConfigAdminFirstEmail        = "config-admin-one@example.com"
	testConfigAdminSecondEmail       = "config-admin-two@example.com"
	testEnvironmentAdminFirstEmail   = "environment-admin-one@example.com"
	testEnvironmentAdminSecondEmail  = "environment-admin-two@example.com"
	testConfigFileName               = "config.yaml"
)

func TestEnsureRequiredConfigurationDetectsMissingFields(t *testing.T) {
	baseConfig := ServerConfig{
		ApplicationAddress:     ":0",
		DatabaseDriverName:     storage.DriverNameSQLite,
		DatabaseDataSourceName: "",
		AdminEmailAddresses:    []string{testAdminEmail},
		GoogleClientID:         testGoogleClientID,
		GoogleClientSecret:     testGoogleClientSecret,
		SessionSecret:          testSessionSecret,
		PublicBaseURL:          testDefaultPublicBaseURL,
		ConfigFilePath:         "testdata/config.yaml",
	}

	testCases := []struct {
		name          string
		mutate        func(*ServerConfig)
		expectsError  bool
		expectedToken string
	}{
		{
			name: "missing database driver",
			mutate: func(config *ServerConfig) {
				config.DatabaseDriverName = ""
			},
			expectsError:  true,
			expectedToken: testFlagNameDatabaseDriver,
		},
		{
			name: "missing datasource for non-sqlite",
			mutate: func(config *ServerConfig) {
				config.DatabaseDriverName = "postgres"
				config.DatabaseDataSourceName = ""
			},
			expectsError:  true,
			expectedToken: testFlagNameDatabaseDataSource,
		},
		{
			name: "missing admins",
			mutate: func(config *ServerConfig) {
				config.AdminEmailAddresses = nil
			},
			expectsError:  true,
			expectedToken: configurationKeyAdmins,
		},
		{
			name: "missing google client id",
			mutate: func(config *ServerConfig) {
				config.GoogleClientID = ""
			},
			expectsError:  true,
			expectedToken: flagNameGoogleClientID,
		},
		{
			name: "missing google client secret",
			mutate: func(config *ServerConfig) {
				config.GoogleClientSecret = ""
			},
			expectsError:  true,
			expectedToken: flagNameGoogleClientSecret,
		},
		{
			name: "missing session secret",
			mutate: func(config *ServerConfig) {
				config.SessionSecret = ""
			},
			expectsError:  true,
			expectedToken: flagNameSessionSecret,
		},
		{
			name: "missing public base url",
			mutate: func(config *ServerConfig) {
				config.PublicBaseURL = ""
			},
			expectsError:  true,
			expectedToken: flagNamePublicBaseURL,
		},
		{
			name:         "valid configuration",
			mutate:       func(config *ServerConfig) {},
			expectsError: false,
		},
	}

	application := NewServerApplication()

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(testingT *testing.T) {
			config := baseConfig
			testCase.mutate(&config)
			err := application.ensureRequiredConfiguration(config)
			if testCase.expectsError {
				require.Error(testingT, err)
				require.Contains(testingT, err.Error(), testCase.expectedToken)
			} else {
				require.NoError(testingT, err)
			}
		})
	}
}

func TestServerCommandFlagDefaults(t *testing.T) {
	expectedSQLiteDataSource := fmt.Sprintf(testSQLiteDataSourcePattern, testSQLiteDefaultFileName)

	application := NewServerApplication()
	command, commandErr := application.Command()
	require.NoError(t, commandErr)

	driverFlag := command.Flag(testFlagNameDatabaseDriver)
	require.NotNil(t, driverFlag)
	require.Equal(t, storage.DriverNameSQLite, driverFlag.DefValue)

	dataSourceFlag := command.Flag(testFlagNameDatabaseDataSource)
	require.NotNil(t, dataSourceFlag)
	require.Equal(t, expectedSQLiteDataSource, dataSourceFlag.DefValue)

	publicBaseFlag := command.Flag(testFlagNamePublicBaseURL)
	require.NotNil(t, publicBaseFlag)
	require.Equal(t, testDefaultPublicBaseURL, publicBaseFlag.DefValue)
}

func TestLoadServerConfigReadsAdminEmailsFromEnvironment(t *testing.T) {
	tempDirectory := t.TempDir()
	configFilePath := filepath.Join(tempDirectory, testConfigFileName)
	configFileContents := fmt.Sprintf("admins:\n  - %s\n  - %s\n", testConfigAdminFirstEmail, testConfigAdminSecondEmail)
	writeErr := os.WriteFile(configFilePath, []byte(configFileContents), 0600)
	require.NoError(t, writeErr)

	testCases := []struct {
		name                                string
		environmentAdministratorsValue      string
		expectedAdministratorEmailAddresses []string
	}{
		{
			name:                                "config administrators used when environment empty",
			environmentAdministratorsValue:      "",
			expectedAdministratorEmailAddresses: []string{testConfigAdminFirstEmail, testConfigAdminSecondEmail},
		},
		{
			name:                                "environment administrators override config",
			environmentAdministratorsValue:      fmt.Sprintf("%s,%s", testEnvironmentAdminFirstEmail, testEnvironmentAdminSecondEmail),
			expectedAdministratorEmailAddresses: []string{testEnvironmentAdminFirstEmail, testEnvironmentAdminSecondEmail},
		},
		{
			name:                                "environment administrators trimmed whitespace",
			environmentAdministratorsValue:      fmt.Sprintf("%s, %s", testEnvironmentAdminFirstEmail, testEnvironmentAdminSecondEmail),
			expectedAdministratorEmailAddresses: []string{testEnvironmentAdminFirstEmail, testEnvironmentAdminSecondEmail},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(testingT *testing.T) {
			testingT.Setenv(testAdministratorsEnvironmentKey, testCase.environmentAdministratorsValue)

			application := NewServerApplication()
			application.configurationLoader.AutomaticEnv()

			serverConfig, loadErr := application.loadServerConfig(configFilePath)
			require.NoError(testingT, loadErr)
			require.Equal(testingT, testCase.expectedAdministratorEmailAddresses, serverConfig.AdminEmailAddresses)
		})
	}
}
