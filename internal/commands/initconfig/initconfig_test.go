package initconfig

import (
	"os"
	"testing"

	"github.com/observeinc/observe-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func Test_InitConfigCommand(t *testing.T) {
	t.Cleanup(func() {
		os.Remove("./test-config.yaml")
	})
	testcases := []struct {
		args           []string
		expectedConfig config.AgentConfig
		expectErr      string
	}{
		{
			args: []string{"--config_path=./test-config.yaml", "--token=test-token", "--observe_url=test-url", "--host_monitoring::logs::include=/test/path,/test/path2"},
			expectedConfig: config.AgentConfig{
				Token:      "test-token",
				ObserveURL: "test-url",
				SelfMonitoring: config.SelfMonitoringConfig{
					Enabled: true,
				},
				HostMonitoring: config.HostMonitoringConfig{
					Enabled: true,
					Logs: config.HostMonitoringLogsConfig{
						Enabled: true,
						Include: []string{"/test/path", "/test/path2"},
					},
					Metrics: config.HostMonitoringMetricsConfig{
						Host: config.HostMonitoringHostMetricsConfig{
							Enabled: true,
						},
						Process: config.HostMonitoringProcessMetricsConfig{
							Enabled: false,
						},
					},
				},
			},
			expectErr: "",
		},
		{
			args: []string{"--config_path=./test-config.yaml", "--token=test-token", "--observe_url=test-url", "--self_monitoring::enabled=false", "--host_monitoring::enabled=false", "--host_monitoring::logs::enabled=false", "--host_monitoring::metrics::host::enabled=false", "--host_monitoring::metrics::process::enabled=false"},
			expectedConfig: config.AgentConfig{
				Token:      "test-token",
				ObserveURL: "test-url",
				HostMonitoring: config.HostMonitoringConfig{
					Enabled: false,
					Logs: config.HostMonitoringLogsConfig{
						Enabled: false,
					},
					Metrics: config.HostMonitoringMetricsConfig{
						Host: config.HostMonitoringHostMetricsConfig{
							Enabled: false,
						},
						Process: config.HostMonitoringProcessMetricsConfig{
							Enabled: false,
						},
					},
				},
			},
			expectErr: "",
		},
	}
	initConfigCmd := NewConfigureCmd()
	RegisterConfigFlags(initConfigCmd)
	for _, tc := range testcases {
		initConfigCmd.SetArgs(tc.args)
		err := initConfigCmd.Execute()
		if err != nil {
			if tc.expectErr == "" {
				t.Errorf("Expected no error, got %v", err)
			}
		}
		var config config.AgentConfig
		configFile, err := os.ReadFile("./test-config.yaml")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		err = yaml.Unmarshal(configFile, &config)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		assert.Equal(t, tc.expectedConfig, config)
	}
}
