package main

import (
	"net/netip"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestTrafficIdentityProductionProxyNetwork(testingT *testing.T) {
	payload, err := os.ReadFile("../../.mprlab/deploy/resources.yml")
	require.NoError(testingT, err)
	var document struct {
		Resources struct {
			Targets struct {
				DockerCompose struct {
					Projects []struct {
						Services []struct {
							ID          string
							Environment []struct{ Name, Value string }
						}
					}
				} `yaml:"docker_compose"`
			}
		} `yaml:"mprlab_resources"`
	}
	require.NoError(testingT, yaml.Unmarshal(payload, &document))
	found := false
	for _, project := range document.Resources.Targets.DockerCompose.Projects {
		for _, service := range project.Services {
			if service.ID != "loopaware-api" {
				continue
			}
			for _, variable := range service.Environment {
				if variable.Name != "TRUSTED_PROXY_CIDR" {
					continue
				}
				found = true
				prefix, err := netip.ParsePrefix(variable.Value)
				require.NoError(testingT, err)
				// Observed Caddy peer and owned runtime network on November 19, 2026.
				require.Equal(testingT, "192.168.96.0/20", prefix.String())
				require.True(testingT, prefix.Contains(netip.MustParseAddr("192.168.96.5")))
				require.False(testingT, prefix.Contains(netip.MustParseAddr("192.168.1.2")))
			}
		}
	}
	require.True(testingT, found)
}
