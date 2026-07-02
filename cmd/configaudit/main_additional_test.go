package main

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/MarkoPoloResearchLab/loopaware/internal/serverconfig"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	testAssetScanFileName           = "sample.js"
	testAssetScanHostPortPrefix     = "localhost:"
	testAssetScanLocalhostNoPort    = "http://localhost"
	testAssetScanLocalhostURLPrefix = "http://localhost:"
	testAssetScanJoinSeparator      = " "
	testInvalidComposeContents      = "services: [}"
	testPlaceholderFreeTemplate     = "plain text"
	testInvalidVolumeMapping        = "invalid"
	testInvalidVolumeMissingHost    = ":/config/config.yml"
	testInvalidVolumeMissingTarget  = "./config/config.yml:"
	testAssetScanSymlinkName        = "link.js"
	testAssetScanIgnoredFileName    = "notes.txt"
	testMissingEnvFileName          = "missing.env"
	testInlineEnvironmentKey        = "PINGUIN_ADDR"
	testStringListValue             = "list-entry"
	testEnvDecodeKey                = "KEY"
	testEnvDecodeValue              = "VALUE"
	testMalformedEnvLineNoEquals    = "MALFORMED"
	testMalformedEnvLineEmptyKey    = "=value"
	testEnvScanPrefix               = "KEY="
	testEnvScanPad                  = "a"
	testEnvScanLineLength           = 70000
	testConfigTemplateContainerPath = "/config/config.yml"
	testNonTemplateVolumeMapping    = "./config/other.yml:/config/other.yml"
	testLocalPatternNoGroups        = "localhost"
	testLocalHostPortPatternEmpty   = "(localhost):(\\d*)"
	testHostPortServiceName         = "service-a"
	testEnvKeyPinguinSigning        = "TAUTH_SIGNING_KEY"
	testEnvKeyPinguinGoogleClient   = "LOOPAWARE_LOCAL_GOOGLE_CLIENT_ID"
	testEnvKeyPinguinAuthToken      = "GRPC_AUTH_TOKEN"
	testEnvKeyTauthSigning          = "TAUTH_LOOPAWARE_JWT_SIGNING_KEY"
	testEnvKeyTauthGoogleClient     = "TAUTH_LOOPAWARE_GOOGLE_WEB_CLIENT_ID"
	testEnvErrorMessageSnippet      = "no environment variables resolved"
	testTemplateVolumeMapping       = "./config/" + testConfigTemplateFile + ":" + testConfigTemplateContainerPath
)

func TestStringListUnmarshalYAMLHandlesNilAndBlank(testingT *testing.T) {
	var values stringList
	unmarshalErr := values.UnmarshalYAML(nil)
	require.NoError(testingT, unmarshalErr)
	require.Nil(testingT, []string(values))

	node := &yaml.Node{Kind: yaml.ScalarNode, Value: "   "}
	unmarshalErr = values.UnmarshalYAML(node)
	require.NoError(testingT, unmarshalErr)
	require.Nil(testingT, []string(values))
}

func TestEnvironmentMapUnmarshalYAMLHandlesNilAndSequence(testingT *testing.T) {
	var environment environmentMap
	unmarshalErr := environment.UnmarshalYAML(nil)
	require.NoError(testingT, unmarshalErr)
	require.Nil(testingT, map[string]string(environment))

	sequenceYAML := "- KEY_ONLY\n- KEY_TWO=value\n- =skipped\n- \"\"\n"
	unmarshalErr = yaml.Unmarshal([]byte(sequenceYAML), &environment)
	require.NoError(testingT, unmarshalErr)
	require.Equal(testingT, map[string]string{
		"KEY_ONLY": "",
		"KEY_TWO":  "value",
	}, map[string]string(environment))
}

func TestLoadServiceEnvironmentUsesExampleEnvFileWhenRuntimeEnvIsMissing(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	result := auditResult{}
	examplePath := filepath.Join(tempDirectory, testMissingEnvFileName+".example")
	require.NoError(testingT, os.WriteFile(examplePath, []byte("KEY=value\n"), 0o600))

	environment, hasAuditableEnvironment, loadErr := loadServiceEnvironment(tempDirectory, "app", []string{testMissingEnvFileName}, environmentMap{}, &result)
	require.NoError(testingT, loadErr)
	require.True(testingT, hasAuditableEnvironment)
	require.Equal(testingT, "value", environment["KEY"])
	require.Empty(testingT, result.errors)
	require.Empty(testingT, result.warnings)
}

