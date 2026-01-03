package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	errAuditFailed       = errors.New("config_audit_failed")
	placeholderPattern   = regexp.MustCompile(`\$\{([A-Z0-9_]+)\}`)
	volumeMappingSuffix  = "/config/config.yml"
	localURLPattern      = regexp.MustCompile(`https?://(?:localhost|127\.0\.0\.1)(?::([0-9]{2,5}))?`)
	localHostPortPattern = regexp.MustCompile(`(?:^|[^a-zA-Z0-9_.-])(localhost|127\.0\.0\.1):([0-9]{2,5})`)
)

type stringList []string

func (list *stringList) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		*list = nil
		return nil
	}
	switch node.Kind {
	case yaml.ScalarNode:
		value := strings.TrimSpace(node.Value)
		if value == "" {
			*list = nil
			return nil
		}
		*list = []string{value}
		return nil
	case yaml.SequenceNode:
		entries := make([]string, 0, len(node.Content))
		for _, child := range node.Content {
			if child == nil {
				continue
			}
			value := strings.TrimSpace(child.Value)
			if value == "" {
				continue
			}
			entries = append(entries, value)
		}
		*list = entries
		return nil
	default:
		return fmt.Errorf("unsupported yaml node kind %d for list", node.Kind)
	}
}

type environmentMap map[string]string

func (environment *environmentMap) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		*environment = nil
		return nil
	}
	switch node.Kind {
	case yaml.MappingNode:
		decoded := make(map[string]string)
		if err := node.Decode(&decoded); err != nil {
			return err
		}
		normalized := make(map[string]string, len(decoded))
		for key, value := range decoded {
			normalized[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
		*environment = normalized
		return nil
	case yaml.SequenceNode:
		decoded := make([]string, 0, len(node.Content))
		if err := node.Decode(&decoded); err != nil {
			return err
		}
		normalized := make(map[string]string)
		for _, entry := range decoded {
			trimmed := strings.TrimSpace(entry)
			if trimmed == "" {
				continue
			}
			key, value, ok := strings.Cut(trimmed, "=")
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			if !ok {
				normalized[key] = ""
				continue
			}
			normalized[key] = strings.TrimSpace(value)
		}
		*environment = normalized
		return nil
	default:
		return fmt.Errorf("unsupported yaml node kind %d for environment", node.Kind)
	}
}

type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	EnvFile     stringList     `yaml:"env_file"`
	Environment environmentMap `yaml:"environment"`
	Volumes     stringList     `yaml:"volumes"`
	Ports       stringList     `yaml:"ports"`
	DependsOn   yaml.Node      `yaml:"depends_on"`
	PullPolicy  string         `yaml:"pull_policy"`
	Restart     string         `yaml:"restart"`
	Image       string         `yaml:"image"`
	Build       interface{}    `yaml:"build"`
	Develop     interface{}    `yaml:"develop"`
	Container   string         `yaml:"container_name"`
	OtherKeys   map[string]any `yaml:",inline"`
	OtherFields map[string]any `yaml:"-"`
}

type auditResult struct {
	errors   []string
	warnings []string
}

func (result *auditResult) addError(message string, arguments ...any) {
	result.errors = append(result.errors, fmt.Sprintf(message, arguments...))
}

func (result *auditResult) addWarning(message string, arguments ...any) {
	result.warnings = append(result.warnings, fmt.Sprintf(message, arguments...))
}

func (result auditResult) ok() bool {
	return len(result.errors) == 0
}

func main() {
	result := runAudit("docker-compose.yml")
	sort.Strings(result.errors)
	sort.Strings(result.warnings)

	for _, warning := range result.warnings {
		_, _ = fmt.Fprintf(os.Stdout, "WARN: %s\n", warning)
	}
	for _, errorMessage := range result.errors {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s\n", errorMessage)
	}
	if !result.ok() {
		_, _ = fmt.Fprintf(os.Stderr, "config-audit failed\n")
		os.Exit(1)
	}
	_, _ = fmt.Fprintf(os.Stdout, "config-audit OK\n")
}

