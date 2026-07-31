package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	testComposeFileName        = "docker-compose.yml"
	testLoopAwareService       = "loopaware"
	testPinguinService         = "pinguin"
	testTauthService           = "tauth"
	testLoopAwareEnvFile       = "loopaware.env"
	testPinguinEnvFile         = "pinguin.env"
	testConfigDirectory        = "config"
	testConfigTemplateFile     = "config.yml"
	testLoopAwareConfigFile    = "config.loopaware.yml"
	testPlaceholderMissingKey  = "MISSING_VALUE"
	testLoopAwareHostPort      = "8080"
	testLoopAwareContainerPort = "8080"
	testPinguinContainerPort   = "50051"
	testTauthContainerPort     = "8081"
	testGoogleClientValue      = "client"
	testSessionSecretValue     = "session-secret"
	testCookieNameValue        = "app_session"
	testPublicBaseURLValue     = "http://example.com"
	testPinguinAddressValue    = "pinguin:50051"
	testTenantValue            = "tenant"
	testAuthTokenValue         = "token"
	testSigningKeyValue        = "signing"
	testSharedSigningKeyValue  = "shared"
	testTauthBaseURLValue      = "http://tauth:8080"
	testNonNumericHostPortMap  = "80a0:3000"
)

func validLoopAwareRuntimeConfigYAML() string {
	return strings.Join([]string{
		"server:",
		"  address: \"${APP_ADDR}\"",
		"  public_base_url: \"${PUBLIC_BASE_URL}\"",
		"database:",
		"  driver: \"${DB_DRIVER}\"",
		"  dsn: \"${DB_DSN}\"",
		"auth:",
		"  session_secret: \"${SESSION_SECRET}\"",
		"  tauth:",
		"    base_url: \"${TAUTH_BASE_URL}\"",
		"    tenant_id: \"${TAUTH_TENANT_ID}\"",
		"    jwt_signing_key: \"${TAUTH_JWT_SIGNING_KEY}\"",
		"    session_cookie_name: \"${TAUTH_SESSION_COOKIE_NAME}\"",
		"pinguin:",
		"  address: \"${PINGUIN_ADDR}\"",
		"  auth_token: \"${PINGUIN_AUTH_TOKEN}\"",
		"  tenant_id: \"${PINGUIN_TENANT_ID}\"",
		"  connection_timeout_seconds: 5",
		"  operation_timeout_seconds: 30",
		"notifications:",
		"  subscription_enabled: true",
		"  traffic_report_emails_enabled: true",
		"admins:",
		"  - admin@example.com",
		"",
	}, "\n")
}

func writeLoopAwareRuntimeConfig(testingT *testing.T, baseDirectory string) string {
	configDirectory := filepath.Join(baseDirectory, testConfigDirectory)
	require.NoError(testingT, os.MkdirAll(configDirectory, 0o755))
	configPath := filepath.Join(configDirectory, testLoopAwareConfigFile)
	require.NoError(testingT, os.WriteFile(configPath, []byte(validLoopAwareRuntimeConfigYAML()), 0o600))
	return "./" + testConfigDirectory + "/" + testLoopAwareConfigFile
}

func loopAwareRuntimeConfigVolume(testingT *testing.T, baseDirectory string) string {
	hostPath := writeLoopAwareRuntimeConfig(testingT, baseDirectory)
	return hostPath + ":" + loopAwareRuntimeConfigContainerPath + ":ro"
}

func loopAwareRuntimeEnvironmentLines() []string {
	return []string{
		"APP_ADDR=:8080",
		"DB_DRIVER=sqlite",
		"DB_DSN=file:/app/data/loopaware.sqlite?_foreign_keys=on",
		"SESSION_SECRET=" + testSessionSecretValue,
		"TAUTH_BASE_URL=" + testTauthBaseURLValue,
		"TAUTH_TENANT_ID=" + testTenantValue,
		"TAUTH_JWT_SIGNING_KEY=" + testSigningKeyValue,
		"TAUTH_SESSION_COOKIE_NAME=" + testCookieNameValue,
		"PUBLIC_BASE_URL=" + testPublicBaseURLValue,
		"PINGUIN_ADDR=" + testPinguinAddressValue,
		"PINGUIN_AUTH_TOKEN=" + testAuthTokenValue,
		"PINGUIN_TENANT_ID=" + testTenantValue,
	}
}