func TestLoadServiceEnvironmentSkipsMissingEnvFileWithoutTrackedTemplate(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	result := auditResult{}

	environment, hasAuditableEnvironment, loadErr := loadServiceEnvironment(tempDirectory, "app", []string{testMissingEnvFileName}, environmentMap{}, &result)
	require.NoError(testingT, loadErr)
	require.False(testingT, hasAuditableEnvironment)
	require.Empty(testingT, environment)
	require.Empty(testingT, result.errors)
	require.NotEmpty(testingT, result.warnings)
}

func TestRunAuditReportsNoServices(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	composePath := filepath.Join(tempDirectory, testComposeFileName)
	require.NoError(testingT, os.WriteFile(composePath, []byte("services: {}"), 0o600))

	result := runAudit(composePath)
	require.False(testingT, result.ok())
	require.NotEmpty(testingT, result.errors)
	require.Contains(testingT, result.errors[0], "no services defined")
}

func TestParseVolumeMappingHandlesWhitespace(testingT *testing.T) {
	hostPath, containerPath, ok := parseVolumeMapping(" ./config/config.yml : /config/config.yml ")
	require.True(testingT, ok)
	require.Equal(testingT, "./config/config.yml", hostPath)
	require.Equal(testingT, "/config/config.yml", containerPath)
}

func TestScanAssetRootReturnsErrorForMissingRoot(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	missingRoot := filepath.Join(tempDirectory, "missing")

	result := auditResult{}
	scanErr := scanAssetRoot(missingRoot, map[string]struct{}{}, &result)
	require.Error(testingT, scanErr)
}

func TestCheckWebAssetLocalhostPortsScansWebRoot(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	originalDirectory, originalErr := os.Getwd()
	require.NoError(testingT, originalErr)
	require.NoError(testingT, os.Chdir(tempDirectory))
	testingT.Cleanup(func() {
		_ = os.Chdir(originalDirectory)
	})

	webPath := filepath.Join(tempDirectory, "web")
	require.NoError(testingT, os.MkdirAll(webPath, 0o755))
	require.NoError(testingT, os.WriteFile(filepath.Join(webPath, "sample.html"), []byte("http://localhost:1234"), 0o600))

	hostPortToService := map[string]string{"1234": "app"}
	result := auditResult{}
	checkWebAssetLocalhostPorts(hostPortToService, &result)
	require.NotEmpty(testingT, result.warnings)
	require.Empty(testingT, result.errors)
}

func TestCheckForbiddenLocalThirdPartyPathsRejectsMprUiCheckout(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	require.NoError(testingT, os.MkdirAll(filepath.Join(tempDirectory, "tools", "mpr-ui"), 0o755))

	result := auditResult{}
	checkForbiddenLocalThirdPartyPaths(tempDirectory, &result)
	require.NotEmpty(testingT, result.errors)
	require.Contains(testingT, result.errors[0], filepath.Join("tools", "mpr-ui"))
}

func TestCheckForbiddenLocalThirdPartyPathsAllowsMissingTools(testingT *testing.T) {
	result := auditResult{}
	checkForbiddenLocalThirdPartyPaths(testingT.TempDir(), &result)
	require.Empty(testingT, result.errors)
}

func TestParseDotEnvReportsMissingFile(testingT *testing.T) {
	missingPath := filepath.Join(testingT.TempDir(), "missing.env")
	values, duplicates, parseErr := parseDotEnv(missingPath)
	require.Error(testingT, parseErr)
	require.Nil(testingT, values)
	require.Nil(testingT, duplicates)
}

func TestExtractPlaceholdersReportsMissingFile(testingT *testing.T) {
	missingPath := filepath.Join(testingT.TempDir(), "missing.yml")
	placeholders, extractErr := extractPlaceholders(missingPath)
	require.Error(testingT, extractErr)
	require.Nil(testingT, placeholders)
}

func TestRecordLocalPortSkipsInvalidPort(testingT *testing.T) {
	result := auditResult{}
	recordLocalPort(testAssetScanFileName, 1, "abc", map[string]struct{}{}, &result)
	require.Empty(testingT, result.errors)
	require.Empty(testingT, result.warnings)
}

