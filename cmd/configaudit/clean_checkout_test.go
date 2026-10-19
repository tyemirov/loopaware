package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCleanCheckoutConfigurationAuditMakeCommand(testingT *testing.T) {
	repositoryRoot, rootError := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(testingT, rootError)
	selection := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", "-z", "--",
		"Makefile", "go.mod", "go.sum", "cmd/configaudit", "internal/serverconfig", "configs", "tests/configs",
		"docker-compose.yml", "docker-compose.computercat.yml")
	selection.Dir = repositoryRoot
	selectedPaths, selectionError := selection.Output()
	require.NoError(testingT, selectionError)
	sourceFiles := fstest.MapFS{}
	for _, sourcePath := range strings.Split(string(selectedPaths), "\x00") {
		if sourcePath == "" {
			continue
		}
		contents, readError := os.ReadFile(filepath.Join(repositoryRoot, sourcePath))
		require.NoError(testingT, readError)
		sourceFiles[sourcePath] = &fstest.MapFile{Data: contents}
	}

	for _, missingProxy := range []bool{false, true} {
		name := "complete audit fixtures"
		if missingProxy {
			name = "missing proxy value is rejected"
		}
		testingT.Run(name, func(testingT *testing.T) {
			fixtureFiles := fstest.MapFS{}
			for name, file := range sourceFiles {
				fixtureFiles[name] = &fstest.MapFile{Data: append([]byte(nil), file.Data...)}
			}
			if missingProxy {
				var compose composeFile
				require.NoError(testingT, yaml.Unmarshal(fixtureFiles[testComposeFileName].Data, &compose))
				auditPath := filepath.ToSlash(filepath.Clean(compose.Services[testLoopAwareService].AuditEnvFile))
				fixtureFiles[auditPath].Data = []byte(strings.Join(filterProxyLines(string(fixtureFiles[auditPath].Data)), "\n"))
			}
			fixtureRoot := testingT.TempDir()
			require.NoError(testingT, os.CopyFS(fixtureRoot, fixtureFiles))
			command := exec.Command("make", "config-audit")
			command.Dir = fixtureRoot
			command.Env = append(os.Environ(), "CI=1", "TRUSTED_PROXY_CIDR=203.0.113.10/32")
			output, commandError := command.CombinedOutput()
			if missingProxy {
				require.Error(testingT, commandError, string(output))
				require.Contains(testingT, string(output), "TRUSTED_PROXY_CIDR is not defined in env")
				return
			}
			require.NoError(testingT, commandError, string(output))
			require.Contains(testingT, string(output), "config-audit OK")
		})
	}
}

func filterProxyLines(contents string) []string {
	var retained []string
	for _, line := range strings.Split(contents, "\n") {
		if !strings.HasPrefix(line, "TRUSTED_PROXY_CIDR=") {
			retained = append(retained, line)
		}
	}
	return retained
}
