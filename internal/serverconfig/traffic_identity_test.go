package serverconfig_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MarkoPoloResearchLab/loopaware/internal/serverconfig"
	"github.com/stretchr/testify/require"
)

func TestTrafficIdentityRequiresExplicitPrivateIPPolicy(testingT *testing.T) {
	tracked, err := os.ReadFile("../../configs/config.loopaware.yml")
	require.NoError(testingT, err)
	base := strings.ReplaceAll(string(tracked), "  exclude_private_ips: true\n", "")
	for _, scenario := range []struct{ value, errorText string }{
		{"", "analytics.exclude_private_ips"},
		{"  exclude_private_ips: null\n", "analytics.exclude_private_ips"},
		{"  exclude_private_ips: not-a-boolean\n", "bool"},
		{"  exclude_private_ips: true\n", ""},
		{"  exclude_private_ips: false\n", ""},
	} {
		testingT.Run(scenario.value, func(testingT *testing.T) {
			path := filepath.Join(testingT.TempDir(), "config.yml")
			payload := strings.Replace(base, "analytics:\n", "analytics:\n"+scenario.value, 1)
			require.NoError(testingT, os.WriteFile(path, []byte(payload), 0o600))
			config, loadErr := serverconfig.LoadWithLookup(path, func(name string) (string, bool) {
				values := map[string]string{
					"APP_ADDR": ":8080", "PUBLIC_BASE_URL": "https://loopaware.example", "TRUSTED_PROXY_CIDR": "192.168.96.0/20",
					"DB_DRIVER": "sqlite", "DB_DSN": "file::memory:", "SESSION_SECRET": "test-session", "TAUTH_BASE_URL": "https://auth.example",
					"TAUTH_TENANT_ID": "test", "TAUTH_JWT_SIGNING_KEY": "test-signing", "TAUTH_SESSION_COOKIE_NAME": "test-session",
					"PINGUIN_ADDR": "localhost:50051", "PINGUIN_TENANT_API_KEY": "test-key",
				}
				value, exists := values[name]
				return value, exists
			})
			if scenario.errorText != "" {
				require.ErrorContains(testingT, loadErr, scenario.errorText)
			} else {
				require.NoError(testingT, loadErr)
				require.Equal(testingT, strings.Contains(scenario.value, "true"), config.ExcludePrivateVisitIPs)
			}
		})
	}
}