func TestCheckLocalPortMatchesHandlesHostPortPattern(testingT *testing.T) {
	allowedPorts := map[string]struct{}{testLoopAwareHostPort: {}}
	result := auditResult{}
	checkLocalPortMatches(testAssetScanFileName, 10, "connect to localhost:"+testLoopAwareHostPort, allowedPorts, &result)
	require.NotEmpty(testingT, result.warnings)
}

func TestCheckHostPortCollisionsDetectsDuplicate(testingT *testing.T) {
	hostPortToService := make(map[string]string)
	result := auditResult{}
	checkHostPortCollisions(testHostPortServiceName, []string{"8080:3000"}, hostPortToService, &result)
	checkHostPortCollisions("service-b", []string{"8080:4000"}, hostPortToService, &result)
	require.NotEmpty(testingT, result.errors)
}

func TestScanAssetRootRecordsLocalhostPorts(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	assetPath := filepath.Join(tempDirectory, testAssetScanFileName)
	assetLines := []string{
		testAssetScanLocalhostURLPrefix + testLoopAwareHostPort,
		testAssetScanLocalhostNoPort,
		testAssetScanHostPortPrefix + testTauthContainerPort,
	}
	require.NoError(testingT, os.WriteFile(assetPath, []byte(strings.Join(assetLines, "\n")), 0o600))

	allowedPorts := map[string]struct{}{testLoopAwareHostPort: {}}
	result := auditResult{}
	scanErr := scanAssetRoot(tempDirectory, allowedPorts, &result)
	require.NoError(testingT, scanErr)

	combinedWarnings := strings.Join(result.warnings, testAssetScanJoinSeparator)
	combinedErrors := strings.Join(result.errors, testAssetScanJoinSeparator)
	require.Contains(testingT, combinedWarnings, testLoopAwareHostPort)
	require.Contains(testingT, combinedErrors, testTauthContainerPort)
}

func TestLoadLoopAwareRuntimeConfigReportsMissingRequiredValue(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	configPath := filepath.Join(tempDirectory, testLoopAwareConfigFile)
	require.NoError(testingT, os.WriteFile(configPath, []byte(validLoopAwareRuntimeConfigYAML()), 0o600))

	environment := make(map[string]string)
	for _, environmentLine := range loopAwareRuntimeEnvironmentLines() {
		key, value, found := strings.Cut(environmentLine, "=")
		require.True(testingT, found)
		environment[key] = value
	}
	environment["SESSION_SECRET"] = ""

	_, loadErr := loadLoopAwareRuntimeConfig(configPath, environment)
	require.Error(testingT, loadErr)
	require.Contains(testingT, loadErr.Error(), "auth.session_secret")
}

func TestResolveLoopAwareRuntimeConfigUsesComposeServiceAlias(testingT *testing.T) {
	resolvedConfig, found := resolveLoopAwareRuntimeConfig(map[string]serverconfig.Config{
		"loopaware-api": {
			PinguinAuthToken: testAuthTokenValue,
		},
	})
	require.True(testingT, found)
	require.Equal(testingT, testAuthTokenValue, resolvedConfig.PinguinAuthToken)
}

func TestCheckCrossServiceInvariantsUsesComposeServiceAliases(testingT *testing.T) {
	result := auditResult{}
	checkCrossServiceInvariants(map[string]map[string]string{
		"la-pinguin": {
			testEnvKeyPinguinSigning:      "wrong-signing-key",
			testEnvKeyPinguinGoogleClient: testGoogleClientValue,
			testEnvKeyPinguinAuthToken:    testAuthTokenValue,
		},
		"la-tauth": {
			testEnvKeyTauthSigning:      testSharedSigningKeyValue,
			testEnvKeyTauthGoogleClient: testGoogleClientValue,
		},
	}, map[string]serverconfig.Config{
		"loopaware-api": {
			PinguinAuthToken: testAuthTokenValue,
		},
	}, &result)
	require.NotEmpty(testingT, result.errors)
	require.Contains(testingT, strings.Join(result.errors, " "), "invariant check failed")
}