func runAudit(composePath string) auditResult {
	var result auditResult

	composeDocument, readErr := os.ReadFile(composePath)
	if readErr != nil {
		result.addError("read compose file %s: %v", composePath, readErr)
		return result
	}

	var compose composeFile
	decoder := yaml.NewDecoder(strings.NewReader(string(composeDocument)))
	if decodeErr := decoder.Decode(&compose); decodeErr != nil {
		result.addError("parse compose file %s: %v", composePath, decodeErr)
		return result
	}
	if len(compose.Services) == 0 {
		result.addError("compose file %s: no services defined", composePath)
		return result
	}

	composeDirectory := filepath.Dir(composePath)
	environmentByService := make(map[string]map[string]string, len(compose.Services))
	hostPortToService := make(map[string]string)

	for serviceName, service := range compose.Services {
		env, envErr := loadServiceEnvironment(composeDirectory, serviceName, service.EnvFile, service.Environment, &result)
		if envErr != nil {
			result.addError("service %s: %v", serviceName, envErr)
			continue
		}
		environmentByService[serviceName] = env

		configTemplates := resolveConfigTemplates(composeDirectory, service.Volumes)
		for _, templatePath := range configTemplates {
			placeholders, placeholderErr := extractPlaceholders(templatePath)
			if placeholderErr != nil {
				result.addError("service %s: %v", serviceName, placeholderErr)
				continue
			}
			for _, placeholderName := range placeholders {
				if _, ok := env[placeholderName]; !ok {
					result.addError("service %s: %s references ${%s} but %s is not defined in env", serviceName, templatePath, placeholderName, placeholderName)
				}
			}
		}

		checkHostPortCollisions(serviceName, service.Ports, hostPortToService, &result)
	}

	checkCrossServiceInvariants(environmentByService, &result)
	checkLoopAwareRequiredEnvironment(environmentByService, &result)
	checkWebAssetLocalhostPorts(hostPortToService, &result)

	return result
}

func loadServiceEnvironment(composeDirectory string, serviceName string, envFiles []string, environment environmentMap, result *auditResult) (map[string]string, error) {
	merged := make(map[string]string)

	for _, envFile := range envFiles {
		resolvedPath := filepath.Clean(filepath.Join(composeDirectory, envFile))
		if _, statErr := os.Stat(resolvedPath); statErr != nil {
			result.addError("service %s: env_file %s is missing (%v)", serviceName, envFile, statErr)
			continue
		}
		values, duplicates, parseErr := parseDotEnv(resolvedPath)
		if parseErr != nil {
			return nil, fmt.Errorf("parse env_file %s: %w", envFile, parseErr)
		}
		for _, duplicate := range duplicates {
			result.addError("service %s: env_file %s defines %s more than once", serviceName, envFile, duplicate)
		}
		for key, value := range values {
			merged[key] = value
		}
	}

	for key, value := range environment {
		if strings.TrimSpace(key) == "" {
			continue
		}
		merged[key] = value
	}

	if len(merged) == 0 {
		return nil, fmt.Errorf("%w: no environment variables resolved", errAuditFailed)
	}

	return merged, nil
}

func parseDotEnv(path string) (map[string]string, []string, error) {
	file, openErr := os.Open(path)
	if openErr != nil {
		return nil, nil, openErr
	}
	defer func() { _ = file.Close() }()

	entries := make(map[string]string)
	seen := make(map[string]struct{})
	var duplicates []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if _, already := seen[key]; already {
			duplicates = append(duplicates, key)
		}
		seen[key] = struct{}{}
		entries[key] = value
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, nil, scanErr
	}

	sort.Strings(duplicates)
	duplicates = uniqueStrings(duplicates)
	return entries, duplicates, nil
}

