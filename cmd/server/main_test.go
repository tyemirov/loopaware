package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/notifications/pinguinpb"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
)

const (
	testAdminEmailsEnv        = "admin@example.com, owner@example.com"
	testAuthTokenValue        = "test-auth-token"
	testTenantValue           = "test-tenant"
	testGoogleClientIDValue   = "test-google-client"
	testSessionSecretValue    = "test-session-secret"
	testTauthBaseURLValue     = "http://tauth.test"
	testTauthTenantIDValue    = "tenant-id"
	testTauthSigningKeyValue  = "signing-key"
	testTauthCookieNameValue  = "app_session"
	testPublicBaseURLValue    = "http://localhost:8080"
	testPinguinAddress        = "bufnet"
	testDatabaseDriverValue   = storage.DriverNameSQLite
	testDatabaseDSNValue      = "file:server-test?mode=memory&cache=shared&_foreign_keys=on"
	testPinguinTimeoutSeconds = "1"
	testDatabaseOpenerMessage = "database opener error"
	testBufferSize            = 1024 * 1024
	testInvalidTimeoutValue   = "invalid"
	testServerAddress         = "127.0.0.1:0"
	testBindFlagAddress       = "127.0.0.1:9999"
)

type stubNotificationServer struct {
	pinguinpb.UnimplementedNotificationServiceServer
}

func (stub *stubNotificationServer) SendNotification(context.Context, *pinguinpb.NotificationRequest) (*pinguinpb.NotificationResponse, error) {
	return &pinguinpb.NotificationResponse{Status: pinguinpb.Status_SENT}, nil
}

func (stub *stubNotificationServer) GetNotificationStatus(context.Context, *pinguinpb.GetNotificationStatusRequest) (*pinguinpb.NotificationResponse, error) {
	return &pinguinpb.NotificationResponse{Status: pinguinpb.Status_SENT}, nil
}

func startPinguinServer(testingT *testing.T) *bufconn.Listener {
	listener := bufconn.Listen(testBufferSize)
	grpcServer := grpc.NewServer()
	pinguinpb.RegisterNotificationServiceServer(grpcServer, &stubNotificationServer{})

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	testingT.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
	})

	return listener
}

func createPinguinDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(requestContext context.Context, _ string) (net.Conn, error) {
		return listener.DialContext(requestContext)
	}
}

func setRequiredEnvironment(testingT *testing.T, pinguinAddress string) {
	testingT.Setenv(environmentKeyApplicationAddress, "127.0.0.1:0")
	testingT.Setenv(environmentKeyDatabaseDriverName, testDatabaseDriverValue)
	testingT.Setenv(environmentKeyDatabaseDataSource, testDatabaseDSNValue)
	testingT.Setenv(environmentKeyGoogleClientID, testGoogleClientIDValue)
	testingT.Setenv(environmentKeySessionSecret, testSessionSecretValue)
	testingT.Setenv(environmentKeyTauthBaseURL, testTauthBaseURLValue)
	testingT.Setenv(environmentKeyTauthTenantID, testTauthTenantIDValue)
	testingT.Setenv(environmentKeyTauthSigningKey, testTauthSigningKeyValue)
	testingT.Setenv(environmentKeyTauthSessionCookie, testTauthCookieNameValue)
	testingT.Setenv(environmentKeyPublicBaseURL, testPublicBaseURLValue)
	testingT.Setenv(environmentKeyPinguinAddress, pinguinAddress)
	testingT.Setenv(environmentKeyPinguinAuthToken, testAuthTokenValue)
	testingT.Setenv(environmentKeyPinguinTenantID, testTenantValue)
	testingT.Setenv(environmentKeyPinguinConnTimeout, testPinguinTimeoutSeconds)
	testingT.Setenv(environmentKeyPinguinOpTimeout, testPinguinTimeoutSeconds)
	testingT.Setenv(environmentKeySubscriptionNotify, "true")
}

func TestBindFlagReturnsErrorWhenMissing(testingT *testing.T) {
	application := NewServerApplication()
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)

	bindErr := application.bindFlag(flagSet, environmentKeyApplicationAddress, "missing-flag")
	require.Error(testingT, bindErr)
}

func TestBindFlagBindsValue(testingT *testing.T) {
	application := NewServerApplication()
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String(flagNameApplicationAddress, defaultApplicationAddress, "")

	bindErr := application.bindFlag(flagSet, environmentKeyApplicationAddress, flagNameApplicationAddress)
	require.NoError(testingT, bindErr)

	require.NoError(testingT, flagSet.Set(flagNameApplicationAddress, testBindFlagAddress))
	require.Equal(testingT, testBindFlagAddress, application.configurationLoader.GetString(environmentKeyApplicationAddress))
}

