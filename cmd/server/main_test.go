package main_test

import (
	"bytes"
	"strings"
	"testing"

	servercmd "github.com/MarkoPoloResearchLab/feedback_svc/cmd/server"
	"gorm.io/gorm"
)

const (
	testEnvironmentKeyDatabaseDataSourceName = "DB_DSN"
	testEnvironmentKeyAdminBearerToken       = "ADMIN_BEARER_TOKEN"
	testPlaceholderDatabaseDSN               = "postgres://example.com/database"
	testPlaceholderAdminBearerToken          = "very-secret-token"
	testMissingConfigurationMessage          = "missing required configuration"
	testFlagNameDatabaseDataSource           = "db-dsn"
	testFlagNameAdminBearerToken             = "admin-bearer-token"
	testFlagIndicator                        = "--"
	testUsagePrefix                          = "Usage:"
)

func TestServerCommandMissingConfigurationShowsHelp(t *testing.T) {
	testCases := []struct {
		name                   string
		databaseDataSourceName string
		adminBearerToken       string
		expectedMissingFlag    string
	}{
		{
			name:                   "missing database dsn",
			databaseDataSourceName: "",
			adminBearerToken:       testPlaceholderAdminBearerToken,
			expectedMissingFlag:    testFlagNameDatabaseDataSource,
		},
		{
			name:                   "missing admin bearer token",
			databaseDataSourceName: testPlaceholderDatabaseDSN,
			adminBearerToken:       "",
			expectedMissingFlag:    testFlagNameAdminBearerToken,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv(testEnvironmentKeyDatabaseDataSourceName, testCase.databaseDataSourceName)
			t.Setenv(testEnvironmentKeyAdminBearerToken, testCase.adminBearerToken)

			databaseOpenerStub := func(databaseDataSourceName string) (*gorm.DB, error) {
				t.Fatalf("database opener invoked with %s", databaseDataSourceName)
				return nil, nil
			}

			application := servercmd.NewServerApplication().WithDatabaseOpener(databaseOpenerStub)
			command, commandErr := application.Command()
			if commandErr != nil {
				t.Fatalf("unexpected command construction error: %v", commandErr)
			}

			commandOutput := &bytes.Buffer{}
			command.SetOut(commandOutput)
			command.SetErr(commandOutput)

			executionErr := command.Execute()
			if executionErr == nil {
				t.Fatalf("expected error for missing configuration")
			}

			combinedOutput := commandOutput.String()
			if !strings.Contains(combinedOutput, testMissingConfigurationMessage) {
				t.Fatalf("expected combined output to mention missing configuration: %s", combinedOutput)
			}

			if !strings.Contains(combinedOutput, testUsagePrefix) {
				t.Fatalf("expected combined output to include usage instructions: %s", combinedOutput)
			}

			expectedFlagIndicator := testFlagIndicator + testCase.expectedMissingFlag
			if !strings.Contains(combinedOutput, expectedFlagIndicator) {
				t.Fatalf("expected help output to include flag %s, actual output: %s", expectedFlagIndicator, combinedOutput)
			}
		})
	}
}