func TestStringListUnmarshalYAML(testingT *testing.T) {
	testCases := []struct {
		name     string
		inputYML string
		expected []string
		hasError bool
	}{
		{
			name:     "scalar value",
			inputYML: "value",
			expected: []string{"value"},
		},
		{
			name:     "sequence values",
			inputYML: "- first\n- second\n",
			expected: []string{"first", "second"},
		},
		{
			name:     "mapping unsupported",
			inputYML: "key: value",
			hasError: true,
		},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			var target stringList
			unmarshalErr := yaml.Unmarshal([]byte(testCase.inputYML), &target)
			if testCase.hasError {
				require.Error(testingT, unmarshalErr)
				return
			}
			require.NoError(testingT, unmarshalErr)
			require.Equal(testingT, testCase.expected, []string(target))
		})
	}
}

func TestEnvironmentMapUnmarshalYAML(testingT *testing.T) {
	testCases := []struct {
		name     string
		inputYML string
		expected map[string]string
		hasError bool
	}{
		{
			name:     "mapping",
			inputYML: "KEY_ONE: value\nKEY_TWO: value2\n",
			expected: map[string]string{"KEY_ONE": "value", "KEY_TWO": "value2"},
		},
		{
			name:     "sequence",
			inputYML: "- KEY_ONE=value\n- KEY_TWO=value2\n",
			expected: map[string]string{"KEY_ONE": "value", "KEY_TWO": "value2"},
		},
		{
			name:     "scalar unsupported",
			inputYML: "value",
			hasError: true,
		},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			var target environmentMap
			unmarshalErr := yaml.Unmarshal([]byte(testCase.inputYML), &target)
			if testCase.hasError {
				require.Error(testingT, unmarshalErr)
				return
			}
			require.NoError(testingT, unmarshalErr)
			require.Equal(testingT, testCase.expected, map[string]string(target))
		})
	}
}

func TestParseDotEnvDetectsDuplicates(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	envPath := filepath.Join(tempDirectory, testLoopAwareEnvFile)
	envContent := "PINGUIN_ADDR=first\nPINGUIN_ADDR=second\n# comment\nKEY=value\n"
	require.NoError(testingT, os.WriteFile(envPath, []byte(envContent), 0o600))

	values, duplicates, parseErr := parseDotEnv(envPath)
	require.NoError(testingT, parseErr)
	require.Equal(testingT, "second", values["PINGUIN_ADDR"])
	require.Contains(testingT, duplicates, "PINGUIN_ADDR")
}

func TestResolveConfigTemplates(testingT *testing.T) {
	volumes := []string{
		"./config/config.yml:/config/config.yml",
		"./config/config.yml:/config/config.yml:ro",
		"",
		"invalid",
	}
	result := resolveConfigTemplates(".", volumes)
	require.Equal(testingT, []string{filepath.Clean("./config/config.yml")}, result)
}

func TestExtractPlaceholders(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	templatePath := filepath.Join(tempDirectory, testConfigTemplateFile)
	templateContent := "value=${FIRST}\nvalue=${SECOND}\nvalue=${FIRST}\n"
	require.NoError(testingT, os.WriteFile(templatePath, []byte(templateContent), 0o600))

	placeholders, extractErr := extractPlaceholders(templatePath)
	require.NoError(testingT, extractErr)
	require.Equal(testingT, []string{"FIRST", "SECOND"}, placeholders)
}