func TestRunAuditReportsParseError(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	composePath := filepath.Join(tempDirectory, testComposeFileName)
	require.NoError(testingT, os.WriteFile(composePath, []byte(testInvalidComposeContents), 0o600))

	result := runAudit(composePath)
	require.False(testingT, result.ok())
	require.NotEmpty(testingT, result.errors)
	require.Contains(testingT, strings.Join(result.errors, " "), "parse compose file")
}

func TestExtractPlaceholdersReturnsNilWhenEmpty(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	templatePath := filepath.Join(tempDirectory, testConfigTemplateFile)
	require.NoError(testingT, os.WriteFile(templatePath, []byte(testPlaceholderFreeTemplate), 0o600))

	placeholders, extractErr := extractPlaceholders(templatePath)
	require.NoError(testingT, extractErr)
	require.Nil(testingT, placeholders)
}

func TestParseVolumeMappingRejectsInvalidEntries(testingT *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "empty",
			input: "",
		},
		{
			name:  "invalid",
			input: testInvalidVolumeMapping,
		},
		{
			name:  "missing_host",
			input: testInvalidVolumeMissingHost,
		},
		{
			name:  "missing_container",
			input: testInvalidVolumeMissingTarget,
		},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			hostPath, containerPath, ok := parseVolumeMapping(testCase.input)
			require.False(testingT, ok)
			require.Empty(testingT, hostPath)
			require.Empty(testingT, containerPath)
		})
	}
}

func TestScanAssetRootSkipsSymlinkAndUnsupportedExtensions(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	targetDirectory := testingT.TempDir()
	targetPath := filepath.Join(targetDirectory, testAssetScanFileName)
	targetContent := testAssetScanLocalhostURLPrefix + testTauthContainerPort
	require.NoError(testingT, os.WriteFile(targetPath, []byte(targetContent), 0o600))

	linkPath := filepath.Join(tempDirectory, testAssetScanSymlinkName)
	if symlinkErr := os.Symlink(targetPath, linkPath); symlinkErr != nil {
		testingT.Skipf("symlink not supported: %v", symlinkErr)
	}

	ignoredPath := filepath.Join(tempDirectory, testAssetScanIgnoredFileName)
	require.NoError(testingT, os.WriteFile(ignoredPath, []byte(targetContent), 0o600))

	allowedPorts := map[string]struct{}{}
	result := auditResult{}
	scanErr := scanAssetRoot(tempDirectory, allowedPorts, &result)
	require.NoError(testingT, scanErr)
	require.Empty(testingT, result.errors)
	require.Empty(testingT, result.warnings)
}

func TestLoadServiceEnvironmentUsesInlineValuesWhenEnvFileMissing(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	result := auditResult{}

	environment := environmentMap{
		testInlineEnvironmentKey: testPinguinAddressValue,
	}

	environmentValues, hasAuditableEnvironment, loadErr := loadServiceEnvironment(tempDirectory, "app", []string{testMissingEnvFileName}, environment, &result)
	require.NoError(testingT, loadErr)
	require.True(testingT, hasAuditableEnvironment)
	require.Empty(testingT, result.errors)
	require.NotEmpty(testingT, result.warnings)
	require.Equal(testingT, testPinguinAddressValue, environmentValues[testInlineEnvironmentKey])
}

func TestCheckHostPortCollisionsSkipsInvalidMapping(testingT *testing.T) {
	hostPortToService := make(map[string]string)
	result := auditResult{}
	checkHostPortCollisions(testHostPortServiceName, []string{testInvalidVolumeMapping}, hostPortToService, &result)
	require.Empty(testingT, result.errors)
	require.Empty(testingT, hostPortToService)
}

func TestCheckLocalPortMatchesSkipsEmptyPort(testingT *testing.T) {
	result := auditResult{}
	checkLocalPortMatches(testAssetScanFileName, 1, testAssetScanLocalhostNoPort, map[string]struct{}{}, &result)
	require.Empty(testingT, result.errors)
	require.Empty(testingT, result.warnings)
}

func TestStringListUnmarshalYAMLSkipsNilAndBlankEntries(testingT *testing.T) {
	var values stringList
	node := &yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			nil,
			{Kind: yaml.ScalarNode, Value: "   "},
			{Kind: yaml.ScalarNode, Value: testStringListValue},
		},
	}

	unmarshalErr := values.UnmarshalYAML(node)
	require.NoError(testingT, unmarshalErr)
	require.Equal(testingT, []string{testStringListValue}, []string(values))
}