func resolveConfigTemplates(composeDirectory string, volumes []string) []string {
	var templates []string
	for _, volume := range volumes {
		hostPath, containerPath, ok := parseVolumeMapping(volume)
		if !ok {
			continue
		}
		if !strings.HasSuffix(containerPath, volumeMappingSuffix) {
			continue
		}
		resolvedHost := filepath.Clean(filepath.Join(composeDirectory, hostPath))
		templates = append(templates, resolvedHost)
	}
	sort.Strings(templates)
	return uniqueStrings(templates)
}

func parseVolumeMapping(entry string) (string, string, bool) {
	trimmed := strings.TrimSpace(entry)
	if trimmed == "" {
		return "", "", false
	}
	parts := strings.SplitN(trimmed, ":", 3)
	if len(parts) < 2 {
		return "", "", false
	}
	hostPath := strings.TrimSpace(parts[0])
	containerPath := strings.TrimSpace(parts[1])
	if hostPath == "" || containerPath == "" {
		return "", "", false
	}
	return hostPath, containerPath, true
}

func extractPlaceholders(path string) ([]string, error) {
	payload, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil, fmt.Errorf("read config template %s: %v", path, readErr)
	}

	matches := placeholderPattern.FindAllStringSubmatch(string(payload), -1)
	if len(matches) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(matches))
	var placeholders []string
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		placeholders = append(placeholders, name)
	}
	sort.Strings(placeholders)
	return placeholders, nil
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	sort.Strings(values)
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if len(unique) == 0 || unique[len(unique)-1] != value {
			unique = append(unique, value)
		}
	}
	return unique
}

func checkHostPortCollisions(serviceName string, ports []string, hostPortToService map[string]string, result *auditResult) {
	for _, mapping := range ports {
		trimmed := strings.TrimSpace(mapping)
		if trimmed == "" {
			continue
		}
		hostPort, ok := parseHostPort(trimmed)
		if !ok {
			continue
		}
		if existingService, already := hostPortToService[hostPort]; already {
			result.addError("compose: host port %s is published by both %s and %s", hostPort, existingService, serviceName)
		} else {
			hostPortToService[hostPort] = serviceName
		}
	}
}

func parseHostPort(portMapping string) (string, bool) {
	trimmed := strings.Trim(portMapping, `"`)
	parts := strings.Split(trimmed, ":")
	if len(parts) < 2 {
		return "", false
	}
	hostPort := strings.TrimSpace(parts[len(parts)-2])
	if hostPort == "" {
		return "", false
	}
	for _, runeValue := range hostPort {
		if runeValue < '0' || runeValue > '9' {
			return "", false
		}
	}
	return hostPort, true
}

func checkCrossServiceInvariants(environmentByService map[string]map[string]string, result *auditResult) {
	pinguinEnv, pinguinOk := environmentByService["pinguin"]
	tauthEnv, tauthOk := environmentByService["tauth"]
	loopawareEnv, loopawareOk := environmentByService["loopaware"]

	if pinguinOk && tauthOk {
		expectEqual("pinguin.TAUTH_SIGNING_KEY", pinguinEnv["TAUTH_SIGNING_KEY"], "tauth.TAUTH_LOOPAWARE_JWT_SIGNING_KEY", tauthEnv["TAUTH_LOOPAWARE_JWT_SIGNING_KEY"], result)
		expectEqual("pinguin.LOOPAWARE_LOCAL_GOOGLE_CLIENT_ID", pinguinEnv["LOOPAWARE_LOCAL_GOOGLE_CLIENT_ID"], "tauth.TAUTH_LOOPAWARE_GOOGLE_WEB_CLIENT_ID", tauthEnv["TAUTH_LOOPAWARE_GOOGLE_WEB_CLIENT_ID"], result)
	}

	if pinguinOk && loopawareOk {
		expectEqual("loopaware.PINGUIN_AUTH_TOKEN", loopawareEnv["PINGUIN_AUTH_TOKEN"], "pinguin.GRPC_AUTH_TOKEN", pinguinEnv["GRPC_AUTH_TOKEN"], result)
	}
}