func TestRunAuditReportsErrorsForMissingEnvironment(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	configDirectory := filepath.Join(tempDirectory, testConfigDirectory)
	require.NoError(testingT, os.MkdirAll(configDirectory, 0o755))

	templatePath := filepath.Join(configDirectory, testConfigTemplateFile)
	templateContent := "token=${" + testPlaceholderMissingKey + "}\n"
	require.NoError(testingT, os.WriteFile(templatePath, []byte(templateContent), 0o600))

	loopAwareEnvPath := filepath.Join(tempDirectory, testLoopAwareEnvFile)
	loopAwareEnvironmentLines := loopAwareRuntimeEnvironmentLines()
	loopAwareEnvironmentLines[3] = "SESSION_SECRET="
	loopAwareEnvironmentLines[9] = "PINGUIN_ADDR=pinguin:50051"
	loopAwareEnvironmentLines = append(loopAwareEnvironmentLines, "PINGUIN_ADDR=duplicate", "")
	loopAwareEnv := strings.Join(loopAwareEnvironmentLines, "\n")
	require.NoError(testingT, os.WriteFile(loopAwareEnvPath, []byte(loopAwareEnv), 0o600))

	pinguinEnvPath := filepath.Join(tempDirectory, testPinguinEnvFile)
	pinguinEnv := strings.Join([]string{
		"TAUTH_SIGNING_KEY=" + testSharedSigningKeyValue,
		"LOOPAWARE_LOCAL_GOOGLE_CLIENT_ID=" + testGoogleClientValue,
		"GRPC_AUTH_TOKEN=" + testAuthTokenValue,
		"PINGUIN_TENANT_ID=" + testTenantValue,
		"",
	}, "\n")
	require.NoError(testingT, os.WriteFile(pinguinEnvPath, []byte(pinguinEnv), 0o600))

	composePath := filepath.Join(tempDirectory, testComposeFileName)
	loopAwareConfigVolume := loopAwareRuntimeConfigVolume(testingT, tempDirectory)
	composeContent := strings.Join([]string{
		"services:",
		"  " + testLoopAwareService + ":",
		"    env_file:",
		"      - " + testLoopAwareEnvFile,
		"    environment:",
		"      PINGUIN_TENANT_ID: " + testTenantValue,
		"    volumes:",
		"      - ./config/" + testConfigTemplateFile + ":/config/config.yml",
		"      - " + loopAwareConfigVolume,
		"    ports:",
		"      - \"" + testLoopAwareHostPort + ":" + testLoopAwareContainerPort + "\"",
		"  " + testPinguinService + ":",
		"    env_file:",
		"      - " + testPinguinEnvFile,
		"    ports:",
		"      - \"" + testLoopAwareHostPort + ":" + testPinguinContainerPort + "\"",
		"  " + testTauthService + ":",
		"    environment:",
		"      TAUTH_LOOPAWARE_JWT_SIGNING_KEY: " + testSharedSigningKeyValue,
		"      TAUTH_LOOPAWARE_GOOGLE_WEB_CLIENT_ID: " + testGoogleClientValue,
		"    ports:",
		"      - \"8081:" + testTauthContainerPort + "\"",
		"",
	}, "\n")
	require.NoError(testingT, os.WriteFile(composePath, []byte(composeContent), 0o600))

	result := runAudit(composePath)
	require.False(testingT, result.ok())
	combinedErrors := strings.Join(result.errors, " ")
	require.Contains(testingT, combinedErrors, "env_file "+testLoopAwareEnvFile+" defines PINGUIN_ADDR more than once")
	require.Contains(testingT, combinedErrors, "auth.session_secret")
	require.Contains(testingT, combinedErrors, "references ${"+testPlaceholderMissingKey+"} but "+testPlaceholderMissingKey+" is not defined")
	require.Contains(testingT, combinedErrors, "host port "+testLoopAwareHostPort+" is published by both")
}

