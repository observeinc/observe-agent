package initconfig

import (
	"os"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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
	v := viper.New()
	initConfigCmd := NewConfigureCmd(v)
	RegisterConfigFlags(initConfigCmd, v)
	for _, tc := range testcases {
		initConfigCmd.SetArgs(tc.args)
		err := initConfigCmd.Execute()
		if err != nil {
			if tc.expectErr == "" {
				t.Errorf("Expected no error, got %v", err)
			} else {
				assert.ErrorContains(t, err, tc.expectErr)
			}
		}

		var configYaml, configMapstructure config.AgentConfig

		// Decode via mapstructure (which is how viper does it)
		configFileContents, err := os.ReadFile("./test-config.yaml")
		assert.NoError(t, err)
		var yamlMap map[string]any
		err = yaml.Unmarshal(configFileContents, &yamlMap)
		assert.NoError(t, err)
		err = mapstructure.Decode(yamlMap, &configMapstructure)
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedConfig, configMapstructure)

		// Decode via yaml in order to do strict field checks
		configFile, err := os.Open("./test-config.yaml")
		assert.NoError(t, err)
		decoder := yaml.NewDecoder(configFile)
		// Ensure that all fields in the output yaml are present in our struct
		decoder.KnownFields(true)
		err = decoder.Decode(&configYaml)
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedConfig, configYaml)
	}
}