func expectEqual(leftLabel string, leftValue string, rightLabel string, rightValue string, result *auditResult) {
	leftNormalized := strings.TrimSpace(leftValue)
	rightNormalized := strings.TrimSpace(rightValue)
	if leftNormalized == "" || rightNormalized == "" {
		result.addWarning("invariant check: %s or %s is empty", leftLabel, rightLabel)
		return
	}
	if leftNormalized != rightNormalized {
		result.addError("invariant check failed: %s must match %s", leftLabel, rightLabel)
	}
}

func checkLoopAwareRequiredEnvironment(environmentByService map[string]map[string]string, result *auditResult) {
	loopawareEnv, ok := environmentByService["loopaware"]
	if !ok {
		return
	}
	requiredKeys := []string{
		"GOOGLE_CLIENT_ID",
		"SESSION_SECRET",
		"TAUTH_BASE_URL",
		"TAUTH_TENANT_ID",
		"TAUTH_JWT_SIGNING_KEY",
		"TAUTH_SESSION_COOKIE_NAME",
		"PUBLIC_BASE_URL",
		"PINGUIN_ADDR",
		"PINGUIN_AUTH_TOKEN",
	}
	for _, key := range requiredKeys {
		if strings.TrimSpace(loopawareEnv[key]) == "" {
			result.addError("service loopaware: required env %s is missing or empty", key)
		}
	}
}

func checkWebAssetLocalhostPorts(hostPortToService map[string]string, result *auditResult) {
	allowedPorts := make(map[string]struct{}, len(hostPortToService))
	for hostPort := range hostPortToService {
		allowedPorts[hostPort] = struct{}{}
	}

	assetRoots := []string{
		filepath.Join("internal", "httpapi", "assets"),
		filepath.Join("internal", "httpapi", "templates"),
		"web",
	}

	for _, root := range assetRoots {
		info, statErr := os.Stat(root)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			result.addError("asset scan: stat %s: %v", root, statErr)
			continue
		}
		if !info.IsDir() {
			continue
		}
		if err := scanAssetRoot(root, allowedPorts, result); err != nil {
			result.addError("asset scan: %v", err)
		}
	}
}

func scanAssetRoot(root string, allowedPorts map[string]struct{}, result *auditResult) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".js", ".html", ".tmpl", ".css":
		default:
			return nil
		}

		return scanAssetFile(path, allowedPorts, result)
	})
}

func scanAssetFile(path string, allowedPorts map[string]struct{}, result *auditResult) error {
	file, openErr := os.Open(path)
	if openErr != nil {
		return openErr
	}

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		checkLocalPortMatches(path, lineNumber, line, allowedPorts, result)
	}
	scanErr := scanner.Err()
	closeErr := file.Close()
	if scanErr != nil || closeErr != nil {
		return errors.Join(scanErr, closeErr)
	}
	return nil
}

func checkLocalPortMatches(path string, lineNumber int, line string, allowedPorts map[string]struct{}, result *auditResult) {
	for _, match := range localURLPattern.FindAllStringSubmatch(line, -1) {
		if len(match) < 2 {
			continue
		}
		port := strings.TrimSpace(match[1])
		if port == "" {
			continue
		}
		recordLocalPort(path, lineNumber, port, allowedPorts, result)
	}
	for _, match := range localHostPortPattern.FindAllStringSubmatch(line, -1) {
		if len(match) < 3 {
			continue
		}
		port := strings.TrimSpace(match[2])
		if port == "" {
			continue
		}
		recordLocalPort(path, lineNumber, port, allowedPorts, result)
	}
}

func recordLocalPort(path string, lineNumber int, port string, allowedPorts map[string]struct{}, result *auditResult) {
	if _, err := strconv.Atoi(port); err != nil {
		return
	}
	if _, ok := allowedPorts[port]; ok {
		result.addWarning("asset scan: %s:%d references localhost:%s (allowed by compose ports)", path, lineNumber, port)
		return
	}
	result.addError("asset scan: %s:%d references localhost:%s which is not published in docker-compose.yml", path, lineNumber, port)
}