func TestNewServerApplicationHasDefaults(testingT *testing.T) {
	application := NewServerApplication()
	require.NotNil(testingT, application.configurationLoader)
	require.NotNil(testingT, application.databaseOpener)
	require.NotNil(testingT, application.serverRunner)
	require.Nil(testingT, application.pinguinDialer)
}

func TestWithPinguinDialerOverrides(testingT *testing.T) {
	application := NewServerApplication()
	application.WithPinguinDialer(func(context.Context, string) (net.Conn, error) {
		return nil, errors.New("dialer error")
	})

	require.NotNil(testingT, application.pinguinDialer)
}

func TestWithDatabaseOpenerOverrides(testingT *testing.T) {
	application := NewServerApplication()
	application.WithDatabaseOpener(func(storage.Config) (*gorm.DB, error) {
		return nil, errors.New(testDatabaseOpenerMessage)
	})

	_, openErr := application.databaseOpener(storage.Config{})
	require.ErrorContains(testingT, openErr, testDatabaseOpenerMessage)
}

func TestWithServerRunnerOverrides(testingT *testing.T) {
	application := NewServerApplication()
	var runnerCalls int
	application.WithServerRunner(func(*http.Server) error {
		runnerCalls++
		return http.ErrServerClosed
	})

	require.NotNil(testingT, application.serverRunner)
	_ = application.serverRunner(&http.Server{})
	require.Equal(testingT, 1, runnerCalls)
}

func TestApplyEnvironmentConfigurationSetsFlag(testingT *testing.T) {
	application := NewServerApplication()
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String(flagNameApplicationAddress, "", "")

	testingT.Setenv(environmentKeyApplicationAddress, "127.0.0.1:1234")
	applyErr := application.applyEnvironmentConfiguration(flagSet, environmentKeyApplicationAddress, flagNameApplicationAddress)
	require.NoError(testingT, applyErr)

	value, valueErr := flagSet.GetString(flagNameApplicationAddress)
	require.NoError(testingT, valueErr)
	require.Equal(testingT, "127.0.0.1:1234", value)
}

func TestApplyEnvironmentConfigurationSkipsWhenMissing(testingT *testing.T) {
	application := NewServerApplication()
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String(flagNameApplicationAddress, "default", "")

	environmentKey := "LOOPAWARE_TEST_MISSING_ENV"
	require.NoError(testingT, os.Unsetenv(environmentKey))
	applyErr := application.applyEnvironmentConfiguration(flagSet, environmentKey, flagNameApplicationAddress)
	require.NoError(testingT, applyErr)

	value, valueErr := flagSet.GetString(flagNameApplicationAddress)
	require.NoError(testingT, valueErr)
	require.Equal(testingT, "default", value)
}

func TestApplyEnvironmentConfigurationReportsUnknownFlag(testingT *testing.T) {
	application := NewServerApplication()
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)

	testingT.Setenv(environmentKeyApplicationAddress, "127.0.0.1:1234")
	applyErr := application.applyEnvironmentConfiguration(flagSet, environmentKeyApplicationAddress, "missing-flag")
	require.Error(testingT, applyErr)
}

func TestApplyEnvironmentConfigurationRejectsInvalidInt(testingT *testing.T) {
	application := NewServerApplication()
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.Int(flagNamePinguinConnectionTimeout, 0, "")

	testingT.Setenv(environmentKeyPinguinConnTimeout, testInvalidTimeoutValue)
	applyErr := application.applyEnvironmentConfiguration(flagSet, environmentKeyPinguinConnTimeout, flagNamePinguinConnectionTimeout)
	require.Error(testingT, applyErr)
	require.ErrorContains(testingT, applyErr, environmentConfigurationError)
}

func TestNormalizeEmailAddressesSkipsBlanks(testingT *testing.T) {
	normalized := normalizeEmailAddresses([]string{" admin@example.com ", "", "owner@example.com"})
	require.Equal(testingT, []string{"admin@example.com", "owner@example.com"}, normalized)
}

