package initconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

type AgentConfig struct {
	Token          string `yaml:"token"`
	ObserveURL     string `yaml:"observe_url"`
	HostMonitoring struct {
		Enabled bool `yaml:"enabled"`
		Logs    struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"logs"`
		Metrics struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"metrics"`
	} `yaml:"host_monitoring"`
}

func Test_InitConfigCommand(t *testing.T) {
	t.Cleanup(func() {
		os.Remove("./test-config.yaml")
	})
	testcases := []struct {
		args      []string
		expectErr string
	}{
		{
			args:      []string{"--config_path=./test-config.yaml", "--token=test-token", "--observe_url=test-url"},
			expectErr: "",
		},
		{
			args:      []string{"--config_path=./test-config.yaml", "--token=test-token", "--observe_url=test-url", "--host_monitoring.enabled=false", "--host_monitoring.logs.enabled=false", "--host_monitoring.metrics.enabled=false"},
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
		assert.Equal(t, "test-token", config.Token)
		assert.Equal(t, "test-url", config.ObserveURL)
	}
}