func TestEnvironmentMapUnmarshalYAMLReportsDecodeError(testingT *testing.T) {
	testCases := []struct {
		name string
		node *yaml.Node
	}{
		{
			name: "mapping node",
			node: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.SequenceNode},
					{Kind: yaml.ScalarNode, Value: testEnvDecodeValue},
				},
			},
		},
		{
			name: "sequence node",
			node: &yaml.Node{
				Kind: yaml.SequenceNode,
				Content: []*yaml.Node{
					{
						Kind: yaml.MappingNode,
						Content: []*yaml.Node{
							{Kind: yaml.ScalarNode, Value: testEnvDecodeKey},
							{Kind: yaml.ScalarNode, Value: testEnvDecodeValue},
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			var environment environmentMap
			unmarshalErr := environment.UnmarshalYAML(testCase.node)
			require.Error(testingT, unmarshalErr)
		})
	}
}

func TestParseDotEnvSkipsMalformedEntries(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	envPath := filepath.Join(tempDirectory, testLoopAwareEnvFile)
	envContent := strings.Join([]string{
		testMalformedEnvLineNoEquals,
		testMalformedEnvLineEmptyKey,
		testInlineEnvironmentKey + "=" + testPinguinAddressValue,
		"",
	}, "\n")
	require.NoError(testingT, os.WriteFile(envPath, []byte(envContent), 0o600))

	values, duplicates, parseErr := parseDotEnv(envPath)
	require.NoError(testingT, parseErr)
	require.Empty(testingT, duplicates)
	require.Equal(testingT, testPinguinAddressValue, values[testInlineEnvironmentKey])
}

func TestParseDotEnvReportsScannerError(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	envPath := filepath.Join(tempDirectory, testLoopAwareEnvFile)
	longLine := testEnvScanPrefix + strings.Repeat(testEnvScanPad, testEnvScanLineLength)
	require.NoError(testingT, os.WriteFile(envPath, []byte(longLine), 0o600))

	_, _, parseErr := parseDotEnv(envPath)
	require.Error(testingT, parseErr)
	require.True(testingT, errors.Is(parseErr, bufio.ErrTooLong))
}

func TestLoadServiceEnvironmentReportsParseError(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	envPath := filepath.Join(tempDirectory, testLoopAwareEnvFile)
	longLine := testEnvScanPrefix + strings.Repeat(testEnvScanPad, testEnvScanLineLength)
	require.NoError(testingT, os.WriteFile(envPath, []byte(longLine), 0o600))

	result := auditResult{}
	_, _, loadErr := loadServiceEnvironment(tempDirectory, testLoopAwareService, []string{testLoopAwareEnvFile}, environmentMap{}, &result)
	require.Error(testingT, loadErr)
	require.Contains(testingT, loadErr.Error(), "parse env_file")
}

func TestLoadServiceEnvironmentSkipsBlankKeys(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	result := auditResult{}
	environment := environmentMap{
		" ":                      testEnvDecodeValue,
		testInlineEnvironmentKey: testPinguinAddressValue,
	}

	values, hasAuditableEnvironment, loadErr := loadServiceEnvironment(tempDirectory, testLoopAwareService, nil, environment, &result)
	require.NoError(testingT, loadErr)
	require.True(testingT, hasAuditableEnvironment)
	require.Equal(testingT, testPinguinAddressValue, values[testInlineEnvironmentKey])
	require.Empty(testingT, values[" "])
}

func TestResolveConfigTemplatesSkipsNonTemplateVolume(testingT *testing.T) {
	templates := resolveConfigTemplates(".", []string{testNonTemplateVolumeMapping})
	require.Empty(testingT, templates)
}

func TestCheckHostPortCollisionsSkipsBlankMapping(testingT *testing.T) {
	hostPortToService := make(map[string]string)
	result := auditResult{}
	checkHostPortCollisions(testHostPortServiceName, []string{"  "}, hostPortToService, &result)
	require.Empty(testingT, hostPortToService)
	require.Empty(testingT, result.errors)
}

func TestCheckLocalPortMatchesSkipsUnexpectedPatternShapes(testingT *testing.T) {
	originalLocalURLPattern := localURLPattern
	originalLocalHostPortPattern := localHostPortPattern
	localURLPattern = regexp.MustCompile(testLocalPatternNoGroups)
	localHostPortPattern = regexp.MustCompile(testLocalPatternNoGroups)
	testingT.Cleanup(func() {
		localURLPattern = originalLocalURLPattern
		localHostPortPattern = originalLocalHostPortPattern
	})

	result := auditResult{}
	checkLocalPortMatches(testAssetScanFileName, 1, testAssetScanLocalhostNoPort, map[string]struct{}{}, &result)
	require.Empty(testingT, result.errors)
	require.Empty(testingT, result.warnings)

	localHostPortPattern = regexp.MustCompile(testLocalHostPortPatternEmpty)
	checkLocalPortMatches(testAssetScanFileName, 2, testAssetScanHostPortPrefix, map[string]struct{}{}, &result)
	require.Empty(testingT, result.errors)
	require.Empty(testingT, result.warnings)
}

func TestCheckWebAssetLocalhostPortsHandlesLongLines(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	originalDirectory, originalErr := os.Getwd()
	require.NoError(testingT, originalErr)
	require.NoError(testingT, os.Chdir(tempDirectory))
	testingT.Cleanup(func() {
		_ = os.Chdir(originalDirectory)
	})

	webPath := filepath.Join(tempDirectory, "web")
	require.NoError(testingT, os.MkdirAll(webPath, 0o755))
	longLine := strings.Repeat(testEnvScanPad, testEnvScanLineLength)
	require.NoError(testingT, os.WriteFile(filepath.Join(webPath, testAssetScanFileName), []byte(longLine), 0o600))

	result := auditResult{}
	checkWebAssetLocalhostPorts(map[string]string{}, &result)
	require.Empty(testingT, result.errors)
	require.Empty(testingT, result.warnings)
}

func TestCheckWebAssetLocalhostPortsReportsReadError(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	originalDirectory, originalErr := os.Getwd()
	require.NoError(testingT, originalErr)
	require.NoError(testingT, os.Chdir(tempDirectory))
	testingT.Cleanup(func() {
		_ = os.Chdir(originalDirectory)
	})

	webPath := filepath.Join(tempDirectory, "web")
	require.NoError(testingT, os.MkdirAll(webPath, 0o755))
	assetPath := filepath.Join(webPath, "sample.js")
	require.NoError(testingT, os.WriteFile(assetPath, []byte("http://localhost:1234"), 0o600))
	if chmodErr := os.Chmod(assetPath, 0o000); chmodErr != nil {
		testingT.Skipf("chmod not supported: %v", chmodErr)
	}

	result := auditResult{}
	checkWebAssetLocalhostPorts(map[string]string{}, &result)
	require.NotEmpty(testingT, result.errors)
}

func TestRunAuditReportsLoopAwareServiceWithoutRuntimeConfigVolume(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	composePath := filepath.Join(tempDirectory, testComposeFileName)
	compose := composeFile{
		Services: map[string]composeService{
			testLoopAwareService: {},
		},
	}
	payload, marshalErr := yaml.Marshal(&compose)
	require.NoError(testingT, marshalErr)
	require.NoError(testingT, os.WriteFile(composePath, payload, 0o600))

	result := runAudit(composePath)
	require.False(testingT, result.ok())
	require.Contains(testingT, strings.Join(result.errors, " "), "missing LoopAware runtime config volume")
	require.Empty(testingT, result.warnings)
}

func TestRunAuditReportsTemplateReadError(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	composePath := filepath.Join(tempDirectory, testComposeFileName)
	compose := composeFile{
		Services: map[string]composeService{
			testLoopAwareService: {
				Environment: environmentMap{
					testInlineEnvironmentKey: testPinguinAddressValue,
				},
				Volumes: stringList{testTemplateVolumeMapping},
			},
		},
	}
	payload, marshalErr := yaml.Marshal(&compose)
	require.NoError(testingT, marshalErr)
	require.NoError(testingT, os.WriteFile(composePath, payload, 0o600))

	result := runAudit(composePath)
	require.False(testingT, result.ok())
	require.Contains(testingT, strings.Join(result.errors, " "), "read config template")
}

func TestRunAuditCommandReportsWarnings(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	composePath := filepath.Join(tempDirectory, testComposeFileName)
	compose := composeFile{
		Services: map[string]composeService{
			testPinguinService: {
				Environment: environmentMap{
					testEnvKeyPinguinSigning:      "",
					testEnvKeyPinguinGoogleClient: "",
					testEnvKeyPinguinAuthToken:    "",
				},
			},
			testTauthService: {
				Environment: environmentMap{
					testEnvKeyTauthSigning:      "",
					testEnvKeyTauthGoogleClient: "",
				},
			},
		},
	}
	payload, marshalErr := yaml.Marshal(&compose)
	require.NoError(testingT, marshalErr)
	require.NoError(testingT, os.WriteFile(composePath, payload, 0o600))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runAuditCommand(composePath, &stdout, &stderr)
	require.Equal(testingT, 0, exitCode)
	require.Contains(testingT, stdout.String(), "WARN [")
	require.Contains(testingT, stdout.String(), "config-audit OK")
	require.Empty(testingT, stderr.String())
}

func TestRunAuditCommandsSuccessAcrossMultipleComposeFiles(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	composeOnePath := filepath.Join(tempDirectory, "docker-compose.one.yml")
	composeTwoPath := filepath.Join(tempDirectory, "docker-compose.two.yml")
	loopAwareConfigVolume := loopAwareRuntimeConfigVolume(testingT, tempDirectory)
	composeLines := []string{
		"services:",
		"  loopaware:",
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
	require.NoError(testingT, os.WriteFile(composeOnePath, []byte(composeContent), 0o600))
	require.NoError(testingT, os.WriteFile(composeTwoPath, []byte(composeContent), 0o600))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runAuditCommands([]string{composeOnePath, composeTwoPath}, &stdout, &stderr)
	require.Equal(testingT, 0, exitCode)
	require.Contains(testingT, stdout.String(), "config-audit OK")
	require.Empty(testingT, stderr.String())
}

func TestRunAuditCommandsIgnoresLegacyRootEnvFiles(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	composePath := filepath.Join(tempDirectory, testComposeFileName)
	loopAwareConfigVolume := loopAwareRuntimeConfigVolume(testingT, tempDirectory)
	composeLines := []string{
		"services:",
		"  loopaware:",
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
	require.NoError(testingT, os.WriteFile(filepath.Join(tempDirectory, ".env.loopaware"), []byte("SESSION_SECRET=legacy"), 0o600))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runAuditCommands([]string{composePath}, &stdout, &stderr)
	require.Equal(testingT, 0, exitCode)
	require.Contains(testingT, stdout.String(), "config-audit OK")
	require.Empty(testingT, stderr.String())
}

func TestRunAuditCommandsIgnoresUnsupportedLocalConfigEnvFile(testingT *testing.T) {
	tempDirectory := testingT.TempDir()
	composePath := filepath.Join(tempDirectory, testComposeFileName)
	loopAwareConfigVolume := loopAwareRuntimeConfigVolume(testingT, tempDirectory)
	composeLines := []string{
		"services:",
		"  loopaware:",
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
	require.NoError(testingT, os.MkdirAll(filepath.Join(tempDirectory, "configs"), 0o755))
	require.NoError(testingT, os.WriteFile(filepath.Join(tempDirectory, "configs", ".env.ghttp"), []byte("GHTTP_SERVE_PORT=4443"), 0o600))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := runAuditCommands([]string{composePath}, &stdout, &stderr)
	require.Equal(testingT, 0, exitCode)
	require.Contains(testingT, stdout.String(), "config-audit OK")
	require.Empty(testingT, stderr.String())
}

func TestMainRunsAuditFromRepoRoot(testingT *testing.T) {
	workingDirectory, workingDirectoryErr := os.Getwd()
	require.NoError(testingT, workingDirectoryErr)

	repositoryRoot := filepath.Dir(filepath.Dir(workingDirectory))
	composePath := filepath.Join(repositoryRoot, testComposeFileName)
	_, statErr := os.Stat(composePath)
	require.NoError(testingT, statErr)

	require.NoError(testingT, os.Chdir(repositoryRoot))
	testingT.Cleanup(func() {
		_ = os.Chdir(workingDirectory)
	})

	main()
}
