package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

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
	testAuthTokenValue              = "test-auth-token"
	testTenantValue                 = "test-tenant"
	testSessionSecretValue          = "test-session-secret"
	testTauthBaseURLValue           = "http://tauth.test"
	testTauthTenantIDValue          = "tenant-id"
	testTauthSigningKeyValue        = "signing-key"
	testTauthCookieNameValue        = "app_session"
	testPublicBaseURLValue          = "http://localhost:8080"
	testPinguinAddress              = "bufnet"
	testDatabaseDSNValue            = "file:server-test?mode=memory&cache=shared&_foreign_keys=on"
	testDatabaseOpenerMessage       = "database opener error"
	testBufferSize                  = 1024 * 1024
	testServerAddress               = "127.0.0.1:0"
	testResolveOriginEmpty          = "   "
	testResolveOriginMalformed      = "http://%zz"
	testResolveOriginMissing        = "example.com"
	testResolveOriginValid          = "https://example.com/path"
	testInvalidConfigContents       = ": [}"
	testEnvironmentSessionSecretKey = "LOOPAWARE_TEST_SESSION_SECRET"
	testEnvironmentTrafficEmailsKey = "LOOPAWARE_TEST_TRAFFIC_REPORT_EMAILS"
	testEnvironmentTimeoutKey       = "LOOPAWARE_TEST_PINGUIN_CONNECTION_TIMEOUT"
	testEnvironmentMissingKey       = "LOOPAWARE_TEST_MISSING_CONFIG_VALUE"
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

func validServerConfigYAML(pinguinAddress string) string {
	return strings.Join([]string{
		"server:",
		"  address: \"127.0.0.1:0\"",
		"  public_base_url: \"" + testPublicBaseURLValue + "\"",
		"database:",
		"  driver: \"" + storage.DriverNameSQLite + "\"",
		"  dsn: \"" + testDatabaseDSNValue + "\"",
		"auth:",
		"  session_secret: \"" + testSessionSecretValue + "\"",
		"  tauth:",
		"    base_url: \"" + testTauthBaseURLValue + "\"",
		"    tenant_id: \"" + testTauthTenantIDValue + "\"",
		"    jwt_signing_key: \"" + testTauthSigningKeyValue + "\"",
		"    session_cookie_name: \"" + testTauthCookieNameValue + "\"",
		"pinguin:",
		"  address: \"" + pinguinAddress + "\"",
		"  auth_token: \"" + testAuthTokenValue + "\"",
		"  tenant_id: \"" + testTenantValue + "\"",
		"  connection_timeout_seconds: 1",
		"  operation_timeout_seconds: 1",
		"notifications:",
		"  subscription_enabled: true",
		"  traffic_report_emails_enabled: true",
		"admins:",
		"  - admin@example.com",
		"  - owner@example.com",
		"",
	}, "\n")
}

func writeServerConfig(testingT *testing.T, payload string) string {
	configPath := filepath.Join(testingT.TempDir(), "config.loopaware.yml")
	require.NoError(testingT, os.WriteFile(configPath, []byte(payload), 0o600))
	return configPath
}

func TestNewServerApplicationHasDefaults(testingT *testing.T) {
	application := NewServerApplication()
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

func TestCommandExposesOnlyConfigFlag(testingT *testing.T) {
	application := NewServerApplication()
	command, commandErr := application.Command()
	require.NoError(testingT, commandErr)

	require.NotNil(testingT, command.Flags().Lookup(flagNameConfigFile))
	require.Nil(testingT, command.Flags().Lookup("app-addr"))
	require.Nil(testingT, command.Flags().Lookup("pinguin-auth-token"))
}

func TestResolveOriginValidatesInput(testingT *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedOrigin string
		expectError    bool
	}{
		{
			name:        "empty",
			input:       testResolveOriginEmpty,
			expectError: true,
		},
		{
			name:        "malformed",
			input:       testResolveOriginMalformed,
			expectError: true,
		},
		{
			name:        "missing_scheme",
			input:       testResolveOriginMissing,
			expectError: true,
		},
		{
			name:           "valid",
			input:          testResolveOriginValid,
			expectedOrigin: "https://example.com",
			expectError:    false,
		},
	}

	for _, testCase := range testCases {
		origin, originErr := resolveOrigin(testCase.input)
		if testCase.expectError {
			require.Error(testingT, originErr, testCase.name)
			continue
		}
		require.NoError(testingT, originErr, testCase.name)
		require.Equal(testingT, testCase.expectedOrigin, origin, testCase.name)
	}
}

func TestLoadServerConfigExpandsShellValuesOnlyDuringConfigParse(testingT *testing.T) {
	application := NewServerApplication()
	configPayload := strings.ReplaceAll(validServerConfigYAML(testPinguinAddress), testSessionSecretValue, "${"+testEnvironmentSessionSecretKey+"}")
	configPayload = strings.ReplaceAll(configPayload, "connection_timeout_seconds: 1", "connection_timeout_seconds: ${"+testEnvironmentTimeoutKey+"}")
	configPayload = strings.ReplaceAll(configPayload, "traffic_report_emails_enabled: true", "traffic_report_emails_enabled: ${"+testEnvironmentTrafficEmailsKey+"}")
	configPath := writeServerConfig(testingT, configPayload)

	testingT.Setenv(testEnvironmentSessionSecretKey, testSessionSecretValue)
	testingT.Setenv(testEnvironmentTimeoutKey, "2")
	testingT.Setenv(testEnvironmentTrafficEmailsKey, "false")

	config, loadErr := application.loadServerConfig(configPath)
	require.NoError(testingT, loadErr)
	require.Equal(testingT, testSessionSecretValue, config.SessionSecret)
	require.Equal(testingT, 2, config.PinguinConnTimeoutSec)
	require.False(testingT, config.TrafficReportEmails)
}