func TestRunAuditUsesDeclaredAuditFixturesWhenRuntimeEnvFilesAreMissing(testingT *testing.T) {
	tempDirectory := testingT.TempDir()

	loopAwareExamplePath := filepath.Join(tempDirectory, testLoopAwareEnvFile+".example")
	loopAwareExample := "EXAMPLE_MUST_NOT_BE_LOADED=true\n"
	require.NoError(testingT, os.WriteFile(loopAwareExamplePath, []byte(loopAwareExample), 0o600))
	loopAwareAuditFile := "loopaware.audit.env"
	loopAwareAudit := strings.Join(append(loopAwareRuntimeEnvironmentLines(), ""), "\n")
	require.NoError(testingT, os.WriteFile(filepath.Join(tempDirectory, loopAwareAuditFile), []byte(loopAwareAudit), 0o600))

	pinguinExamplePath := filepath.Join(tempDirectory, testPinguinEnvFile+".example")
	require.NoError(testingT, os.WriteFile(pinguinExamplePath, []byte("EXAMPLE_MUST_NOT_BE_LOADED=true\n"), 0o600))
	pinguinAuditFile := "pinguin.audit.env"
	pinguinAudit := strings.Join([]string{
		"TAUTH_SIGNING_KEY=" + testSigningKeyValue,
		"LOOPAWARE_LOCAL_GOOGLE_CLIENT_ID=" + testGoogleClientValue,
		"GRPC_AUTH_TOKEN=" + testAuthTokenValue,
		"",
	}, "\n")
	require.NoError(testingT, os.WriteFile(filepath.Join(tempDirectory, pinguinAuditFile), []byte(pinguinAudit), 0o600))

	composePath := filepath.Join(tempDirectory, testComposeFileName)
	loopAwareConfigVolume := loopAwareRuntimeConfigVolume(testingT, tempDirectory)
	composeContent := strings.Join([]string{
		"services:",
		"  " + testLoopAwareService + ":",
		"    env_file:",
		"      - " + testLoopAwareEnvFile,
		"    x-config-audit-env-file: " + loopAwareAuditFile,
		"    volumes:",
		"      - " + loopAwareConfigVolume,
		"  " + testPinguinService + ":",
		"    env_file:",
		"      - " + testPinguinEnvFile,
		"    x-config-audit-env-file: " + pinguinAuditFile,
		"  " + testTauthService + ":",
		"    environment:",
		"      TAUTH_TENANT_ID_LOOPAWARE: " + testTenantValue,
		"      TAUTH_TENANT_JWT_SIGNING_KEY_LOOPAWARE: " + testSigningKeyValue,
		"      TAUTH_TENANT_SESSION_COOKIE_NAME_LOOPAWARE: " + testCookieNameValue,
		"      TAUTH_LOOPAWARE_JWT_SIGNING_KEY: " + testSigningKeyValue,
		"      TAUTH_LOOPAWARE_GOOGLE_WEB_CLIENT_ID: " + testGoogleClientValue,
		"",
	}, "\n")
	require.NoError(testingT, os.WriteFile(composePath, []byte(composeContent), 0o600))

	result := runAudit(composePath)
	require.True(testingT, result.ok())
	require.Empty(testingT, result.warnings)
	require.Empty(testingT, result.errors)
}

func TestRunAuditAllowsCrossServiceOperatorValueDrift(testingT *testing.T) {
	tempDirectory := testingT.TempDir()

	loopAwareEnvPath := filepath.Join(tempDirectory, testLoopAwareEnvFile)
	loopAwareEnvironmentLines := loopAwareRuntimeEnvironmentLines()
	loopAwareEnvironmentLines[5] = "TAUTH_TENANT_ID=loopaware"
	loopAwareEnvironmentLines[6] = "TAUTH_JWT_SIGNING_KEY=loopaware-owned-signing-key"
	loopAwareEnvironmentLines[7] = "TAUTH_SESSION_COOKIE_NAME=loopaware-owned-session"
	loopAwareEnvironmentLines[10] = "PINGUIN_AUTH_TOKEN=loopaware-owned-pinguin-token"
	loopAwareEnv := strings.Join(append(loopAwareEnvironmentLines, ""), "\n")
	require.NoError(testingT, os.WriteFile(loopAwareEnvPath, []byte(loopAwareEnv), 0o600))

	pinguinEnvPath := filepath.Join(tempDirectory, testPinguinEnvFile)
	pinguinEnv := strings.Join([]string{
		"TAUTH_SIGNING_KEY=pinguin-owned-signing-key",
		"LOOPAWARE_LOCAL_GOOGLE_CLIENT_ID=" + testGoogleClientValue,
		"GRPC_AUTH_TOKEN=pinguin-owned-token",
		"",
	}, "\n")
	require.NoError(testingT, os.WriteFile(pinguinEnvPath, []byte(pinguinEnv), 0o600))

	tauthEnvPath := filepath.Join(tempDirectory, testTauthService+".env")
	tauthEnv := strings.Join([]string{
		"TAUTH_TENANT_ID_LOOPAWARE=loopaware",
		"TAUTH_TENANT_JWT_SIGNING_KEY_LOOPAWARE=tauth-owned-signing-key",
		"TAUTH_TENANT_SESSION_COOKIE_NAME_LOOPAWARE=tauth-owned-session",
		"TAUTH_LOOPAWARE_JWT_SIGNING_KEY=tauth-owned-pinguin-signing-key",
		"",
	}, "\n")
	require.NoError(testingT, os.WriteFile(tauthEnvPath, []byte(tauthEnv), 0o600))

	composePath := filepath.Join(tempDirectory, testComposeFileName)
	loopAwareConfigVolume := loopAwareRuntimeConfigVolume(testingT, tempDirectory)
	composeContent := strings.Join([]string{
		"services:",
		"  " + testLoopAwareService + ":",
		"    env_file:",
		"      - " + testLoopAwareEnvFile,
		"    volumes:",
		"      - " + loopAwareConfigVolume,
		"  " + testPinguinService + ":",
		"    env_file:",
		"      - " + testPinguinEnvFile,
		"  " + testTauthService + ":",
		"    env_file:",
		"      - " + testTauthService + ".env",
		"",
	}, "\n")
	require.NoError(testingT, os.WriteFile(composePath, []byte(composeContent), 0o600))

	result := runAudit(composePath)
	require.True(testingT, result.ok())
	require.Empty(testingT, result.warnings)
	require.Empty(testingT, result.errors)
}

