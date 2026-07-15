package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/MarkoPoloResearchLab/loopaware/internal/serverconfig"
	"gopkg.in/yaml.v3"
)

const (
	assetScanMaxTokenBytes              = 8 * 1024 * 1024
	sharedConfigContainerPath           = "/config/config.yml"
	loopAwareRuntimeConfigContainerPath = "/app/configs/config.loopaware.yml"
)

var (
	placeholderPattern   = regexp.MustCompile(`\$\{([A-Z0-9_]+)\}`)
	localURLPattern      = regexp.MustCompile(`https?://(?:localhost|127\.0\.0\.1)(?::([0-9]{2,5}))?`)
	localHostPortPattern = regexp.MustCompile(`(?:^|[^a-zA-Z0-9_.-])(localhost|127\.0\.0\.1):([0-9]{2,5})`)
	defaultComposePaths  = []string{
		"docker-compose.yml",
		"docker-compose.computercat.yml",
		".mprlab/deploy/docker-compose.yml",
	}
	loopAwareServiceAliases = []string{
		"loopaware",
		"loopaware-api",
	}
	forbiddenLocalThirdPartyPaths = []string{
		filepath.Join("tools", "mpr-ui"),
	}
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
	exitCode := runAuditCommands(defaultComposePaths, os.Stdout, os.Stderr)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func runAuditCommands(composePaths []string, stdout io.Writer, stderr io.Writer) int {
	overallSuccess := true

	for _, composePath := range composePaths {
		result := runAudit(composePath)
		sort.Strings(result.errors)
		sort.Strings(result.warnings)

		for _, warning := range result.warnings {
			_, _ = fmt.Fprintf(stdout, "WARN [%s]: %s\n", composePath, warning)
		}
		for _, errorMessage := range result.errors {
			_, _ = fmt.Fprintf(stderr, "ERROR [%s]: %s\n", composePath, errorMessage)
		}
		if !result.ok() {
			overallSuccess = false
		}
	}

	if !overallSuccess {
		_, _ = fmt.Fprintf(stderr, "config-audit failed\n")
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "config-audit OK\n")
	return 0
}

func runAuditCommand(composePath string, stdout io.Writer, stderr io.Writer) int {
	return runAuditCommands([]string{composePath}, stdout, stderr)
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
	hostPortToService := make(map[string]string)

	for serviceName, service := range compose.Services {
		env, hasAuditableEnvironment, envErr := loadServiceEnvironment(composeDirectory, serviceName, service.EnvFile, service.Environment, &result)
		if envErr != nil {
			result.addError("service %s: %v", serviceName, envErr)
			continue
		}
		if hasAuditableEnvironment {
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
		} else if configTemplates := resolveConfigTemplates(composeDirectory, service.Volumes); len(configTemplates) > 0 {
			result.addWarning("service %s: skipped config template env audit because no tracked environment data is available", serviceName)
		}
		if isLoopAwareService(serviceName) {
			loopAwareRuntimeConfigTemplates := resolveLoopAwareRuntimeConfigTemplates(composeDirectory, service.Volumes)
			if len(loopAwareRuntimeConfigTemplates) == 0 {
				result.addError("service %s: missing LoopAware runtime config volume %s", serviceName, loopAwareRuntimeConfigContainerPath)
			}
			if hasAuditableEnvironment {
				for _, templatePath := range loopAwareRuntimeConfigTemplates {
					_, runtimeConfigErr := loadLoopAwareRuntimeConfig(templatePath, env)
					if runtimeConfigErr != nil {
						result.addError("service %s: %v", serviceName, runtimeConfigErr)
						continue
					}
				}
			} else if len(loopAwareRuntimeConfigTemplates) > 0 {
				result.addWarning("service %s: skipped LoopAware runtime config audit because no tracked environment data is available", serviceName)
			}
		}

		checkHostPortCollisions(serviceName, service.Ports, hostPortToService, &result)
	}

	checkForbiddenLocalThirdPartyPaths(composeDirectory, &result)
	checkWebAssetLocalhostPorts(hostPortToService, &result)

	return result
}

func loadServiceEnvironment(composeDirectory string, serviceName string, envFiles []string, environment environmentMap, result *auditResult) (map[string]string, bool, error) {
	merged := make(map[string]string)
	hasAuditableEnvironment := false

	for _, envFile := range envFiles {
		auditPath, displayPath, _, found, resolveErr := resolveAuditEnvFile(composeDirectory, envFile)
		if resolveErr != nil {
			return nil, false, fmt.Errorf("resolve env_file %s: %w", envFile, resolveErr)
		}
		if !found {
			result.addWarning("service %s: env_file %s is absent and no tracked template exists; skipping env audit for this file", serviceName, envFile)
			continue
		}
		values, duplicates, parseErr := parseDotEnv(auditPath)
		if parseErr != nil {
			return nil, false, fmt.Errorf("parse env_file %s: %w", displayPath, parseErr)
		}
		for _, duplicate := range duplicates {
			result.addError("service %s: env_file %s defines %s more than once", serviceName, displayPath, duplicate)
		}
		for key, value := range values {
			merged[key] = value
		}
		hasAuditableEnvironment = true
	}

	for key, value := range environment {
		if strings.TrimSpace(key) == "" {
			continue
		}
		merged[key] = value
		hasAuditableEnvironment = true
	}

	return merged, hasAuditableEnvironment, nil
}

func resolveAuditEnvFile(composeDirectory string, envFile string) (string, string, bool, bool, error) {
	resolvedPath := filepath.Clean(filepath.Join(composeDirectory, envFile))
	foundPath, found, resolveErr := findExistingAuditFile(resolvedPath)
	if resolveErr != nil {
		return "", "", false, false, resolveErr
	}
	if found {
		return foundPath, envFile, false, true, nil
	}

	examplePath := resolvedPath + ".example"
	foundExamplePath, foundExample, exampleErr := findExistingAuditFile(examplePath)
	if exampleErr != nil {
		return "", "", false, false, exampleErr
	}
	if foundExample {
		return foundExamplePath, envFile + ".example", true, true, nil
	}

	return "", "", false, false, nil
}

func findExistingAuditFile(path string) (string, bool, error) {
	fileInfo, statErr := os.Stat(path)
	if statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, statErr
	}
	if fileInfo.IsDir() {
		return "", false, fmt.Errorf("%s must not be a directory", path)
	}
	return path, true, nil
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
		if containerPath != sharedConfigContainerPath && containerPath != loopAwareRuntimeConfigContainerPath {
			continue
		}
		resolvedHost := filepath.Clean(filepath.Join(composeDirectory, hostPath))
		templates = append(templates, resolvedHost)
	}
	sort.Strings(templates)
	return uniqueStrings(templates)
}

