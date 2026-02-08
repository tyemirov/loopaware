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
	loopAwareEnv := strings.Join([]string{
		"SESSION_SECRET=",
		"TAUTH_BASE_URL=" + testTauthBaseURLValue,
		"TAUTH_TENANT_ID=" + testTenantValue,
		"TAUTH_JWT_SIGNING_KEY=" + testSigningKeyValue,
		"TAUTH_SESSION_COOKIE_NAME=" + testCookieNameValue,
		"PUBLIC_BASE_URL=http://localhost:8080",
		"PINGUIN_ADDR=pinguin:50051",
		"PINGUIN_ADDR=duplicate",
		"PINGUIN_AUTH_TOKEN=" + testAuthTokenValue,
		"",
	}, "\n")
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
	composeContent := strings.Join([]string{
		"services:",
		"  " + testLoopAwareService + ":",
		"    env_file:",
		"      - " + testLoopAwareEnvFile,
		"    environment:",
		"      PINGUIN_TENANT_ID: " + testTenantValue,
		"    volumes:",
		"      - ./config/" + testConfigTemplateFile + ":/config/config.yml",
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
	require.Contains(testingT, combinedErrors, "required env SESSION_SECRET is missing or empty")
	require.Contains(testingT, combinedErrors, "references ${"+testPlaceholderMissingKey+"} but "+testPlaceholderMissingKey+" is not defined")
	require.Contains(testingT, combinedErrors, "host port "+testLoopAwareHostPort+" is published by both")
}

func TestRunAuditCommandSuccess(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	composePath := filepath.Join(tempDirectory, testComposeFileName)
	composeContent := strings.Join([]string{
		"services:",
		"  " + testLoopAwareService + ":",
		"    environment:",
		"      SESSION_SECRET: " + testSessionSecretValue,
		"      TAUTH_BASE_URL: " + testTauthBaseURLValue,
		"      TAUTH_TENANT_ID: " + testTenantValue,
		"      TAUTH_JWT_SIGNING_KEY: " + testSigningKeyValue,
		"      TAUTH_SESSION_COOKIE_NAME: " + testCookieNameValue,
		"      PUBLIC_BASE_URL: " + testPublicBaseURLValue,
		"      PINGUIN_ADDR: " + testPinguinAddressValue,
		"      PINGUIN_AUTH_TOKEN: " + testAuthTokenValue,
		"      PINGUIN_TENANT_ID: " + testTenantValue,
		"",
	}, "\n")
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

func TestExpectEqualReportsWarningAndError(testingT *testing.T) {
	var result auditResult
	expectEqual("left", "", "right", "", &result)
	require.NotEmpty(testingT, result.warnings)

	result = auditResult{}
	expectEqual("left", "one", "right", "two", &result)
	require.NotEmpty(testingT, result.errors)
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
