package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
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
		PinguinAddress:         defaultPinguinAddress,
		PinguinAuthToken:       "test-token",
		PinguinConnTimeoutSec:  defaultPinguinConnTimeoutSeconds,
		PinguinOpTimeoutSec:    defaultPinguinOpTimeoutSeconds,
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
			expectsError: false,
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
			name: "missing pinguin address",
			mutate: func(config *ServerConfig) {
				config.PinguinAddress = ""
			},
			expectsError:  true,
			expectedToken: flagNamePinguinAddress,
		},
		{
			name: "missing pinguin auth token",
			mutate: func(config *ServerConfig) {
				config.PinguinAuthToken = ""
			},
			expectsError:  true,
			expectedToken: flagNamePinguinAuthToken,
		},
		{
			name: "missing pinguin connection timeout",
			mutate: func(config *ServerConfig) {
				config.PinguinConnTimeoutSec = 0
			},
			expectsError:  true,
			expectedToken: flagNamePinguinConnectionTimeout,
		},
		{
			name: "missing pinguin operation timeout",
			mutate: func(config *ServerConfig) {
				config.PinguinOpTimeoutSec = 0
			},
			expectsError:  true,
			expectedToken: flagNamePinguinOperationTimeout,
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

	pinguinAddrFlag := command.Flag(flagNamePinguinAddress)
	require.NotNil(t, pinguinAddrFlag)
	require.Equal(t, defaultPinguinAddress, pinguinAddrFlag.DefValue)

	pinguinAuthFlag := command.Flag(flagNamePinguinAuthToken)
	require.NotNil(t, pinguinAuthFlag)
	require.Equal(t, "", pinguinAuthFlag.DefValue)

	pinguinConnFlag := command.Flag(flagNamePinguinConnectionTimeout)
	require.NotNil(t, pinguinConnFlag)
	require.Equal(t, fmt.Sprintf("%d", defaultPinguinConnTimeoutSeconds), pinguinConnFlag.DefValue)

	pinguinOpFlag := command.Flag(flagNamePinguinOperationTimeout)
	require.NotNil(t, pinguinOpFlag)
	require.Equal(t, fmt.Sprintf("%d", defaultPinguinOpTimeoutSeconds), pinguinOpFlag.DefValue)
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

func TestDockerfileUsesLoopawareBinary(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "Dockerfile"))
	require.NoError(t, err)
	text := string(contents)
	require.Contains(t, text, "/out/loopaware")
	require.Contains(t, text, "/app/loopaware")
	require.NotContains(t, text, "feedbacksvc")
}

func TestLoadServerConfigAllowsMissingConfigFile(t *testing.T) {
	tempDirectory := t.TempDir()
	missingConfigFilePath := filepath.Join(tempDirectory, testConfigFileName)

	t.Setenv(testAdministratorsEnvironmentKey, fmt.Sprintf("%s,%s", testEnvironmentAdminFirstEmail, testEnvironmentAdminSecondEmail))
	t.Setenv(environmentKeyGoogleClientID, testGoogleClientID)
	t.Setenv(environmentKeyGoogleClientSecret, testGoogleClientSecret)
	t.Setenv(environmentKeySessionSecret, testSessionSecret)
	t.Setenv(environmentKeyPublicBaseURL, testDefaultPublicBaseURL)

	application := NewServerApplication()
	application.configurationLoader.AutomaticEnv()

	serverConfig, loadErr := application.loadServerConfig(missingConfigFilePath)
	require.NoError(t, loadErr)
	require.Equal(t, []string{testEnvironmentAdminFirstEmail, testEnvironmentAdminSecondEmail}, serverConfig.AdminEmailAddresses)
	require.Equal(t, missingConfigFilePath, serverConfig.ConfigFilePath)
}

func TestLoadServerConfigFallsBackToSharedAuthToken(t *testing.T) {
	tempDirectory := t.TempDir()
	configFilePath := filepath.Join(tempDirectory, testConfigFileName)
	require.NoError(t, os.WriteFile(configFilePath, []byte("admins: []\n"), 0600))

	t.Setenv(environmentKeyGoogleClientID, testGoogleClientID)
	t.Setenv(environmentKeyGoogleClientSecret, testGoogleClientSecret)
	t.Setenv(environmentKeySessionSecret, testSessionSecret)
	t.Setenv(environmentKeyPublicBaseURL, testDefaultPublicBaseURL)
	t.Setenv(environmentKeyPinguinAuthToken, "")
	t.Setenv(environmentKeyPinguinSharedAuth, "shared-token")

	application := NewServerApplication()
	application.configurationLoader.AutomaticEnv()

	serverConfig, loadErr := application.loadServerConfig(configFilePath)
	require.NoError(t, loadErr)
	require.Equal(t, "shared-token", serverConfig.PinguinAuthToken)
}

func TestLogAdministratorWarning(t *testing.T) {
	logObserver, logObserverEntries := observer.New(zapcore.WarnLevel)
	logger := zap.New(logObserver)

	application := NewServerApplication()

	application.logAdministratorWarning(logger, ServerConfig{AdminEmailAddresses: nil})
	require.Equal(t, 1, logObserverEntries.Len())
	warningEntry := logObserverEntries.All()[0]
	require.Equal(t, zapcore.WarnLevel, warningEntry.Level)
	require.Equal(t, logMessageMissingAdministrators, warningEntry.Message)

	logObserverEntries.TakeAll()

	application.logAdministratorWarning(logger, ServerConfig{AdminEmailAddresses: []string{testAdminEmail}})
	require.Equal(t, 0, logObserverEntries.Len())
}