func TestLoadServerConfigUsesDefaultsAndEnvironment(testingT *testing.T) {
	application := NewServerApplication()
	_, commandErr := application.Command()
	require.NoError(testingT, commandErr)

	testingT.Setenv(environmentKeyAdmins, testAdminEmailsEnv)
	testingT.Setenv(environmentKeyDatabaseDriverName, storage.DriverNameSQLite)
	testingT.Setenv(environmentKeyDatabaseDataSource, "")
	testingT.Setenv(environmentKeyPinguinSharedAuth, testAuthTokenValue)

	config, loadErr := application.loadServerConfig("")
	require.NoError(testingT, loadErr)
	require.Equal(testingT, []string{"admin@example.com", "owner@example.com"}, config.AdminEmailAddresses)
	require.Equal(testingT, defaultSQLiteDataSourceName, config.DatabaseDataSourceName)
	require.Equal(testingT, testAuthTokenValue, config.PinguinAuthToken)
}

func TestLoadServerConfigReportsConfigurationFileError(testingT *testing.T) {
	application := NewServerApplication()
	_, commandErr := application.Command()
	require.NoError(testingT, commandErr)

	tempDirectory := testingT.TempDir()
	configPath := filepath.Join(tempDirectory, "config.yaml")
	require.NoError(testingT, os.WriteFile(configPath, []byte(": [}"), 0o600))

	_, loadErr := application.loadServerConfig(configPath)
	require.Error(testingT, loadErr)
	require.ErrorContains(testingT, loadErr, configurationFileLoadError)
}

func TestEnsureRequiredConfigurationReportsMissing(testingT *testing.T) {
	application := NewServerApplication()
	missingErr := application.ensureRequiredConfiguration(ServerConfig{})
	require.Error(testingT, missingErr)

	config := ServerConfig{
		DatabaseDriverName:        storage.DriverNameSQLite,
		DatabaseDataSourceName:    testDatabaseDSNValue,
		GoogleClientID:            testGoogleClientIDValue,
		SessionSecret:             testSessionSecretValue,
		TauthBaseURL:              testTauthBaseURLValue,
		TauthTenantID:             testTauthTenantIDValue,
		TauthSigningKey:           testTauthSigningKeyValue,
		PublicBaseURL:             testPublicBaseURLValue,
		PinguinAddress:            "127.0.0.1:50051",
		PinguinAuthToken:          testAuthTokenValue,
		PinguinTenantID:           testTenantValue,
		PinguinConnTimeoutSec:     1,
		PinguinOpTimeoutSec:       1,
		SubscriptionNotifications: true,
	}
	require.NoError(testingT, application.ensureRequiredConfiguration(config))
}

func TestLogAdministratorWarningEmitsWhenMissing(testingT *testing.T) {
	observedCore, observedLogs := observer.New(zap.WarnLevel)
	logger := zap.New(observedCore)

	application := NewServerApplication()
	application.logAdministratorWarning(logger, ServerConfig{})
	require.Equal(testingT, 1, observedLogs.Len())

	observedLogs.TakeAll()
	application.logAdministratorWarning(logger, ServerConfig{AdminEmailAddresses: []string{"admin@example.com"}})
	require.Equal(testingT, 0, observedLogs.Len())
}

func TestRunCommandUsesServerRunner(testingT *testing.T) {
	listener := startPinguinServer(testingT)
	setRequiredEnvironment(testingT, testPinguinAddress)

	application := NewServerApplication()
	application.WithPinguinDialer(createPinguinDialer(listener))
	command, commandErr := application.Command()
	require.NoError(testingT, commandErr)

	var runnerCalls int
	application.WithServerRunner(func(*http.Server) error {
		runnerCalls++
		return http.ErrServerClosed
	})

	runErr := application.runCommand(command, nil)
	require.NoError(testingT, runErr)
	require.Equal(testingT, 1, runnerCalls)
}

func TestRunCommandWebModeSkipsDatabaseOpener(testingT *testing.T) {
	testingT.Setenv(environmentKeyApplicationAddress, testServerAddress)
	testingT.Setenv(environmentKeyServeMode, string(ServeModeWeb))
	testingT.Setenv(environmentKeyGoogleClientID, testGoogleClientIDValue)
	testingT.Setenv(environmentKeyTauthBaseURL, testTauthBaseURLValue)
	testingT.Setenv(environmentKeyTauthTenantID, testTauthTenantIDValue)
	testingT.Setenv(environmentKeyTauthSigningKey, testTauthSigningKeyValue)
	testingT.Setenv(environmentKeyTauthSessionCookie, testTauthCookieNameValue)
	testingT.Setenv(environmentKeyPublicBaseURL, testPublicBaseURLValue)

	application := NewServerApplication()
	command, commandErr := application.Command()
	require.NoError(testingT, commandErr)

	databaseOpenerCalls := 0
	application.WithDatabaseOpener(func(storage.Config) (*gorm.DB, error) {
		databaseOpenerCalls++
		return nil, errors.New(testDatabaseOpenerMessage)
	})

	var runnerCalls int
	application.WithServerRunner(func(*http.Server) error {
		runnerCalls++
		return http.ErrServerClosed
	})

	runErr := application.runCommand(command, nil)
	require.NoError(testingT, runErr)
	require.Equal(testingT, 0, databaseOpenerCalls)
	require.Equal(testingT, 1, runnerCalls)
}