func TestLoadServerConfigReportsMissingInterpolation(testingT *testing.T) {
	application := NewServerApplication()
	configPayload := strings.ReplaceAll(validServerConfigYAML(testPinguinAddress), testSessionSecretValue, "${"+testEnvironmentMissingKey+"}")
	configPath := writeServerConfig(testingT, configPayload)

	_, loadErr := application.loadServerConfig(configPath)
	require.Error(testingT, loadErr)
	require.ErrorContains(testingT, loadErr, testEnvironmentMissingKey)
}

func TestLoadServerConfigRejectsUnknownYAMLFields(testingT *testing.T) {
	application := NewServerApplication()
	configPayload := strings.Replace(validServerConfigYAML(testPinguinAddress), "server:\n", "server:\n  unknown: true\n", 1)
	configPath := writeServerConfig(testingT, configPayload)

	_, loadErr := application.loadServerConfig(configPath)
	require.Error(testingT, loadErr)
	require.ErrorContains(testingT, loadErr, "field unknown not found")
}

func TestLoadServerConfigRejectsMissingRequiredValues(testingT *testing.T) {
	application := NewServerApplication()
	configPayload := strings.ReplaceAll(validServerConfigYAML(testPinguinAddress), "auth_token: \""+testAuthTokenValue+"\"", "auth_token: \"\"")
	configPath := writeServerConfig(testingT, configPayload)

	_, loadErr := application.loadServerConfig(configPath)
	require.Error(testingT, loadErr)
	require.ErrorContains(testingT, loadErr, "pinguin.auth_token")
}

func TestLoadServerConfigDoesNotUseAdminEnvironmentOverride(testingT *testing.T) {
	application := NewServerApplication()
	configPath := writeServerConfig(testingT, validServerConfigYAML(testPinguinAddress))
	testingT.Setenv("ADMINS", "env-admin@example.com")

	config, loadErr := application.loadServerConfig(configPath)
	require.NoError(testingT, loadErr)
	require.Equal(testingT, []string{"admin@example.com", "owner@example.com"}, config.AdminEmailAddresses)
}

func TestLoadServerConfigDoesNotUseSharedPinguinAuthAlias(testingT *testing.T) {
	application := NewServerApplication()
	configPayload := strings.ReplaceAll(validServerConfigYAML(testPinguinAddress), "auth_token: \""+testAuthTokenValue+"\"", "auth_token: \"\"")
	configPath := writeServerConfig(testingT, configPayload)
	testingT.Setenv("GRPC_AUTH_TOKEN", testAuthTokenValue)

	_, loadErr := application.loadServerConfig(configPath)
	require.Error(testingT, loadErr)
	require.ErrorContains(testingT, loadErr, "pinguin.auth_token")
}

func TestLoadServerConfigRejectsBrowserRuntimeConfigShape(testingT *testing.T) {
	application := NewServerApplication()
	configPath := writeServerConfig(testingT, "services:\n  tauthOrigin: http://localhost:8082\n")

	_, loadErr := application.loadServerConfig(configPath)
	require.Error(testingT, loadErr)
	require.ErrorContains(testingT, loadErr, "field services not found")
}

func TestLoadServerConfigReportsConfigurationFileError(testingT *testing.T) {
	application := NewServerApplication()
	configPath := writeServerConfig(testingT, testInvalidConfigContents)

	_, loadErr := application.loadServerConfig(configPath)
	require.Error(testingT, loadErr)
	require.ErrorContains(testingT, loadErr, configurationFileLoadError)
}

func TestRunCommandReportsConfigurationFileError(testingT *testing.T) {
	application := NewServerApplication()
	command, commandErr := application.Command()
	require.NoError(testingT, commandErr)

	configPath := writeServerConfig(testingT, testInvalidConfigContents)
	require.NoError(testingT, command.Flags().Set(flagNameConfigFile, configPath))

	runErr := application.runCommand(command, nil)
	require.Error(testingT, runErr)
	require.ErrorContains(testingT, runErr, configurationFileLoadError)
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
	configPath := writeServerConfig(testingT, validServerConfigYAML(testPinguinAddress))

	application := NewServerApplication()
	application.WithPinguinDialer(createPinguinDialer(listener))
	command, commandErr := application.Command()
	require.NoError(testingT, commandErr)
	require.NoError(testingT, command.Flags().Set(flagNameConfigFile, configPath))

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

	configPayload := strings.ReplaceAll(validServerConfigYAML(testPinguinAddress), "session_secret: \""+testSessionSecretValue+"\"", "session_secret: \"\"")
	configPath := writeServerConfig(testingT, configPayload)
	require.NoError(testingT, command.Flags().Set(flagNameConfigFile, configPath))

	runErr := application.runCommand(command, nil)
	require.Error(testingT, runErr)
	require.ErrorContains(testingT, runErr, "auth.session_secret")
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