func resolveLoopAwareRuntimeConfigTemplates(composeDirectory string, volumes []string) []string {
	var templates []string
	for _, volume := range volumes {
		hostPath, containerPath, ok := parseVolumeMapping(volume)
		if !ok {
			continue
		}
		if containerPath != loopAwareRuntimeConfigContainerPath {
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

func loadLoopAwareRuntimeConfig(configPath string, environment map[string]string) (serverconfig.Config, error) {
	loadedConfig, loadError := serverconfig.LoadWithLookup(configPath, func(name string) (string, bool) {
		value, found := environment[name]
		return value, found
	})
	if loadError != nil {
		return serverconfig.Config{}, fmt.Errorf("load LoopAware runtime config %s: %w", configPath, loadError)
	}
	return loadedConfig, nil
}

func isLoopAwareService(serviceName string) bool {
	for _, alias := range loopAwareServiceAliases {
		if serviceName == alias {
			return true
		}
	}
	return false
}

func checkWebAssetLocalhostPorts(hostPortToService map[string]string, result *auditResult) {
	allowedPorts := make(map[string]struct{}, len(hostPortToService))
	for hostPort := range hostPortToService {
		allowedPorts[hostPort] = struct{}{}
	}

	assetRoots := []string{
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

func checkForbiddenLocalThirdPartyPaths(root string, result *auditResult) {
	for _, relativePath := range forbiddenLocalThirdPartyPaths {
		fullPath := filepath.Join(root, relativePath)
		if _, statErr := os.Lstat(fullPath); statErr == nil {
			result.addError("third-party asset path %s must not exist; use CDN-hosted assets only", relativePath)
			continue
		} else if !os.IsNotExist(statErr) {
			result.addError("third-party asset path %s could not be checked: %v", relativePath, statErr)
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
	scanBuffer := make([]byte, 0, assetScanMaxTokenBytes)
	scanner.Buffer(scanBuffer, assetScanMaxTokenBytes)
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