func TestRunCommandAPIModeUsesServerRunner(testingT *testing.T) {
	listener := startPinguinServer(testingT)
	setRequiredEnvironment(testingT, testPinguinAddress)
	testingT.Setenv(environmentKeyServeMode, string(ServeModeAPI))

	application := NewServerApplication()
	application.WithPinguinDialer(createPinguinDialer(listener))
	command, commandErr := application.Command()
	require.NoError(testingT, commandErr)

	var runnerCalls int
	application.WithServerRunner(func(*http.Server) error {
		runnerCalls++
		return http.ErrServerClosed
	})

	runErr := application.runCommand(command, nil)
	require.NoError(testingT, runErr)
	require.Equal(testingT, 1, runnerCalls)
}

func TestRunCommandReportsMissingConfiguration(testingT *testing.T) {
	application := NewServerApplication()
	command, commandErr := application.Command()
	require.NoError(testingT, commandErr)

	testingT.Setenv(environmentKeyApplicationAddress, "")
	testingT.Setenv(environmentKeyDatabaseDriverName, "")
	testingT.Setenv(environmentKeyDatabaseDataSource, "")
	testingT.Setenv(environmentKeyGoogleClientID, "")
	testingT.Setenv(environmentKeySessionSecret, "")
	testingT.Setenv(environmentKeyTauthBaseURL, "")
	testingT.Setenv(environmentKeyTauthTenantID, "")
	testingT.Setenv(environmentKeyTauthSigningKey, "")
	testingT.Setenv(environmentKeyPublicBaseURL, "")
	testingT.Setenv(environmentKeyPinguinAddress, "")
	testingT.Setenv(environmentKeyPinguinAuthToken, "")
	testingT.Setenv(environmentKeyPinguinTenantID, "")
	testingT.Setenv(environmentKeyPinguinConnTimeout, "0")
	testingT.Setenv(environmentKeyPinguinOpTimeout, "0")

	runErr := application.runCommand(command, nil)
	require.Error(testingT, runErr)
	require.ErrorContains(testingT, runErr, missingConfigurationMessage)
}

func TestRunCommandRejectsInvalidServeMode(testingT *testing.T) {
	application := NewServerApplication()
	command, commandErr := application.Command()
	require.NoError(testingT, commandErr)

	testingT.Setenv(environmentKeyServeMode, "invalid")
	runErr := application.runCommand(command, nil)
	require.Error(testingT, runErr)
	require.ErrorIs(testingT, runErr, ErrInvalidServeMode)
}

func TestRunCommandRejectsArguments(testingT *testing.T) {
	application := NewServerApplication()
	command, commandErr := application.Command()
	require.NoError(testingT, commandErr)

	runErr := application.runCommand(command, []string{"unexpected"})
	require.Error(testingT, runErr)
}

func TestServerRunnerCanBeOverridden(testingT *testing.T) {
	application := NewServerApplication()
	var runnerCalls int
	application.WithServerRunner(func(*http.Server) error {
		runnerCalls++
		return nil
	})

	server := &http.Server{
		Addr:              testServerAddress,
		ReadHeaderTimeout: time.Second,
	}
	runErr := application.serverRunner(server)
	require.NoError(testingT, runErr)
	require.Equal(testingT, 1, runnerCalls)
}

func TestCommandReportsInvalidEnvironmentConfiguration(testingT *testing.T) {
	application := NewServerApplication()
	testingT.Setenv(environmentKeyPinguinConnTimeout, testInvalidTimeoutValue)

	command, commandErr := application.Command()
	require.Error(testingT, commandErr)
	require.Nil(testingT, command)
}

func TestDefaultServerRunnerHandlesServerClose(testingT *testing.T) {
	application := NewServerApplication()
	server := &http.Server{
		Addr:              testServerAddress,
		ReadHeaderTimeout: time.Second,
	}

	closeDone := make(chan struct{})
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = server.Close()
		close(closeDone)
	}()

	runErr := application.serverRunner(server)
	if runErr != nil && errors.Is(runErr, syscall.EPERM) {
		testingT.Skip("server listen not permitted in sandbox")
	}
	require.ErrorIs(testingT, runErr, http.ErrServerClosed)
	<-closeDone
}