func TestRunAuditCommandSuccess(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	composePath := filepath.Join(tempDirectory, testComposeFileName)
	loopAwareConfigVolume := loopAwareRuntimeConfigVolume(testingT, tempDirectory)
	composeLines := []string{
		"services:",
		"  " + testLoopAwareService + ":",
		"    environment:",
	}
	for _, environmentLine := range loopAwareRuntimeEnvironmentLines() {
		composeLines = append(composeLines, "      - "+environmentLine)
	}
	composeLines = append(composeLines,
		"    volumes:",
		"      - "+loopAwareConfigVolume,
		"",
	)
	composeContent := strings.Join(composeLines, "\n")
	require.NoError(testingT, os.WriteFile(composePath, []byte(composeContent), 0o600))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runAuditCommand(composePath, &stdout, &stderr)
	require.Equal(testingT, 0, exitCode)
	require.Contains(testingT, stdout.String(), "config-audit OK")
	require.Empty(testingT, stderr.String())
}

func TestRunAuditCommandReportsMissingComposeFile(testingT *testing.T) {
	composePath := filepath.Join(testingT.TempDir(), "missing-compose.yml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runAuditCommand(composePath, &stdout, &stderr)
	require.Equal(testingT, 1, exitCode)
	require.Contains(testingT, stderr.String(), "read compose file")
	require.Contains(testingT, stderr.String(), "config-audit failed")
}

func TestParseHostPort(testingT *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectPort  string
		expectValid bool
	}{
		{
			name:        "valid mapping",
			input:       "\"8080:3000\"",
			expectPort:  testLoopAwareHostPort,
			expectValid: true,
		},
		{
			name:        "invalid mapping",
			input:       "invalid",
			expectValid: false,
		},
		{
			name:        "missing port",
			input:       ":3000",
			expectValid: false,
		},
		{
			name:        "non-numeric host",
			input:       testNonNumericHostPortMap,
			expectValid: false,
		},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			port, isValid := parseHostPort(testCase.input)
			require.Equal(testingT, testCase.expectValid, isValid)
			if testCase.expectValid {
				require.Equal(testingT, testCase.expectPort, port)
			}
		})
	}
}

func TestCheckLocalPortMatchesRecordsAllowedAndBlocked(testingT *testing.T) {
	allowedPorts := map[string]struct{}{testLoopAwareHostPort: {}}
	result := auditResult{}
	checkLocalPortMatches("file.js", 1, "http://localhost:"+testLoopAwareHostPort, allowedPorts, &result)
	require.NotEmpty(testingT, result.warnings)

	result = auditResult{}
	checkLocalPortMatches("file.js", 1, "http://localhost:9999", allowedPorts, &result)
	require.NotEmpty(testingT, result.errors)
}

func TestScanAssetRootReportsLocalhostPorts(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	assetPath := filepath.Join(tempDirectory, "test.html")
	content := "http://localhost:" + testLoopAwareHostPort + "\nhttp://localhost:9999\n"
	require.NoError(testingT, os.WriteFile(assetPath, []byte(content), 0o600))

	allowedPorts := map[string]struct{}{testLoopAwareHostPort: {}}
	result := auditResult{}
	scanErr := scanAssetRoot(tempDirectory, allowedPorts, &result)
	require.NoError(testingT, scanErr)
	require.NotEmpty(testingT, result.warnings)
	require.NotEmpty(testingT, result.errors)
}

func TestScanAssetFileReportsMissingFile(testingT *testing.T) {
	missingPath := filepath.Join(testingT.TempDir(), "missing.js")
	result := auditResult{}
	scanErr := scanAssetFile(missingPath, map[string]struct{}{}, &result)
	require.Error(testingT, scanErr)
}
