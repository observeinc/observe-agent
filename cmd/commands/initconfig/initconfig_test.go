package initconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func Test_InitConfigCommand(t *testing.T) {
	t.Cleanup(func() {
		os.Remove("./test-config.yaml")
	})
	testcases := []struct {
		args           []string
		expectedConfig AgentConfig
		expectErr      string
	}{
		{
			args: []string{"--config_path=./test-config.yaml", "--token=test-token", "--observe_url=test-url"},
			expectedConfig: AgentConfig{
				Token:      "test-token",
				ObserveURL: "test-url",
				SelfMonitoring: SelfMonitoringConfig{
					Enabled: true,
				},
				HostMonitoring: HostMonitoringConfig{
					Enabled: true,
					Logs: HostMonitoringLogsConfig{
						Enabled: true,
					},
					Metrics: HostMonitoringMetricsConfig{
						Enabled: true,
					},
				},
			},
			expectErr: "",
		},
		{
			args: []string{"--config_path=./test-config.yaml", "--token=test-token", "--observe_url=test-url", "--self_monitoring::enabled=false", "--host_monitoring::enabled=false", "--host_monitoring::logs::enabled=false", "--host_monitoring::metrics::enabled=false"},
			expectedConfig: AgentConfig{
				Token:      "test-token",
				ObserveURL: "test-url",
				HostMonitoring: HostMonitoringConfig{
					Enabled: false,
					Logs: HostMonitoringLogsConfig{
						Enabled: false,
					},
					Metrics: HostMonitoringMetricsConfig{
						Enabled: false,
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
		var config AgentConfig
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
